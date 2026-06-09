package humaschema_test

import (
	"reflect"
	"testing"

	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/humaschema"
	"github.com/woodleighschool/woodstar/internal/munki/artifacts"
	"github.com/woodleighschool/woodstar/internal/munki/hoststate"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/osquery/checks"
	"github.com/woodleighschool/woodstar/internal/osquery/reports"
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	santarules "github.com/woodleighschool/woodstar/internal/santa/rules"
)

func TestWoodstarSchemaNamerPrefixesAmbiguousCapabilityNames(t *testing.T) {
	tests := []struct {
		name string
		typ  reflect.Type
		want string
	}{
		{
			name: "munki host state",
			typ:  reflect.TypeFor[hoststate.State](),
			want: "MunkiState",
		},
		{
			name: "munki artifact",
			typ:  reflect.TypeFor[artifacts.Artifact](),
			want: "MunkiArtifact",
		},
		{
			name: "munki software include",
			typ:  reflect.TypeFor[munkisoftware.SoftwareInclude](),
			want: "MunkiSoftwareInclude",
		},
		{
			name: "munki software targets",
			typ:  reflect.TypeFor[munkisoftware.SoftwareTargets](),
			want: "MunkiSoftwareTargets",
		},
		{
			name: "munki package mutation",
			typ:  reflect.TypeFor[packages.PackageMutation](),
			want: "MunkiPackageMutation",
		},
		{
			name: "munki package create mutation",
			typ:  reflect.TypeFor[packages.PackageCreateMutation](),
			want: "MunkiPackageCreateMutation",
		},
		{
			name: "munki software mutation",
			typ:  reflect.TypeFor[munkisoftware.SoftwareMutation](),
			want: "MunkiSoftwareMutation",
		},
		{
			name: "osquery check",
			typ:  reflect.TypeFor[checks.Check](),
			want: "OsqueryCheck",
		},
		{
			name: "osquery report mutation",
			typ:  reflect.TypeFor[reports.ReportMutation](),
			want: "OsqueryReportMutation",
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
			typ:  reflect.TypeFor[[]*hoststate.State](),
			want: "MunkiState",
		},
		{
			name: "non ambiguous capability name",
			typ:  reflect.TypeFor[hoststate.Item](),
			want: "Item",
		},
		{
			name: "non capability package",
			typ:  reflect.TypeFor[directory.User](),
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
