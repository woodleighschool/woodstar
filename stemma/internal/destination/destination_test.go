package destination

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/woodleighschool/stemma/plugin"
)

func TestCompiledPluginReconcilesContentAndPresence(t *testing.T) {
	binary := buildPlugin(t)
	state, connection := serveFixture(t)
	request := plugin.ReconcileRequest{Method: "plan", Identity: plugin.Identity{Project: "fixture", Software: "Example", Destination: "woodstar"}, Config: connection, Artifact: installerFixture(t, "Example.pkg", "synthetic installer bytes; never executed", "1.0")}
	setPkginfo(t, &request, `{"name":"Example App","version":"1.0","description":"Managed description","unattended_install":true,"blocking_applications":[],"receipts":[{"packageid":"test.example","version":"1.0"}],"installs":[{"type":"application","path":"/Applications/Example.app","CFBundleIdentifier":"test.example","CFBundleShortVersionString":"1.0"}]}`)
	described, err := plugin.Run(t.Context(), binary, plugin.Request{Method: "describe"})
	if err != nil {
		t.Fatal(err)
	}
	var descriptor plugin.Descriptor
	if err := json.Unmarshal(described.Output, &descriptor); err != nil || len(descriptor.Operations) != 1 || descriptor.Operations[0].Name != "woodstar.munki" {
		t.Fatalf("descriptor=%+v err=%v", descriptor, err)
	}
	if !t.Run("create and converge", func(t *testing.T) {
		checkPluginCreation(t, binary, state, &request)
	}) {
		return
	}
	t.Run("rejected declaration writes nothing", func(t *testing.T) {
		invalid := request
		setPkginfo(t, &invalid, `{"name":"Example App","version":"1.0","uninstallable":true,"uninstall_method":"removepackages","receipts":[]}`)
		before := state.writes()
		for _, method := range []string{"plan", "apply"} {
			invalid.Method = method
			if _, err := runPlugin(t, binary, invalid); err == nil || state.writes() != before {
				t.Fatalf("%s error=%v writes=%d", method, err, state.writes())
			}
		}
	})
	if !t.Run("metadata ownership", func(t *testing.T) {
		checkPluginMetadata(t, binary, state, &request)
	}) {
		return
	}
	t.Run("response loss and digest verification", func(t *testing.T) {
		checkPluginRecovery(t, binary, state, &request)
	})
}

func TestApplyAdoptsAnExistingPublication(t *testing.T) {
	fixture, connection := serveFixture(t)
	imported := "installer another tool imported"
	fixture.software = map[string]any{"id": json.Number("1"), "name": "Example", "description": "Imported by hand", "category": "Utilities", "targets": map[string]any{"include": []any{}, "exclude": []any{}}}
	fixture.objects[40], fixture.names[40] = []byte(imported), "Example-1.0.pkg"
	fixture.setPackage(map[string]any{"id": json.Number("5"), "version": "1.0", "installer_type": "pkg", "notes": "Imported by hand", "installer_object_id": json.Number("40")})
	request := plugin.ReconcileRequest{
		Method: "plan", Prepared: true, Config: connection, Identity: plugin.Identity{Software: "Example"},
		Metadata: json.RawMessage(`{"pkginfo":{"description":"Managed description"}}`),
		Artifact: installerFixture(t, "Example.pkg", imported, "1.0"),
	}
	// A name and version the repository already holds are the publication itself,
	// so what differs is ordinary drift.
	planned, err := Handle(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if got := changedFields(planned.Changes); !slices.Equal(got, []string{"software.description"}) || fixture.writes() != 0 {
		t.Fatalf("adoption plan=%v writes=%d", got, fixture.writes())
	}
	request.Method = "apply"
	if _, err := Handle(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	if title := fixture.title(); title["description"] != "Managed description" || title["category"] != "Utilities" || fixture.version("1.0")["notes"] != "Imported by hand" || fixture.uploadCount() != 0 {
		t.Fatalf("adopted software=%v package=%v uploads=%d", title, fixture.version("1.0"), fixture.uploadCount())
	}
	assertFixtureConverged(t, fixture, request)

	// The same version with other bytes is drift too, replaced where it lives.
	before := request.Artifact.SHA256
	request.Artifact = installerFixture(t, "Example.pkg", "rebuilt installer", "1.0")
	request.Method = "plan"
	if planned, err = Handle(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	want := plugin.Change{Kind: "content", Field: "package.installer", Action: "upload", Before: raw(before), After: raw(request.Artifact.SHA256)}
	if len(planned.Changes) != 1 || !reflect.DeepEqual(planned.Changes[0], want) {
		t.Fatalf("changed bytes plan=%+v", planned.Changes)
	}
	request.Method = "apply"
	if _, err := Handle(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	installer, _ := fixture.version("1.0")["installer_file"].(map[string]any)
	if installer["sha256"] != request.Artifact.SHA256 || number(fixture.version("1.0")["id"]) != 5 || fixture.uploadCount() != 1 || fixture.softwareCount()+fixture.packageCount() != 0 {
		t.Fatalf("installer=%v package=%v uploads=%d", installer, fixture.version("1.0"), fixture.uploadCount())
	}
	assertFixtureConverged(t, fixture, request)
}

func TestLostCreateRepliesAreFoundByNativeIdentity(t *testing.T) {
	fixture, connection := serveFixture(t)
	fixture.dropSoftwareReply, fixture.dropPackageReply = true, true
	request := plugin.ReconcileRequest{Method: "apply", Prepared: true, Config: connection, Identity: plugin.Identity{Software: "Example"}, Artifact: installerFixture(t, "Example.pkg", "synthetic installer bytes", "1.0")}
	if _, err := Handle(t.Context(), request); err != nil {
		t.Fatalf("committed creates lost their replies: %v", err)
	}
	if fixture.softwareCount() != 1 || fixture.packageCount() != 1 || fixture.uploadCount() != 1 {
		t.Fatalf("software=%d packages=%d uploads=%d", fixture.softwareCount(), fixture.packageCount(), fixture.uploadCount())
	}
	assertFixtureConverged(t, fixture, request)
}

func TestRefusedPackageReleasesItsUploadAndTheNextRunConverges(t *testing.T) {
	fixture, connection := serveFixture(t)
	fixture.failPackageSave = true
	request := plugin.ReconcileRequest{Method: "apply", Prepared: true, Config: connection, Identity: plugin.Identity{Software: "Example"}, Artifact: installerFixture(t, "Example.pkg", "synthetic installer bytes", "1.0")}
	if _, err := Handle(t.Context(), request); err == nil {
		t.Fatal("reported a publication whose package was refused")
	}
	// The refusal leaves a title without a package. Nothing references the
	// uploaded installer, so the run releases it.
	if !fixture.wasReleased(1) || fixture.softwareCount() != 1 || fixture.packageCount() != 0 {
		t.Fatalf("released=%t software=%d packages=%d", fixture.wasReleased(1), fixture.softwareCount(), fixture.packageCount())
	}
	fixture.mu.Lock()
	fixture.failPackageSave = false
	fixture.mu.Unlock()
	if _, err := Handle(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	if fixture.softwareCount() != 1 || fixture.packageCount() != 1 || fixture.uploadCount() != 2 || fixture.wasReleased(2) {
		t.Fatalf("software=%d packages=%d uploads=%d released=%t", fixture.softwareCount(), fixture.packageCount(), fixture.uploadCount(), fixture.wasReleased(2))
	}
	assertFixtureConverged(t, fixture, request)
}

func setPkginfo(t *testing.T, request *plugin.ReconcileRequest, document string) {
	t.Helper()
	fields, err := object(request.Metadata)
	if err != nil {
		t.Fatal(err)
	}
	fields["pkginfo"] = json.RawMessage(document)
	request.Metadata = raw(fields)
	request.Prepared = true
}

func runPlugin(t *testing.T, binary string, request plugin.ReconcileRequest) (plugin.ReconcileResponse, error) {
	t.Helper()
	response, err := plugin.Run(t.Context(), binary, plugin.Request{Operation: "woodstar.munki", Method: request.Method, Input: raw(request)})
	var result plugin.ReconcileResponse
	if len(response.Output) > 0 {
		if decodeErr := json.Unmarshal(response.Output, &result); decodeErr != nil {
			t.Fatal(decodeErr)
		}
	}
	return result, err
}

// assertFixtureConverged replans and reapplies an unchanged declaration. Nothing
// passes between runs, so each must find the repository already as declared.
func assertFixtureConverged(t *testing.T, fixture *apiFixture, request plugin.ReconcileRequest) {
	t.Helper()
	before := fixture.writes()
	for _, method := range []string{"plan", "apply"} {
		request.Method = method
		response, err := Handle(t.Context(), request)
		if err != nil {
			t.Fatal(err)
		}
		if len(response.Changes) != 0 || fixture.writes() != before {
			t.Fatalf("unchanged %s: changes=%+v writes=%d, want %d", method, response.Changes, fixture.writes(), before)
		}
	}
}

func changedFields(changes []plugin.Change) []string {
	fields := make([]string, 0, len(changes))
	for _, change := range changes {
		fields = append(fields, change.Field)
	}
	slices.Sort(fields)
	return fields
}

// apiFixture is one software title's repository. It supplies deterministic
// faults to the compiled executable and to in-process runs. The adapter also
// runs against the real API in the PostgreSQL tests.
type apiFixture struct {
	mu                                                 sync.Mutex
	origin                                             string
	software, pkg                                      map[string]any
	packages                                           map[int64]map[string]any
	objects                                            map[int64][]byte
	names                                              map[int64]string
	labels                                             map[string]int64
	held                                               map[int64]bool // Packages another title references.
	released                                           []int64        // Installer objects deleted on request.
	createdSoftware, createdPackages, updates, uploads int
	mutations, lastPackage                             int
	dropSoftwareReply, dropPackageReply                bool
	failPackageSave, forgeDigest                       bool
	dropIconReply                                      bool
	failIconAttach                                     bool
}

// serveFixture starts the fake API and returns its connection settings.
func serveFixture(t *testing.T) (*apiFixture, json.RawMessage) {
	t.Helper()
	fixture := &apiFixture{objects: map[int64][]byte{}, names: map[int64]string{}}
	connection, origin := serveAPI(t, fixture)
	fixture.origin = origin
	return fixture, connection
}

// serveAPI starts a TLS server for handler and returns connection settings that
// trust it, with the server's origin.
func serveAPI(t *testing.T, handler http.Handler) (json.RawMessage, string) {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)
	caPath := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0o600); err != nil {
		t.Fatal(err)
	}
	return raw(Config{URL: server.URL, APIKey: "synthetic-key", CAFile: caPath}), server.URL
}

// writes counts every request that could change the repository, refused or not.
func (fixture *apiFixture) writes() int {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	return fixture.mutations
}

func (fixture *apiFixture) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	write := func(value any) {
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(value)
	}
	read := func() map[string]any {
		var body map[string]any
		decoder := json.NewDecoder(request.Body)
		decoder.UseNumber()
		_ = decoder.Decode(&body)
		return body
	}
	if request.Method != http.MethodGet {
		fixture.mutations++
	}
	if strings.HasPrefix(request.URL.Path, "/transfer/") {
		fixture.serveTransfer(response, request)
		return
	}

	if request.Header.Get("Authorization") != "Bearer synthetic-key" {
		http.Error(response, "unauthorized", http.StatusUnauthorized)
		return
	}
	if fixture.failIconAttach && request.Method == http.MethodPut && request.URL.Path == "/api/munki/software/1/icon" {
		http.Error(response, "attach unavailable", http.StatusServiceUnavailable)
		return
	}
	if request.URL.Path == "/api/labels" && request.Method == http.MethodGet {
		items := []any{}
		for _, name := range slices.Sorted(maps.Keys(fixture.labels)) {
			if matches(name, request.URL.Query().Get("q")) {
				items = append(items, map[string]any{"id": fixture.labels[name], "name": name})
			}
		}
		write(map[string]any{"items": items, "count": len(items)})
		return
	}
	if strings.HasPrefix(request.URL.Path, "/api/munki/software") {
		fixture.serveSoftware(response, request, read, write)
		return
	}
	if strings.HasPrefix(request.URL.Path, "/api/munki/package-installers") || request.URL.Path == "/api/munki/icons" {
		fixture.serveUpload(response, request, read, write)
		return
	}
	fixture.servePackages(response, request, read, write)
}

func (fixture *apiFixture) setPackage(body map[string]any) {
	if number(body["id"]) == 0 {
		// Ids and creation times ascend together, as the repository assigns them.
		for id := range fixture.packages {
			fixture.lastPackage = max(fixture.lastPackage, int(id))
		}
		fixture.lastPackage++
		body["id"] = json.Number(strconv.Itoa(fixture.lastPackage))
		body["created_at"] = time.Date(2026, time.June, 1, 9, fixture.lastPackage, 0, 0, time.UTC).Format(time.RFC3339)
	}
	body["software"] = map[string]any{"id": json.Number("1"), "name": fixture.software["name"]}
	delete(body, "software_id")
	if id := number(body["installer_object_id"]); id != 0 {
		content := fixture.objects[id]
		digest := sha256.Sum256(content)
		body["installer_file"] = map[string]any{"filename": fixture.names[id], "size_bytes": len(content), "sha256": hex.EncodeToString(digest[:])}
	}
	fixture.pkg = body
	if fixture.packages == nil {
		fixture.packages = map[int64]map[string]any{}
	}
	fixture.packages[number(body["id"])] = body
}

func (fixture *apiFixture) uploadCount() int {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	return fixture.uploads
}

func (fixture *apiFixture) packageCount() int {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	return fixture.createdPackages
}

func (fixture *apiFixture) softwareCount() int {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	return fixture.createdSoftware
}

// title returns a copy of the software the repository holds.
func (fixture *apiFixture) title() map[string]any {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	return maps.Clone(fixture.software)
}

func (fixture *apiFixture) wasReleased(object int64) bool {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	return slices.Contains(fixture.released, object)
}

// version returns the package the repository holds for a version, if any.
func (fixture *apiFixture) version(version string) map[string]any {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	for _, item := range fixture.packages {
		if item["version"] == version {
			return item
		}
	}
	return nil
}

func (fixture *apiFixture) versions() []string {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	versions := make([]string, 0, len(fixture.packages))
	for _, item := range fixture.packages {
		version, _ := item["version"].(string)
		versions = append(versions, version)
	}
	slices.Sort(versions)
	return versions
}

func mergeFixture(current, fields map[string]any) map[string]any {
	result := maps.Clone(current)
	if result == nil {
		result = map[string]any{}
	}
	for key, value := range fields {
		if nested, ok := value.(map[string]any); ok {
			prior, _ := result[key].(map[string]any)
			value = mergeFixture(prior, nested)
		}
		if value == nil && key == "description" {
			value = ""
		}
		result[key] = value
	}
	return result
}

// matches mirrors the repository's list search, a case-insensitive substring,
// so one query can return more than the exact name.
func matches(value any, query string) bool {
	text, _ := value.(string)
	return strings.Contains(strings.ToLower(text), strings.ToLower(query))
}

func number(value any) int64 {
	switch value := value.(type) {
	case json.Number:
		n, _ := value.Int64()
		return n
	case int:
		return int64(value)
	case int64:
		return value
	case float64:
		return int64(value)
	default:
		return 0
	}
}

// dropReply ends the connection after the repository committed the request.
func dropReply(response http.ResponseWriter) {
	if connection, _, err := http.NewResponseController(response).Hijack(); err == nil {
		_ = connection.Close()
	}
}

func buildPlugin(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "plugin")
	// #nosec G204 -- Builds a fixed package into a test-owned temporary path.
	command := exec.CommandContext(t.Context(), "go", "build", "-o", binary, "../../cmd/plugin")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build plugin: %v\n%s", err, output)
	}
	return binary
}

func (fixture *apiFixture) serveSoftware(response http.ResponseWriter, request *http.Request, read func() map[string]any, write func(any)) {
	switch request.URL.Path {
	case "/api/munki/software":
		if request.Method == http.MethodGet {
			items := []any{}
			if fixture.software != nil && matches(fixture.software["name"], request.URL.Query().Get("q")) {
				items = append(items, fixture.software)
			}
			write(map[string]any{"items": items, "count": len(items)})
			return
		}
		if request.Method == http.MethodPost {
			if fixture.software != nil {
				http.Error(response, "conflict", http.StatusConflict)
				return
			}
			fixture.software = read()
			fixture.software["id"] = json.Number("1")
			fixture.createdSoftware++
			if fixture.dropSoftwareReply {
				fixture.dropSoftwareReply = false
				dropReply(response)
				return
			}
			write(fixture.software)
			return
		}
	case "/api/munki/software/1/icon":
		fixture.serveIcon(response, request, read, write)
		return
	case "/api/munki/software/1":
		if fixture.software == nil {
			http.NotFound(response, request)
			return
		}
		if request.Method == http.MethodPatch {
			body := read()
			fixture.software = mergeFixture(fixture.software, body)
			if value, exists := body["icon_object_id"]; exists && value == nil {
				delete(fixture.software, "icon_file")
				delete(fixture.software, "icon_object_id")
			}
			fixture.updates++
		}
		write(fixture.software)
		return
	}
	http.NotFound(response, request)
}

func (fixture *apiFixture) serveIcon(response http.ResponseWriter, request *http.Request, read func() map[string]any, write func(any)) {
	if request.Method != http.MethodPut || fixture.software == nil {
		http.NotFound(response, request)
		return
	}
	body := read()
	id := number(body["object_id"])
	content, exists := fixture.objects[id]
	if !exists {
		http.Error(response, "missing bytes", http.StatusBadRequest)
		return
	}
	digest := sha256.Sum256(content)
	hash := hex.EncodeToString(digest[:])
	fixture.software["icon_object_id"] = json.Number(strconv.FormatInt(id, 10))
	fixture.software["icon_file"] = map[string]any{"filename": fixture.names[id], "sha256": hash, "size_bytes": len(content)}
	if fixture.dropIconReply {
		http.Error(response, "lost reply", http.StatusInternalServerError)
		return
	}
	write(map[string]any{"id": id, "sha256": hash, "size_bytes": len(content)})
}

func fixtureIncludes(fields map[string]any) []any {
	targets, _ := fields["targets"].(map[string]any)
	include, _ := targets["include"].([]any)
	return include
}

func (fixture *apiFixture) serveTransfer(response http.ResponseWriter, request *http.Request) {
	if request.Header.Get("Authorization") != "" {
		http.Error(response, "credential leaked to transfer", http.StatusBadRequest)
		return
	}
	if request.Method != http.MethodPut {
		http.Error(response, "method", http.StatusMethodNotAllowed)
		return
	}
	id, _ := strconv.ParseInt(strings.TrimPrefix(request.URL.Path, "/transfer/"), 10, 64)
	fixture.objects[id], _ = io.ReadAll(request.Body)
	response.WriteHeader(http.StatusNoContent)
}

func checkPluginCreation(t *testing.T, binary string, state *apiFixture, request *plugin.ReconcileRequest) {
	t.Helper()
	plan, err := runPlugin(t, binary, *request)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Changes) == 0 || state.writes() != 0 {
		t.Fatalf("plan changes=%v writes=%d", plan.Changes, state.writes())
	}
	request.Method = "apply"
	if _, err := runPlugin(t, binary, *request); err != nil {
		t.Fatal(err)
	}
	if state.uploadCount() != 1 {
		t.Fatalf("uploads=%d", state.uploadCount())
	}
	// Each run is a new process holding nothing from the last, so repeating the
	// request is the whole convergence check.
	initialWrites := state.writes()
	for _, method := range []string{"plan", "apply"} {
		request.Method = method
		second, err := runPlugin(t, binary, *request)
		if err != nil {
			t.Fatal(err)
		}
		if len(second.Changes) != 0 || state.writes() != initialWrites {
			t.Fatalf("unchanged %s wrote: changes=%+v writes=%d", method, second.Changes, state.writes())
		}
	}
}

func checkPluginMetadata(t *testing.T, binary string, state *apiFixture, request *plugin.ReconcileRequest) {
	t.Helper()
	state.mu.Lock()
	state.software["description"] = "Manual description"
	state.software["category"] = "Manual category"
	state.software["icon_object_id"] = json.Number("90")
	state.software["targets"] = map[string]any{"include": []any{map[string]any{"label_id": 1, "package": map[string]any{"strategy": "latest"}, "actions": []any{"optional_installs"}}}, "exclude": []any{}}
	state.pkg["notes"] = "Manual package note"
	state.mu.Unlock()
	setPkginfo(t, request, `{"name":"Example App","version":"1.0","developer":"Managed developer","unattended_install":false}`)
	third, err := runPlugin(t, binary, *request)
	if err != nil {
		t.Fatal(err)
	}
	// Derivation owns a PKG's detection and removal fields. No run remembers that
	// an earlier document set them, so without evidence they are cleared.
	want := []string{"package.installs", "package.receipts", "package.unattended_install", "package.uninstall_method", "package.uninstallable", "software.developer"}
	if got := changedFields(third.Changes); !slices.Equal(got, want) {
		t.Fatalf("metadata changes=%v, want %v", got, want)
	}
	state.mu.Lock()
	if state.software["description"] != "Manual description" || state.software["category"] != "Manual category" || number(state.software["icon_object_id"]) != 90 || state.pkg["notes"] != "Manual package note" || state.pkg["unattended_install"] != false {
		t.Errorf("omitted data changed: software=%v package=%v", state.software, state.pkg)
	}
	if len(fixtureIncludes(state.software)) != 1 {
		t.Error("omitted targets were cleared")
	}
	receipts, _ := state.pkg["receipts"].([]any)
	installs, _ := state.pkg["installs"].([]any)
	if len(receipts)+len(installs) != 0 || state.pkg["uninstallable"] != false || state.pkg["uninstall_method"] != nil {
		t.Errorf("derivation-owned fields outlived their declaration: %v", state.pkg)
	}
	state.mu.Unlock()
	if state.uploadCount() != 1 {
		t.Fatal("metadata edit reuploaded installer")
	}
	request.Metadata = json.RawMessage(`{"targets":{"include":[],"exclude":[]}}`)
	setPkginfo(t, request, `{"name":"Example App","version":"1.0","description":null}`)
	if _, err := runPlugin(t, binary, *request); err != nil {
		t.Fatal(err)
	}
	before := state.writes()
	if rerun, err := runPlugin(t, binary, *request); err != nil || len(rerun.Changes) != 0 || state.writes() != before {
		t.Fatalf("rerun changes=%+v error=%v writes=%d", rerun.Changes, err, state.writes())
	}
	state.mu.Lock()
	if state.software["description"] != "" || len(fixtureIncludes(state.software)) != 0 {
		t.Errorf("explicit clears were not applied: %v", state.software)
	}
	if state.createdSoftware != 1 || state.createdPackages != 1 {
		t.Errorf("reruns duplicated resources: software=%d package=%d", state.createdSoftware, state.createdPackages)
	}
	state.mu.Unlock()
}

func checkPluginRecovery(t *testing.T, binary string, state *apiFixture, request *plugin.ReconcileRequest) {
	t.Helper()
	changeInstaller(t, request, "version two installer")
	setPkginfo(t, request, `{"name":"Example App","version":"2.0"}`)
	state.mu.Lock()
	state.dropPackageReply = true
	state.mu.Unlock()
	if _, err := runPlugin(t, binary, *request); err != nil {
		t.Fatalf("recover committed package response loss: %v", err)
	}
	if rerun, err := runPlugin(t, binary, *request); err != nil || len(rerun.Changes) != 0 || state.packageCount() != 2 {
		t.Fatalf("ambiguous create: changes=%+v error=%v packages=%d", rerun.Changes, err, state.packageCount())
	}
	state.mu.Lock()
	state.forgeDigest = true
	state.mu.Unlock()
	changeInstaller(t, request, "version three installer")
	setPkginfo(t, request, `{"name":"Example App","version":"3.0"}`)
	if _, err := runPlugin(t, binary, *request); err == nil {
		t.Fatal("accepted incorrect finalized digest")
	}
	if state.packageCount() != 2 {
		t.Fatal("published metadata before content identity was verified")
	}
	// A later run uploads afresh, so the reservation that failed is not kept.
	if reserved := int64(state.uploadCount()); !state.wasReleased(reserved) {
		t.Fatalf("failed upload kept its reserved object %d", reserved)
	}
}

func (fixture *apiFixture) serveUpload(response http.ResponseWriter, request *http.Request, read func() map[string]any, write func(any)) {
	object, _ := strconv.ParseInt(strings.TrimPrefix(request.URL.Path, "/api/munki/package-installers/"), 10, 64)
	switch {
	case (request.URL.Path == "/api/munki/package-installers" || request.URL.Path == "/api/munki/icons") && request.Method == http.MethodPost:
		body := read()
		fixture.uploads++
		id := int64(fixture.uploads)
		fixture.names[id], _ = body["filename"].(string)
		write(map[string]any{"object_id": id, "upload": map[string]any{"strategy": "direct-put", "target": map[string]any{"url": fixture.origin + "/transfer/" + strconv.FormatInt(id, 10), "method": "PUT", "headers": map[string]string{"Content-Type": "application/octet-stream"}}}})
		return
	case object != 0 && request.Method == http.MethodPut:
		content := fixture.objects[object]
		digest := sha256.Sum256(content)
		hash := hex.EncodeToString(digest[:])
		if fixture.forgeDigest {
			hash = strings.Repeat("0", 64)
		}
		write(map[string]any{"id": object, "sha256": hash, "size_bytes": len(content)})
		return
	case object != 0 && request.Method == http.MethodDelete:
		// The repository keeps an installer that a package references.
		for _, item := range fixture.packages {
			if number(item["installer_object_id"]) == object {
				http.Error(response, "conflict", http.StatusConflict)
				return
			}
		}
		delete(fixture.objects, object)
		delete(fixture.names, object)
		fixture.released = append(fixture.released, object)
		response.WriteHeader(http.StatusNoContent)
		return
	}
	http.NotFound(response, request)
}

func (fixture *apiFixture) servePackages(response http.ResponseWriter, request *http.Request, read func() map[string]any, write func(any)) {
	if fixture.failPackageSave && (request.Method == http.MethodPost || request.Method == http.MethodPatch) {
		http.Error(response, "save unavailable", http.StatusServiceUnavailable)
		return
	}
	switch {
	case request.URL.Path == "/api/munki/packages":
		if request.Method == http.MethodGet {
			if request.URL.Query().Get("software_id") != "1" {
				http.Error(response, "unscoped package read", http.StatusBadRequest)
				return
			}
			items := fixture.packageList(request.URL.Query().Get("q"))
			write(map[string]any{"items": items, "count": len(items)})
			return
		}
		if request.Method == http.MethodDelete {
			id, _ := strconv.ParseInt(request.URL.Query().Get("ids"), 10, 64)
			// Foreign keys refuse a package that another title still references.
			if fixture.held[id] {
				http.Error(response, "conflict", http.StatusConflict)
				return
			}
			delete(fixture.packages, id)
			fixture.updates++
			response.WriteHeader(http.StatusNoContent)
			return
		}
		if request.Method == http.MethodPost {
			body := read()
			for _, item := range fixture.packages {
				if item["version"] == body["version"] {
					http.Error(response, "conflict", http.StatusConflict)
					return
				}
			}
			fixture.createdPackages++
			fixture.setPackage(body)
			if fixture.dropPackageReply {
				fixture.dropPackageReply = false
				dropReply(response)
				return
			}
			write(fixture.pkg)
			return
		}
	case strings.HasPrefix(request.URL.Path, "/api/munki/packages/"):
		id, _ := strconv.ParseInt(strings.TrimPrefix(request.URL.Path, "/api/munki/packages/"), 10, 64)
		fixture.pkg = fixture.packages[id]
		if fixture.pkg == nil {
			http.NotFound(response, request)
			return
		}
		if request.Method == http.MethodPatch {
			fixture.setPackage(mergeFixture(fixture.pkg, read()))
			fixture.updates++
		}
		write(fixture.pkg)
		return
	}
	http.NotFound(response, request)
}

// installerFixture leases synthetic installer bytes; the extension is the format.
func installerFixture(t *testing.T, filename, body, version string) plugin.Artifact {
	t.Helper()
	path := filepath.Join(t.TempDir(), filename)
	if err := os.WriteFile(path, []byte(body), 0o400); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256([]byte(body))
	return plugin.Artifact{Path: path, Filename: filename, Format: strings.TrimPrefix(filepath.Ext(filename), "."), Version: version, Size: int64(len(body)), SHA256: hex.EncodeToString(hash[:])}
}

func changeInstaller(t *testing.T, request *plugin.ReconcileRequest, body string) {
	t.Helper()
	request.Artifact = installerFixture(t, "Example.pkg", body, "")
}

func (fixture *apiFixture) packageList(query string) []any {
	items := []any{}
	for _, item := range fixture.packages {
		if matches(item["version"], query) {
			items = append(items, item)
		}
	}
	return items
}
