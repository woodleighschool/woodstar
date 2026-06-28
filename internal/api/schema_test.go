package api

import (
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
