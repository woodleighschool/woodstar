package api

import (
	"log/slog"
	"reflect"
	"testing"

	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/osquery/checks"
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
)

func TestSchemaNamerScopesCapabilityTypes(t *testing.T) {
	tests := []struct {
		name string
		typ  reflect.Type
		want string
	}{
		{
			name: "santa root type",
			typ:  reflect.TypeFor[santa.HostState](),
			want: "SantaHostState",
		},
		{
			name: "santa subpackage type",
			typ:  reflect.TypeFor[configurations.Configuration](),
			want: "SantaConfiguration",
		},
		{
			name: "osquery subpackage type",
			typ:  reflect.TypeFor[checks.Check](),
			want: "OsqueryCheck",
		},
		{
			name: "munki host state",
			typ:  reflect.TypeFor[munki.HostState](),
			want: "MunkiHostState",
		},
		{
			name: "munki type without a familiar name still scoped",
			typ:  reflect.TypeFor[munki.Item](),
			want: "MunkiItem",
		},
		{
			name: "munki package type is scoped",
			typ:  reflect.TypeFor[packages.Package](),
			want: "MunkiPackage",
		},
		{
			name: "pointer slice element keeps capability scope",
			typ:  reflect.TypeFor[[]*munki.HostState](),
			want: "MunkiHostState",
		},
		{
			name: "core package type left bare",
			typ:  reflect.TypeFor[directory.User](),
			want: "User",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := schemaNamer(tt.typ, ""); got != tt.want {
				t.Fatalf("schemaNamer() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProtectedOperationsDeclareAuthentication(t *testing.T) {
	api := BuildSchemaAPI("test", &Dependencies{Logger: slog.New(slog.DiscardHandler)})
	doc := api.OpenAPI()

	if doc.Components.SecuritySchemes["cookieAuth"] == nil {
		t.Fatal("cookieAuth security scheme is missing")
	}
	bearer := doc.Components.SecuritySchemes["bearerAuth"]
	if bearer == nil || bearer.Type != "http" || bearer.Scheme != "bearer" {
		t.Fatalf("bearerAuth = %#v, want HTTP bearer scheme", bearer)
	}

	hosts := doc.Paths["/api/hosts"].Get
	want := []map[string][]string{{"cookieAuth": {}}, {"bearerAuth": {}}}
	if !reflect.DeepEqual(hosts.Security, want) {
		t.Fatalf("hosts security = %#v, want %#v", hosts.Security, want)
	}
	if hosts.Responses["401"] == nil {
		t.Fatal("protected operation does not declare a 401 response")
	}
	if doc.Paths["/api/hosts/bulk-delete"].Post.Responses["403"] == nil {
		t.Fatal("ordinary mutation does not declare a 403 response")
	}
	if doc.Paths["/api/agent-secrets"].Get.Responses["403"] == nil {
		t.Fatal("sensitive read does not declare a 403 response")
	}

	if setup := doc.Paths["/api/setup"].Post; len(setup.Security) != 0 {
		t.Fatalf("setup security = %#v, want public operation", setup.Security)
	}
	if session := doc.Paths["/api/auth/session"].Get; len(session.Security) != 0 {
		t.Fatalf("session security = %#v, want optional authentication", session.Security)
	}
}
