package humaschema_test

import (
	"reflect"
	"testing"

	"github.com/woodleighschool/woodstar/internal/humaschema"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	santarules "github.com/woodleighschool/woodstar/internal/santa/rules"
	"github.com/woodleighschool/woodstar/internal/users"
)

func TestWoodstarSchemaNamerPrefixesAmbiguousCapabilityNames(t *testing.T) {
	tests := []struct {
		name string
		typ  reflect.Type
		want string
	}{
		{
			name: "munki host state",
			typ:  reflect.TypeFor[munki.HostState](),
			want: "MunkiHostState",
		},
		{
			name: "santa host state",
			typ:  reflect.TypeFor[santa.HostState](),
			want: "SantaHostState",
		},
		{
			name: "santa subpackage configuration",
			typ:  reflect.TypeFor[configurations.Configuration](),
			want: "SantaConfiguration",
		},
		{
			name: "santa subpackage rule",
			typ:  reflect.TypeFor[santarules.Rule](),
			want: "SantaRule",
		},
		{
			name: "pointer slice element",
			typ:  reflect.TypeFor[[]*munki.HostState](),
			want: "MunkiHostState",
		},
		{
			name: "non ambiguous capability name",
			typ:  reflect.TypeFor[munki.HostItem](),
			want: "HostItem",
		},
		{
			name: "non capability package",
			typ:  reflect.TypeFor[users.User](),
			want: "User",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := humaschema.WoodstarSchemaNamer(tt.typ, ""); got != tt.want {
				t.Fatalf("WoodstarSchemaNamer() = %q, want %q", got, tt.want)
			}
		})
	}
}
