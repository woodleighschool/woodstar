package api

import (
	"reflect"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

const woodstarInternalPrefix = "github.com/woodleighschool/woodstar/internal/"

func schemaNamer(t reflect.Type, hint string) string {
	t = derefSchemaType(t)
	name := huma.DefaultSchemaNamer(t, hint)

	prefix := capabilityPrefix(t.PkgPath())
	if prefix == "" || strings.HasPrefix(name, prefix) {
		return name
	}
	return prefix + name
}

// capabilityPrefix maps a package path to its OpenAPI name prefix, or "" for
// shared and core packages that need no capability scoping.
func capabilityPrefix(pkgPath string) string {
	rest, ok := strings.CutPrefix(pkgPath, woodstarInternalPrefix)
	if !ok {
		return ""
	}
	capability, _, _ := strings.Cut(rest, "/")
	switch capability {
	case "munki":
		return "Munki"
	case "osquery":
		return "Osquery"
	case "santa":
		return "Santa"
	default:
		return ""
	}
}

func derefSchemaType(t reflect.Type) reflect.Type {
	for {
		switch t.Kind() {
		case reflect.Array, reflect.Map, reflect.Pointer, reflect.Slice:
			t = t.Elem()
		default:
			return t
		}
	}
}
