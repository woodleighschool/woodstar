package destination

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/woodleighschool/stemma/plugin"
)

func TestIconUploadRecoveryAndOwnership(t *testing.T) {
	state := &apiFixture{objects: map[int64][]byte{}, names: map[int64]string{}, dropIconReply: true}
	server := httptest.NewTLSServer(state)
	t.Cleanup(server.Close)
	state.origin = server.URL
	ca := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0o600); err != nil {
		t.Fatal(err)
	}
	icon := iconFixture(t, 10)
	request := plugin.ReconcileRequest{Method: "plan", Prepared: true, Identity: plugin.Identity{Software: "Policy"}, Config: raw(map[string]string{"url": server.URL, "api_key": "synthetic-key", "ca_file": ca}), Metadata: raw(map[string]any{"pkginfo": map[string]string{"installer_type": "nopkg", "version": "1"}}), Inputs: map[string]plugin.Artifact{"icon": icon}}
	if _, err := Handle(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	if state.writes() != 0 {
		t.Fatal("plan wrote icon")
	}
	request.Method = "apply"
	result, err := Handle(t.Context(), request)
	if err == nil || len(result.Binding) == 0 {
		t.Fatalf("lost response not retained: %v %s", err, result.Binding)
	}
	request.Binding = result.Binding
	state.mu.Lock()
	state.dropIconReply = false
	state.mu.Unlock()
	result, err = Handle(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	request.Binding = result.Binding
	if state.uploadCount() != 1 {
		t.Fatal("completed icon upload replayed")
	}
	request.Inputs = nil
	result, err = Handle(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	request.Binding = result.Binding
	state.mu.Lock()
	_, exists := state.software["icon_object_id"]
	state.software["icon_object_id"] = json.Number("99")
	state.mu.Unlock()
	if !exists {
		t.Fatal("missing input cleared the published icon")
	}
	if _, err := Handle(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if number(state.software["icon_object_id"]) != 99 {
		t.Fatal("manual icon was cleared")
	}
}

func iconFixture(t *testing.T, value uint8) plugin.Artifact {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.SetRGBA(0, 0, color.RGBA{R: value, A: 255})
	var data bytes.Buffer
	if err := png.Encode(&data, img); err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(t.TempDir(), "icon.png")
	if err := os.WriteFile(name, data.Bytes(), 0o400); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(data.Bytes())
	return plugin.Artifact{Path: name, Filename: "icon.png", Format: "png", Size: int64(data.Len()), SHA256: hex.EncodeToString(hash[:])}
}

func TestCompiledPluginIconBootstrapAndRefresh(t *testing.T) { //nolint:funlen // One publication lifecycle verifies icon writes against unchanged parent and installer state.
	binary := buildPlugin(t)
	state := &apiFixture{objects: map[int64][]byte{}, names: map[int64]string{}}
	server := httptest.NewTLSServer(state)
	t.Cleanup(server.Close)
	state.origin = server.URL
	ca := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0o600); err != nil {
		t.Fatal(err)
	}
	data := []byte("synthetic installer bytes; never executed")
	name := filepath.Join(t.TempDir(), "Example.pkg")
	if err := os.WriteFile(name, data, 0o400); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(data)
	request := plugin.ReconcileRequest{Method: "apply", Prepared: true, Identity: plugin.Identity{Software: "Example"}, Config: raw(map[string]string{"url": server.URL, "api_key": "synthetic-key", "ca_file": ca}), Metadata: raw(map[string]any{"pkginfo": map[string]string{"version": "1.0"}}), Artifact: plugin.Artifact{Path: name, Filename: "Example.pkg", Format: "pkg", Version: "1.0", Size: int64(len(data)), SHA256: hex.EncodeToString(hash[:])}}
	call := func() plugin.ReconcileResponse {
		t.Helper()
		response, err := runPlugin(t, binary, request)
		if err != nil {
			t.Fatal(err)
		}
		request.Binding = response.Binding
		return response
	}
	call()
	portable, native := iconFixture(t, 10), iconFixture(t, 200)
	request.Inputs = map[string]plugin.Artifact{"icon": portable}
	updates := state.updates
	call()
	if state.updates != updates {
		t.Fatal("icon-only publication patched software or package metadata")
	}
	check := func(want plugin.Artifact) {
		t.Helper()
		state.mu.Lock()
		defer state.mu.Unlock()
		actual, _ := state.software["icon_file"].(map[string]any)
		if actual["sha256"] != want.SHA256 {
			t.Fatalf("icon hash = %v, want %s", actual["sha256"], want.SHA256)
		}
		var saved binding
		if err := json.Unmarshal(request.Binding, &saved); err != nil {
			t.Fatal(err)
		}
		if saved.Icon != nil {
			t.Fatal("completed icon remained in upload state")
		}
		installer, _ := state.pkg["installer_file"].(map[string]any)
		if installer["sha256"] != request.Artifact.SHA256 || state.createdSoftware != 1 || state.createdPackages != 1 {
			t.Fatal("icon publication changed parent or installer identity")
		}
	}
	check(portable)
	request.Inputs["icon"] = native
	before := state.writes()
	if response := call(); len(response.Changes) != 0 || state.writes() != before {
		t.Fatal("normal apply replaced an existing icon")
	}
	request.RefreshIcons = true
	request.Method = "plan"
	if response := call(); len(response.Changes) != 1 || response.Changes[0].Field != "software.icon" || state.writes() != before {
		t.Fatal("refresh plan did not isolate icon change")
	}
	request.Method = "apply"
	call()
	check(native)
	if state.updates != updates || state.uploadCount() != 3 {
		t.Fatal("refresh wrote unrelated metadata or reuploaded installer")
	}
	request.RefreshIcons = false
	request.Inputs["icon"] = portable
	before = state.writes()
	call()
	check(native)
	request.Inputs = nil
	call()
	check(native)
	if state.writes() != before {
		t.Fatal("portable or iconless run changed retained icon")
	}
	state.mu.Lock()
	delete(state.software, "icon_object_id")
	delete(state.software, "icon_file")
	state.mu.Unlock()
	request.Inputs = map[string]plugin.Artifact{"icon": portable}
	call()
	check(portable)
	if state.uploadCount() != 4 {
		t.Fatal("missing remote icon did not create one new upload")
	}
}

func TestRefreshResumesSupersededIconUpload(t *testing.T) { //nolint:gocognit // Both interrupted-upload cases share the same sequential publication assertions.
	for _, missing := range []bool{false, true} {
		t.Run(fmt.Sprintf("pending-bytes-missing=%v", missing), func(t *testing.T) {
			state := &apiFixture{objects: map[int64][]byte{}, names: map[int64]string{}}
			server := httptest.NewTLSServer(state)
			t.Cleanup(server.Close)
			state.origin = server.URL
			ca := filepath.Join(t.TempDir(), "ca.pem")
			if err := os.WriteFile(ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0o600); err != nil {
				t.Fatal(err)
			}
			first := iconFixture(t, 10)
			request := plugin.ReconcileRequest{Method: "apply", Prepared: true, Identity: plugin.Identity{Software: "Policy"}, Config: raw(map[string]string{"url": server.URL, "api_key": "synthetic-key", "ca_file": ca}), Metadata: raw(map[string]any{"pkginfo": map[string]string{"installer_type": "nopkg", "version": "1"}}), Inputs: map[string]plugin.Artifact{"icon": first}}
			response, err := Handle(t.Context(), request)
			if err != nil {
				t.Fatal(err)
			}
			request.Binding = response.Binding
			previous := iconFixture(t, 20)
			request.Inputs["icon"] = previous
			request.RefreshIcons = true
			state.failIconAttach = true
			response, err = Handle(t.Context(), request)
			if err == nil {
				t.Fatal("expected interrupted attach")
			}
			request.Binding = response.Binding
			var pending binding
			if err := json.Unmarshal(response.Binding, &pending); err != nil || pending.Icon == nil {
				t.Fatal("lost pending object")
			}
			if err := os.Remove(previous.Path); err != nil {
				t.Fatal(err)
			}
			if missing {
				delete(state.objects, pending.Icon.ObjectID)
			}
			state.failIconAttach = false
			latest := iconFixture(t, 30)
			request.Inputs["icon"] = latest
			response, err = Handle(t.Context(), request)
			if err != nil {
				t.Fatal(err)
			}
			attached, _ := state.software["icon_file"].(map[string]any)
			if attached["sha256"] != latest.SHA256 || state.uploadCount() != 3 {
				t.Fatal("superseded pending artwork prevented current publication or replayed bytes")
			}
			pending = binding{}
			if err := json.Unmarshal(response.Binding, &pending); err != nil || pending.Icon != nil {
				t.Fatal("completed icon retained pending state")
			}
		})
	}
}
