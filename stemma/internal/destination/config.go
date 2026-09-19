package destination

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
)

// CatalogReference points a native relationship at a resource the catalog
// publishes to the same connection. It links through the Munki name that
// resource declares, so it needs no name of its own and survives one changing.
type CatalogReference struct {
	Software string `json:"software"          jsonschema:"required,description=Name of a resource published to this connection."`
	Version  string `json:"version,omitempty" jsonschema:"description=Version of that resource's package to reference instead of the software title."`
}

// Imported is a native pkginfo in the repository's terms.
type Imported struct {
	munki.PkginfoImport

	// Links holds the catalog references under requires and update_for.
	Links map[string][]CatalogReference
}

// Import parses the native document with the shared importer after setting
// aside catalog references. A nopkg package releases any installer it held.
// No reference may name self, the resource being published.
func Import(values map[string]any, self string) (Imported, error) {
	links, err := splitReferences(values, self)
	if err != nil {
		return Imported{}, err
	}
	imported, err := munki.ImportPkginfo(raw(values))
	if err != nil {
		return Imported{}, fmt.Errorf("pkginfo: %w", err)
	}
	if kind, _ := values["installer_type"].(string); strings.TrimSpace(kind) == string(packages.InstallerTypeNoPkg) {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(imported.Package.Bytes(), &fields); err != nil {
			return Imported{}, err
		}
		fields["installer_object_id"] = raw(nil)
		if err := json.Unmarshal(raw(fields), &imported.Package); err != nil {
			return Imported{}, err
		}
	}
	return Imported{PkginfoImport: imported, Links: links}, nil
}

// splitReferences separates catalog references from the Munki names the shared
// importer parses. A list left with no names stays managed and clears the field.
func splitReferences(values map[string]any, self string) (map[string][]CatalogReference, error) {
	links := map[string][]CatalogReference{}
	for _, field := range []string{"requires", "update_for"} {
		items, ok := values[field].([]any)
		if !ok {
			continue
		}
		names := make([]any, 0, len(items))
		for _, item := range items {
			object, ok := item.(map[string]any)
			if !ok {
				names = append(names, item)
				continue
			}
			var reference CatalogReference
			decoder := json.NewDecoder(bytes.NewReader(raw(object)))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&reference); err != nil || strings.TrimSpace(reference.Software) == "" {
				return nil, fmt.Errorf("%s reference must be a Munki name or name a catalog resource under software", field)
			}
			if reference.Software == self {
				return nil, fmt.Errorf("%s reference cannot name the resource itself", field)
			}
			links[field] = append(links[field], reference)
		}
		values[field] = names
	}
	return links, nil
}
