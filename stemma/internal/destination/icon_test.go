package destination

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/woodleighschool/stemma/plugin"
)

func TestInterruptedIconPublicationConverges(t *testing.T) {
	for _, test := range []struct {
		name          string
		refused, lost bool
		uploads       int
	}{
		// A refused attach publishes nothing, so the next run uploads again.
		{name: "attach refused", refused: true, uploads: 2},
		// A lost reply hides an attach the repository made, which the next run observes.
		{name: "attach reply lost", lost: true, uploads: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, connection := serveFixture(t)
			icon := iconFixture(t, 10)
			request := plugin.ReconcileRequest{Method: "plan", Prepared: true, Identity: plugin.Identity{Software: "Policy"}, Config: connection, Metadata: raw(map[string]any{"pkginfo": map[string]string{"installer_type": "nopkg", "version": "1"}}), Inputs: map[string]plugin.Artifact{"icon": icon}}
			if _, err := Handle(t.Context(), request); err != nil {
				t.Fatal(err)
			}
			if state.writes() != 0 {
				t.Fatal("plan wrote icon")
			}
			fault := func(refused, lost bool) {
				state.mu.Lock()
				defer state.mu.Unlock()
				state.failIconAttach, state.dropIconReply = refused, lost
			}
			fault(test.refused, test.lost)
			request.Method = "apply"
			if _, err := Handle(t.Context(), request); err == nil {
				t.Fatal("reported an icon whose attach did not answer")
			}
			fault(false, false)
			if _, err := Handle(t.Context(), request); err != nil {
				t.Fatal(err)
			}
			state.mu.Lock()
			attached, _ := state.software["icon_file"].(map[string]any)
			state.mu.Unlock()
			if attached["sha256"] != icon.SHA256 || state.uploadCount() != test.uploads {
				t.Fatalf("icon=%v uploads=%d, want %d", attached, state.uploadCount(), test.uploads)
			}
			assertFixtureConverged(t, state, request)
		})
	}
}

func TestUndeclaredIconLeavesPublishedArtwork(t *testing.T) {
	state, connection := serveFixture(t)
	request := plugin.ReconcileRequest{Method: "apply", Prepared: true, Identity: plugin.Identity{Software: "Policy"}, Config: connection, Metadata: raw(map[string]any{"pkginfo": map[string]string{"installer_type": "nopkg", "version": "1"}}), Inputs: map[string]plugin.Artifact{"icon": iconFixture(t, 10)}}
	if _, err := Handle(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	request.Inputs = nil
	if _, err := Handle(t.Context(), request); err != nil {
		t.Fatal(err)
	}
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

func TestIconBootstrapAndReplacement(t *testing.T) { //nolint:funlen // One publication lifecycle verifies icon writes against unchanged parent and installer state.
	state, connection := serveFixture(t)
	request := plugin.ReconcileRequest{Method: "apply", Prepared: true, Identity: plugin.Identity{Software: "Example"}, Config: connection, Metadata: raw(map[string]any{"pkginfo": map[string]string{"version": "1.0"}}), Artifact: installerFixture(t, "Example.pkg", "synthetic installer bytes; never executed", "1.0")}
	call := func() plugin.ReconcileResponse {
		t.Helper()
		response, err := Handle(t.Context(), request)
		if err != nil {
			t.Fatal(err)
		}
		return response
	}
	call()
	first, second := iconFixture(t, 10), iconFixture(t, 200)
	request.Inputs = map[string]plugin.Artifact{"icon": first}
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
		installer, _ := state.pkg["installer_file"].(map[string]any)
		if installer["sha256"] != request.Artifact.SHA256 || state.createdSoftware != 1 || state.createdPackages != 1 {
			t.Fatal("icon publication changed parent or installer identity")
		}
	}
	check(first)
	before := state.writes()
	if response := call(); len(response.Changes) != 0 || state.writes() != before {
		t.Fatal("unchanged icon planned or wrote changes")
	}
	// Changed bytes are ordinary drift: planning isolates them, applying replaces them.
	request.Inputs["icon"] = second
	request.Method = "plan"
	if response := call(); len(response.Changes) != 1 || response.Changes[0].Field != "software.icon" || state.writes() != before {
		t.Fatal("changed icon plan did not isolate the icon")
	}
	request.Method = "apply"
	call()
	check(second)
	if state.updates != updates || state.uploadCount() != 3 {
		t.Fatal("icon replacement wrote unrelated metadata or reuploaded installer")
	}
	// Without a declared icon the published artwork is left unchanged.
	before = state.writes()
	request.Inputs = nil
	call()
	check(second)
	if state.writes() != before {
		t.Fatal("iconless run changed the published icon")
	}
	state.mu.Lock()
	delete(state.software, "icon_object_id")
	delete(state.software, "icon_file")
	state.mu.Unlock()
	request.Inputs = map[string]plugin.Artifact{"icon": second}
	call()
	check(second)
	if state.uploadCount() != 4 {
		t.Fatal("missing remote icon did not create one new upload")
	}
}
