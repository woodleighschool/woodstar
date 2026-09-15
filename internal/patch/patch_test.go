package patch_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/munki/software"
)

func TestPackagePatchPreservesNestedFieldsAndReplacesArrays(t *testing.T) {
	current := packages.PackageMutation{
		Version: "1.0", InstallerType: packages.InstallerTypeNoPkg,
		Notes: "old", RestartAction: packages.RestartAction("RequireRestart"), SupportedArchitectures: []string{"arm64"},
		UninstallMethod: packages.UninstallMethod("uninstall_script"),
		PreinstallAlert: packages.PackageAlert{Enabled: true, Title: "Keep", Detail: "Replace"},
		Requires:        []packages.PackageReferenceMutation{{SoftwareID: 1}},
	}
	var document packages.Patch
	if err := json.Unmarshal([]byte(` {"notes":null,"restart_action":null,"uninstall_method":null,"preinstall_alert":{"detail":"New"},"requires":[],"supported_architectures":["x86_64"]} `), &document); err != nil {
		t.Fatal(err)
	}
	updated, err := document.Apply(current)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != "1.0" || updated.Notes != "" || updated.RestartAction != "" || updated.UninstallMethod != "" || len(updated.Requires) != 0 ||
		!updated.PreinstallAlert.Enabled || updated.PreinstallAlert.Title != "Keep" || updated.PreinstallAlert.Detail != "New" {
		t.Fatalf("merged mutation = %+v", updated)
	}
	if current.SupportedArchitectures[0] != "arm64" || current.PreinstallAlert.Detail != "Replace" || current.Notes != "old" ||
		current.RestartAction != "RequireRestart" || current.UninstallMethod != "uninstall_script" {
		t.Fatal("patch mutated current state")
	}
	raw := document.Bytes()
	raw[0] = '['
	encoded, err := json.Marshal(document)
	if err != nil || !json.Valid(encoded) {
		t.Fatalf("patch bytes were mutable: %s, %v", encoded, err)
	}
}

func TestPatchRejectsInvalidDocument(t *testing.T) {
	for _, data := range []string{
		`null`, `[]`, `true`, `{"version":null}`, `{"installer_type":null}`, `{"notes":1}`, `{"id":1}`,
		`{"Name":"alias"}`, `{"Version":"alias"}`, `{"software_id":1}`,
		`{"unattended_install":null}`, `{"requires":null}`, `{"preinstall_alert":null}`,
		`{"preinstall_alert":{"enabled":null}}`, `{"preinstall_alert":{"unknown":true}}`,
		`{"requires":[null]}`, `{"requires":[{"software_id":1,"software_name":"alias"}]}`,
		`{"requires":[{"software_id":1,"package_id":null}]}`, `{"requires":[{}]}`,
		`{"supported_architectures":[null]}`, `{"installer_object_id":1.5}`,
		`{"installer_object_id":9223372036854775808}`, `{"force_install_after_date":"invalid"}`,
	} {
		t.Run(data, func(t *testing.T) {
			var document packages.Patch
			if err := json.Unmarshal([]byte(data), &document); !errors.Is(err, fault.ErrInvalidInput) {
				t.Fatalf("decode error = %v, want invalid input", err)
			}
		})
	}
}

func TestSoftwarePatchNullsAndNestedTargets(t *testing.T) {
	iconID := int64(9007199254740993)
	current := software.UpdateMutation{DisplayName: "Name", IconObjectID: &iconID,
		Targets: software.Targets{Include: []software.Include{{LabelID: 7, Package: software.PackageSelector{Strategy: software.PackageLatest}, Actions: []software.Action{software.ActionManagedInstalls}}}},
	}
	var document software.Patch
	if err := json.Unmarshal([]byte(`{"display_name":null,"targets":{"exclude":[]}}`), &document); err != nil {
		t.Fatal(err)
	}
	updated, err := document.Apply(current)
	if err != nil || updated.DisplayName != "" || updated.IconObjectID == nil || *updated.IconObjectID != iconID || len(updated.Targets.Include) != 1 {
		t.Fatalf("updated = %+v, error = %v", updated, err)
	}
	if err := json.Unmarshal([]byte(`{"icon_object_id":null}`), &document); err != nil {
		t.Fatal(err)
	}
	updated, err = document.Apply(current)
	if err != nil || updated.IconObjectID != nil {
		t.Fatalf("cleared icon = %v, error = %v", updated.IconObjectID, err)
	}
	for _, data := range []string{`{"name":"rename"}`, `{"targets":null}`, `{"targets":{"include":null}}`} {
		if err := json.Unmarshal([]byte(data), &document); !errors.Is(err, fault.ErrInvalidInput) {
			t.Fatalf("decode %s error = %v", data, err)
		}
	}
}

func TestPatchSchemaKeepsReplacementContract(t *testing.T) {
	registry := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	original := registry.Schema(reflect.TypeFor[packages.PackageMutation](), false, "")
	before, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	partial := (packages.Patch{}).Schema(registry)
	if len(partial.Required) != 0 || len(partial.Properties["preinstall_alert"].Required) != 0 ||
		len(partial.Properties["requires"].Items.Required) == 0 {
		t.Fatal("patch must relax objects while retaining array element requirements")
	}
	if partial.Properties["requires"].Nullable || !partial.Properties["installer_object_id"].Nullable || partial.Properties["version"].Nullable {
		t.Fatal("incorrect patch nullability")
	}
	after, err := json.Marshal(original)
	if err != nil || string(before) != string(after) {
		t.Fatalf("patch changed the replacement schema: %v", err)
	}
}

func TestPatchSchemaNullableEnums(t *testing.T) {
	for _, tt := range []struct {
		field    string
		nullable bool
	}{
		{field: "restart_action", nullable: true},
		{field: "uninstall_method", nullable: true},
		{field: "installer_type", nullable: false},
	} {
		t.Run(tt.field, func(t *testing.T) {
			registry := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
			original := registry.Schema(reflect.TypeFor[packages.PackageMutation](), false, "")
			source := original.Properties[tt.field]
			for source.Ref != "" {
				source = registry.SchemaFromRef(source.Ref)
			}
			// Spare capacity exposes appends that would overwrite the source's backing array.
			backing := append(slices.Clone(source.Enum), "unrelated")
			source.Enum = backing[:len(backing)-1]
			before, err := json.Marshal(source)
			if err != nil {
				t.Fatal(err)
			}
			want := slices.Clone(source.Enum)
			if tt.nullable {
				want = append(want, nil)
			}
			partial := (packages.Patch{}).Schema(registry)
			encoded, err := json.Marshal(partial)
			if err != nil {
				t.Fatal(err)
			}
			var generated struct {
				Properties map[string]struct {
					Enum []any `json:"enum"`
				} `json:"properties"`
			}
			if err := json.Unmarshal(encoded, &generated); err != nil {
				t.Fatal(err)
			}
			if got := generated.Properties[tt.field].Enum; !reflect.DeepEqual(got, want) {
				t.Fatalf("generated enum = %v, want %v", got, want)
			}
			after, err := json.Marshal(source)
			if err != nil || string(before) != string(after) || backing[len(backing)-1] != "unrelated" {
				t.Fatalf("patch changed the source enum: %v", err)
			}
		})
	}
}
