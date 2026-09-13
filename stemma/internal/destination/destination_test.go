package destination

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/woodleighschool/stemma/plugin"
)

func TestCompiledPluginReconcilesContentAndPresence(t *testing.T) {
	binary := buildPlugin(t)
	state := &apiFixture{objects: map[int64][]byte{}, names: map[int64]string{}}
	server := httptest.NewTLSServer(state)
	t.Cleanup(server.Close)
	state.origin = server.URL
	caPath := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0o600); err != nil {
		t.Fatal(err)
	}
	data := []byte("synthetic installer bytes; never executed")
	artifactPath := filepath.Join(t.TempDir(), "Example.pkg")
	if err := os.WriteFile(artifactPath, data, 0o400); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	request := plugin.ReconcileRequest{Method: "plan", Identity: plugin.Identity{Project: "fixture", Software: "Example", Destination: "woodstar"}, Config: raw(map[string]any{"url": server.URL, "api_key": "synthetic-key", "ca_file": caPath}), Inputs: map[string]plugin.Artifact{"installer": {Path: artifactPath, Filename: "Example.pkg", Version: "1.0", Size: int64(len(data)), SHA256: hex.EncodeToString(digest[:])}}}
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
	t.Run("failed plan retains recovered binding", func(t *testing.T) {
		invalid := request
		invalid.Method, invalid.Binding = "plan", request.Binding
		setPkginfo(t, &invalid, `{"name":"Example App","version":"1.0","uninstallable":true,"uninstall_method":"removepackages","receipts":[]}`)
		before := state.writes()
		result, err := runPlugin(t, binary, invalid)
		if err == nil || len(result.Binding) == 0 || state.writes() != before {
			t.Fatalf("binding=%s error=%v writes=%d", result.Binding, err, state.writes())
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

func TestValidationDoesNotContactDestination(t *testing.T) {
	request := plugin.ReconcileRequest{Method: "validate", Config: raw(map[string]any{"url": "https://woodstar.test", "api_key": "synthetic-key"})}
	setPkginfo(t, &request, `{"name":"Example","version":"1.0","installer_type":"nopkg"}`)
	if _, err := Handle(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	for _, metadata := range []string{`null`, `{"software":{}}`, `{"package":{}}`, `{"targets":null}`, `{"targets":{"include":null}}`, `{"targets":{"unknown":true}}`} {
		t.Run(metadata, func(t *testing.T) {
			request.Metadata = json.RawMessage(metadata)
			if _, err := Handle(t.Context(), request); err == nil {
				t.Fatal("accepted invalid controls")
			}
		})
	}
}

func setPkginfo(t *testing.T, request *plugin.ReconcileRequest, document string) {
	t.Helper()
	data := []byte(document)
	path := filepath.Join(t.TempDir(), "pkginfo.json")
	if err := os.WriteFile(path, data, 0o400); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	request.Artifact = plugin.Artifact{Path: path, Filename: "pkginfo.json", Format: "json", Size: int64(len(data)), SHA256: hex.EncodeToString(digest[:])}
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

// apiFixture supplies deterministic transfer faults to the compiled executable.
// The adapter also runs against the real API in the PostgreSQL tests.
type apiFixture struct {
	mu                                                 sync.Mutex
	origin                                             string
	software, pkg                                      map[string]any
	packages                                           map[int64]map[string]any
	objects                                            map[int64][]byte
	names                                              map[int64]string
	createdSoftware, createdPackages, updates, uploads int
	dropPackageReply, forgeDigest                      bool
	dropIconReply                                      bool
	failIconAttach                                     bool
}

func (fixture *apiFixture) writes() int {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	return fixture.createdSoftware + fixture.createdPackages + fixture.updates + fixture.uploads
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
		body["id"] = json.Number(strconv.Itoa(fixture.createdPackages))
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
			if fixture.software != nil {
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
			write(fixture.software)
			return
		}
	case "/api/munki/software/1/icon":
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
	first, err := runPlugin(t, binary, *request)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Binding) == 0 || state.uploadCount() != 1 {
		t.Fatalf("binding=%s uploads=%d", first.Binding, state.uploadCount())
	}
	initialWrites := state.writes()
	request.Binding = first.Binding
	second, err := runPlugin(t, binary, *request)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Changes) != 0 || state.writes() != initialWrites {
		t.Fatalf("unchanged run wrote: changes=%+v writes=%d", second.Changes, state.writes())
	}
}

func checkPluginMetadata(t *testing.T, binary string, state *apiFixture, request *plugin.ReconcileRequest) {
	t.Helper()
	state.mu.Lock()
	state.software["description"] = "Manual description"
	state.software["category"] = "Manual category"
	state.software["icon_object_id"] = json.Number("90")
	state.software["targets"] = map[string]any{"include": []any{map[string]any{"label_id": 1, "package": map[string]any{"strategy": "latest"}, "actions": []any{"optional_installs"}}}, "exclude": []any{}}
	state.pkg["unattended_install"] = true
	state.pkg["notes"] = "Manual package note"
	state.mu.Unlock()
	setPkginfo(t, request, `{"name":"Example App","version":"1.0","developer":"Managed developer","unattended_install":false}`)
	third, err := runPlugin(t, binary, *request)
	if err != nil {
		t.Fatal(err)
	}
	if len(third.Changes) != 2 {
		t.Fatalf("metadata changes=%+v", third.Changes)
	}
	state.mu.Lock()
	if state.software["description"] != "Manual description" || state.software["category"] != "Manual category" || number(state.software["icon_object_id"]) != 90 || state.pkg["notes"] != "Manual package note" || state.pkg["unattended_install"] != false {
		t.Errorf("unmanaged data changed: software=%v package=%v", state.software, state.pkg)
	}
	if len(fixtureIncludes(state.software)) != 1 {
		t.Error("omitted targets were cleared")
	}
	state.mu.Unlock()
	if state.uploadCount() != 1 {
		t.Fatal("metadata edit reuploaded installer")
	}
	request.Binding = third.Binding
	setPkginfo(t, request, `{"name":"Example App","version":"1.0","description":null}`)
	request.Metadata = json.RawMessage(`{"targets":{"include":[]}}`)
	cleared, err := runPlugin(t, binary, *request)
	if err != nil {
		t.Fatal(err)
	}
	request.Binding = cleared.Binding
	lost := *request
	lost.Binding = nil
	before := state.writes()
	if _, err := runPlugin(t, binary, lost); err == nil || state.writes() != before {
		t.Fatalf("lost binding error=%v writes=%d", err, state.writes())
	}
	state.mu.Lock()
	if state.software["description"] != "" || len(fixtureIncludes(state.software)) != 0 {
		t.Errorf("explicit clears were not applied: %v", state.software)
	}
	if state.createdSoftware != 1 || state.createdPackages != 1 {
		t.Errorf("lost binding duplicated resources: software=%d package=%d", state.createdSoftware, state.createdPackages)
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
	recovered, err := runPlugin(t, binary, *request)
	if err != nil {
		t.Fatalf("recover committed package response loss: %v", err)
	}
	request.Binding = recovered.Binding
	if state.packageCount() != 2 {
		t.Fatalf("ambiguous create duplicated package: %d", state.packageCount())
	}
	state.mu.Lock()
	state.forgeDigest = true
	state.mu.Unlock()
	changeInstaller(t, request, "version three installer")
	setPkginfo(t, request, `{"name":"Example App","version":"3.0"}`)
	failed, err := runPlugin(t, binary, *request)
	if err == nil {
		t.Fatal("accepted incorrect finalized digest")
	}
	if len(failed.Binding) == 0 {
		t.Fatal("failed upload lost the recovered software binding")
	}
	if state.packageCount() != 2 {
		t.Fatal("published metadata before content identity was verified")
	}
}

func TestValidationVerifiesBothLeasedArtifacts(t *testing.T) {
	request := plugin.ReconcileRequest{Method: "validate", Config: raw(map[string]any{"url": "https://woodstar.test", "api_key": "synthetic-key"})}
	setPkginfo(t, &request, "synthetic installer bytes")
	installer := request.Artifact
	installer.Filename, installer.Format = "Example.pkg", "pkg"
	request.Inputs = map[string]plugin.Artifact{"installer": installer}
	setPkginfo(t, &request, `{"name":"Example","version":"1.0"}`)
	if _, err := Handle(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	t.Run("pkginfo digest", func(t *testing.T) {
		changed := request
		changed.Artifact.SHA256 = strings.Repeat("0", 64)
		if _, err := Handle(t.Context(), changed); err == nil || !strings.Contains(err.Error(), "digest changed") {
			t.Fatalf("pkginfo verification error=%v", err)
		}
	})
	t.Run("installer digest", func(t *testing.T) {
		changed := request
		changedInstaller := installer
		changedInstaller.SHA256 = strings.Repeat("0", 64)
		changed.Inputs = map[string]plugin.Artifact{"installer": changedInstaller}
		if _, err := Handle(t.Context(), changed); err == nil || !strings.Contains(err.Error(), "digest changed") {
			t.Fatalf("installer verification error=%v", err)
		}
	})
	t.Run("source hash consistency", func(t *testing.T) {
		changed := request
		setPkginfo(t, &changed, string(raw(map[string]any{"name": "Example", "version": "1.0", "installer_item_hash": strings.Repeat("0", 64)})))
		if _, err := Handle(t.Context(), changed); err == nil || !strings.Contains(err.Error(), "does not match") {
			t.Fatalf("source verification error=%v", err)
		}
	})
	t.Run("cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		if _, err := Handle(ctx, request); !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation error=%v", err)
		}
	})
}

func (fixture *apiFixture) serveUpload(response http.ResponseWriter, request *http.Request, read func() map[string]any, write func(any)) {
	switch {
	case (request.URL.Path == "/api/munki/package-installers" || request.URL.Path == "/api/munki/icons") && request.Method == http.MethodPost:
		body := read()
		fixture.uploads++
		id := int64(fixture.uploads)
		fixture.names[id], _ = body["filename"].(string)
		write(map[string]any{"object_id": id, "upload": map[string]any{"strategy": "direct-put", "target": map[string]any{"url": fixture.origin + "/transfer/" + strconv.FormatInt(id, 10), "method": "PUT", "headers": map[string]string{"Content-Type": "application/octet-stream"}}}})
		return
	case strings.HasPrefix(request.URL.Path, "/api/munki/package-installers/") && request.Method == http.MethodPut:
		id, _ := strconv.ParseInt(strings.TrimPrefix(request.URL.Path, "/api/munki/package-installers/"), 10, 64)
		content := fixture.objects[id]
		digest := sha256.Sum256(content)
		hash := hex.EncodeToString(digest[:])
		if fixture.forgeDigest {
			hash = strings.Repeat("0", 64)
		}
		write(map[string]any{"id": id, "sha256": hash, "size_bytes": len(content)})
		return
	}
	http.NotFound(response, request)
}

func (fixture *apiFixture) servePackages(response http.ResponseWriter, request *http.Request, read func() map[string]any, write func(any)) {
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
				connection, _, err := http.NewResponseController(response).Hijack()
				if err == nil {
					_ = connection.Close()
				}
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

func TestPlanRejectsPaddedArtifactFilename(t *testing.T) {
	artifact := plugin.Artifact{Filename: " Example.pkg", Version: "1.0", Size: 1, SHA256: strings.Repeat("a", 64)}
	remote := client{config: config{Name: "Example", Version: "1.0"}}
	var metadata metadata
	if err := json.Unmarshal(json.RawMessage(`{"version":"1.0","installer_type":"pkg"}`), &metadata.pkg); err != nil {
		t.Fatal(err)
	}
	if _, err := remote.plan(artifact, metadata, observation{}); err == nil {
		t.Fatal("accepted a filename that changes during upload")
	}
}

func changeInstaller(t *testing.T, request *plugin.ReconcileRequest, body string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "Example.pkg")
	if err := os.WriteFile(path, []byte(body), 0o400); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256([]byte(body))
	request.Inputs["installer"] = plugin.Artifact{Path: path, Filename: "Example.pkg", Format: "pkg", Size: int64(len(body)), SHA256: hex.EncodeToString(hash[:])}
}

func (fixture *apiFixture) packageList(query string) []any {
	items := []any{}
	for _, item := range fixture.packages {
		if query == "" || item["version"] == query {
			items = append(items, item)
		}
	}
	return items
}
