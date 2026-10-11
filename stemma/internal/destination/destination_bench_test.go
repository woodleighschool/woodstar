package destination

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/woodleighschool/stemma/plugin"
	"github.com/woodleighschool/woodstar/stemma/internal/destination/api"
)

func BenchmarkReconcile(b *testing.B) {
	for _, scenario := range []string{"unchanged-plan", "unchanged-apply", "metadata-apply", "publish"} {
		b.Run(scenario, func(b *testing.B) {
			fixture, connection := serveFixture(b)
			installer := installerFixture(b, "Benchmark.pkg", strings.Repeat("installer", (4<<20)/9), "1.0")
			request := plugin.ReconcileRequest[api.Config]{Method: "apply", Prepared: true, Config: connection, Identity: plugin.Identity{Resource: plugin.ResourceReference{Kind: "MacSoftware", Name: "Benchmark"}}, Artifact: installer, Metadata: json.RawMessage(`{"pkginfo":{"description":"Managed"}}`)}
			if _, err := Handle(b.Context(), request); err != nil {
				b.Fatal(err)
			}
			if scenario == "unchanged-plan" {
				request.Method = "plan"
			}
			request.Artifact.Path = ""
			b.ReportAllocs()
			for b.Loop() {
				// Each publication starts from the same remote state without accumulating blobs.
				b.StopTimer()
				fixture.mu.Lock()
				if scenario == "publish" {
					fixture.pkg["installer_file"] = nil
					clear(fixture.objects)
					clear(fixture.declared)
					clear(fixture.names)
				}
				if scenario == "metadata-apply" {
					fixture.software["description"] = "Old"
				}
				fixture.mu.Unlock()
				b.StartTimer()
				response, err := Handle(b.Context(), request)
				if err != nil {
					b.Fatal(err)
				}
				if response.Content != nil {
					leased := request
					leased.Artifact = installer
					response, err = Handle(b.Context(), leased)
					if err != nil || response.Content != nil {
						b.Fatalf("publication=%+v err=%v", response, err)
					}
				}
			}
		})
	}
}
