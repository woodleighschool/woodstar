package destination

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/stemma/internal/destination/api"

	"github.com/woodleighschool/stemma/plugin"
)

func TestValidationVerifiesLeasedArtifacts(t *testing.T) {
	request := plugin.ReconcileRequest[api.Config]{
		Method: "validate", Config: api.Config{URL: "https://woodstar.test", APIKey: "synthetic-key"},
		Artifact: installerFixture(t, "Example.pkg", "synthetic installer bytes", "1.0"),
		Inputs:   map[string]plugin.Artifact{"icon": iconFixture(t, 10)},
	}
	setPkginfo(t, &request, `{"name":"Example","version":"1.0"}`)
	if _, err := Handle(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	t.Run("installer digest", func(t *testing.T) {
		changed := request
		changed.Artifact.SHA256 = strings.Repeat("0", 64)
		if _, err := Handle(t.Context(), changed); err == nil || !strings.Contains(err.Error(), "digest changed") {
			t.Fatalf("installer verification error=%v", err)
		}
	})
	t.Run("icon digest", func(t *testing.T) {
		changed := request
		icon := request.Inputs["icon"]
		icon.SHA256 = strings.Repeat("0", 64)
		changed.Inputs = map[string]plugin.Artifact{"icon": icon}
		if _, err := Handle(t.Context(), changed); err == nil || !strings.Contains(err.Error(), "digest changed") {
			t.Fatalf("icon verification error=%v", err)
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
