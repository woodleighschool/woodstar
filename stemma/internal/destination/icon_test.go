package destination

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
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
	data := []byte("synthetic PNG content")
	file := filepath.Join(t.TempDir(), "icon.png")
	if err := os.WriteFile(file, data, 0o400); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(data)
	icon := plugin.Artifact{Path: file, Filename: "icon.png", Format: "png", Size: int64(len(data)), SHA256: hex.EncodeToString(hash[:])}
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
	if exists {
		t.Fatal("disappearing owned icon was not cleared")
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
