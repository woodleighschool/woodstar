package humaschema

import (
	"reflect"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

const (
	munkiPackagePrefix   = "github.com/woodleighschool/woodstar/internal/munki"
	osqueryPackagePrefix = "github.com/woodleighschool/woodstar/internal/osquery"
	santaPackagePrefix   = "github.com/woodleighschool/woodstar/internal/santa"
)

// WoodstarSchemaNamer keeps OpenAPI component names stable when different
// Woodstar modules use the same local Go type names, such as Munki's State and
// Santa's HostState.
func WoodstarSchemaNamer(t reflect.Type, hint string) string {
	t = derefSchemaType(t)
	name := huma.DefaultSchemaNamer(t, hint)

	switch schemaPackagePrefix(t.PkgPath()) {
	case "Munki":
		return prefixIfAmbiguous("Munki", name)
	case "Osquery":
		return prefixIfAmbiguous("Osquery", name)
	case "Santa":
		return prefixIfAmbiguous("Santa", name)
	default:
		return name
	}
}

func schemaPackagePrefix(path string) string {
	switch {
	case isPackageOrSubpackage(path, munkiPackagePrefix):
		return "Munki"
	case isPackageOrSubpackage(path, osqueryPackagePrefix):
		return "Osquery"
	case isPackageOrSubpackage(path, santaPackagePrefix):
		return "Santa"
	default:
		return ""
	}
}

func isPackageOrSubpackage(path, prefix string) bool {
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

func prefixIfAmbiguous(prefix, name string) string {
	switch name {
	case "Artifact", "ArtifactMutation",
		"Check", "CheckMutation",
		"Configuration", "Event",
		"HostState",
		"Package", "PackageImportMutation", "PackageMutation",
		"Report", "ReportMutation",
		"Rule",
		"SoftwareInclude", "SoftwareMutation", "SoftwareTargets", "SoftwareTitle", "SoftwareTitleMutation",
		"State", "Status":
		return prefix + name
	default:
		return name
	}
}

func derefSchemaType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	for {
		switch t.Kind() {
		case reflect.Array, reflect.Map, reflect.Slice:
			t = t.Elem()
			for t.Kind() == reflect.Pointer {
				t = t.Elem()
			}
		default:
			return t
		}
	}
}
