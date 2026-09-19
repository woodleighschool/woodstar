package destination

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/woodleighschool/woodstar/stemma/internal/destination/api"

	"github.com/woodleighschool/stemma/plugin"

	"github.com/woodleighschool/woodstar/internal/munki/packages"
)

func TestValidationAcceptsResourceReferences(t *testing.T) {
	request := plugin.ReconcileRequest[api.Config]{Method: "validate", Identity: plugin.Identity{Resource: plugin.ResourceReference{Kind: "MacSoftware", Name: "example"}}, Config: api.Config{URL: "https://woodstar.test", APIKey: "synthetic-key"}}
	setPkginfo(t, &request, `{"name":"Example","version":"1.0","installer_type":"nopkg","requires":["Rosetta",{"resource":{"kind":"MacSoftware","name":"office"},"version":"16.1"}],"update_for":[{"resource":{"kind":"MacSoftware","name":"office"}}]}`)
	_, err := Handle(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range []string{`{"software":"office"}`, `{"resource":{"name":"office"}}`, `{"resource":{"kind":"MacSoftware","name":"office","output":"installer"}}`, `{"name":"Rosetta"}`, `{"resource":{"kind":"MacSoftware","name":""}}`, `{"resource":{"kind":"MacSoftware","name":"example"}}`, `{"resource":{"kind":"MacSoftware","name":"office"},"version":1}`} {
		t.Run(entry, func(t *testing.T) {
			invalid := request
			setPkginfo(t, &invalid, `{"name":"Example","version":"1.0","installer_type":"nopkg","requires":[`+entry+`]}`)
			if _, err := Handle(t.Context(), invalid); err == nil || !strings.Contains(err.Error(), "requires reference") {
				t.Fatalf("accepted %s: %v", entry, err)
			}
		})
	}
}

func TestPlanLinksResourceReferencesThroughPeers(t *testing.T) {
	// The repository's search is a substring match, so a name also finds its neighbours.
	titles := map[int64]string{7: "Rosetta", 8: "office", 9: "Rosetta 2 Updater"}
	var writes atomic.Int32
	connection, _ := serveAPI(t, http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writes.Add(1)
			http.Error(response, "unexpected write", http.StatusBadRequest)
			return
		}
		response.Header().Set("Content-Type", "application/json")
		id, _ := strconv.ParseInt(strings.TrimPrefix(request.URL.Path, "/api/munki/software/"), 10, 64)
		switch {
		case request.URL.Path == "/api/munki/software":
			items := []any{}
			for _, id := range []int64{9, 8, 7} {
				if matches(titles[id], request.URL.Query().Get("q")) {
					items = append(items, map[string]any{"id": id, "name": titles[id]})
				}
			}
			_ = json.NewEncoder(response).Encode(map[string]any{"items": items, "count": len(items)})
		case titles[id] != "":
			_ = json.NewEncoder(response).Encode(map[string]any{"id": id, "name": titles[id]})
		case request.URL.Path == "/api/munki/packages" && request.URL.Query().Get("software_id") == "7":
			_, _ = io.WriteString(response, `{"items":[{"id":70,"version":"1.0","software":{"id":7,"name":"Rosetta"}}],"count":1}`)
		default:
			http.NotFound(response, request)
		}
	}))
	request := plugin.ReconcileRequest[api.Config]{
		Method:   "plan",
		Identity: plugin.Identity{Project: "fixture", Resource: plugin.ResourceReference{Kind: "MacSoftware", Name: "example"}, Destination: "woodstar"},
		Config:   connection,
		// A peer is found under the Munki name it declares here, normalized as its own
		// publication is, else under its own name.
		Peers: map[string]json.RawMessage{"stemma/v1alpha1/MacSoftware/rosetta": json.RawMessage(`{"pkginfo":{"name":" Rosetta "}}`), "stemma/v1alpha1/MacSoftware/office": json.RawMessage(`{"pkginfo":{"version":"16.1"}}`)},
	}
	setPkginfo(t, &request, `{"name":"Example","version":"1.0","installer_type":"nopkg","requires":["office",{"resource":{"kind":"MacSoftware","name":"rosetta"},"version":"1.0"}],"update_for":[{"resource":{"kind":"MacSoftware","name":"office"}}]}`)
	response, err := Handle(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]packages.PackageReferenceMutation{
		"package.requires":   {{SoftwareID: 8}, {SoftwareID: 7, PackageID: 70}},
		"package.update_for": {{SoftwareID: 8}},
	}
	for _, change := range response.Changes {
		expected, linked := want[change.Field]
		if !linked {
			continue
		}
		var got []packages.PackageReferenceMutation
		if err := json.Unmarshal(change.After, &got); err != nil || !reflect.DeepEqual(got, expected) {
			t.Fatalf("%s = %s, want %+v (%v)", change.Field, change.After, expected, err)
		}
		delete(want, change.Field)
	}
	if len(want) != 0 || writes.Load() != 0 {
		t.Fatalf("references not planned: %v writes=%d", want, writes.Load())
	}
	for _, test := range []struct{ name, peer, want string }{
		{"resource without this destination", "", `resource stemma/v1alpha1/MacSoftware/rosetta does not publish to this destination`},
		{"resource not yet published", `{"pkginfo":{"name":"Rosetta 2"}}`, `resource stemma/v1alpha1/MacSoftware/rosetta is not published on this connection as "Rosetta 2"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			unlinked := request
			unlinked.Peers = map[string]json.RawMessage{"stemma/v1alpha1/MacSoftware/office": request.Peers["stemma/v1alpha1/MacSoftware/office"]}
			if test.peer != "" {
				unlinked.Peers["stemma/v1alpha1/MacSoftware/rosetta"] = json.RawMessage(test.peer)
			}
			if _, err := Handle(t.Context(), unlinked); err == nil || !strings.Contains(err.Error(), test.want) || writes.Load() != 0 {
				t.Fatalf("error=%v writes=%d", err, writes.Load())
			}
		})
	}
}
