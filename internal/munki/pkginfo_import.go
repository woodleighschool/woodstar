package munki

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/munki/software"
)

// PkginfoImport separates sparse native metadata from references that require
// repository lookup. Nil reference slices are unmanaged; empty slices clear them.
type PkginfoImport struct {
	Name              string
	Software          software.Patch
	Package           packages.Patch
	Requires          []PkginfoReference
	UpdateFor         []PkginfoReference
	InstallerItemHash string
}

// PkginfoReference selects native software identity and an optional version.
type PkginfoReference struct {
	Name    string
	Version string
}

// ImportPkginfo converts a native Munki JSON document to administrative mutations.
// Installer locations belong to object storage, while catalogs and icons require
// separate repository configuration. No omitted metadata field becomes managed.
func ImportPkginfo(data json.RawMessage) (PkginfoImport, error) {
	var result PkginfoImport
	fields, err := pkginfoObject(data)
	if err != nil {
		return result, err
	}
	if err := json.Unmarshal(fields["name"], &result.Name); err != nil {
		return result, fmt.Errorf("name: %w", err)
	}
	identity := software.CreateMutation{Name: result.Name}
	identity.Normalize()
	if err := identity.Validate(); err != nil {
		return result, err
	}
	result.Name = identity.Name
	delete(fields, "name")
	softwareFields := map[string]json.RawMessage{}
	for _, name := range []string{"display_name", "description", "category", "developer"} {
		if value, ok := fields[name]; ok {
			softwareFields[name] = value
			delete(fields, name)
		}
	}
	for _, name := range []string{"restart_action", "on_demand", "blocking_applications_none", "installer_object_id"} {
		if _, exists := fields[name]; exists {
			return result, fmt.Errorf("%s is not a native pkginfo field", name)
		}
	}
	for _, names := range [][2]string{{"RestartAction", "restart_action"}, {"OnDemand", "on_demand"}} {
		renamePkginfoField(fields, names[0], names[1])
	}
	if _, exists := fields["installer_type"]; !exists {
		fields["installer_type"] = json.RawMessage(`"pkg"`)
	}
	for _, name := range []string{"version", "installer_type"} {
		var value string
		if err := json.Unmarshal(fields[name], &value); err != nil || strings.TrimSpace(value) == "" {
			return result, fmt.Errorf("%s must be a nonblank string", name)
		}
	}
	if value, ok := fields["blocking_applications"]; ok {
		var applications []string
		if err := pkginfoArray(value, &applications); err != nil {
			return result, fmt.Errorf("blocking_applications: %w", err)
		}
		fields["blocking_applications_none"] = pkginfoJSON(len(applications) == 0)
	}
	if err := importPkginfoReferences(fields, &result); err != nil {
		return result, err
	}
	if err := importPkginfoLists(fields); err != nil {
		return result, err
	}
	if err := importPkginfoEnvironment(fields); err != nil {
		return result, err
	}
	if err := importPkginfoAlerts(fields); err != nil {
		return result, err
	}
	if err := importPkginfoObjects(fields, &result); err != nil {
		return result, err
	}
	if err := json.Unmarshal(pkginfoJSON(softwareFields), &result.Software); err != nil {
		return result, fmt.Errorf("software: %w", err)
	}
	if err := json.Unmarshal(pkginfoJSON(fields), &result.Package); err != nil {
		return result, fmt.Errorf("package: %w", err)
	}
	return result, nil
}

func importPkginfoObjects(fields map[string]json.RawMessage, result *PkginfoImport) error {
	for _, name := range []string{"catalogs", "icon_name", "icon_hash"} {
		if _, exists := fields[name]; exists {
			return fmt.Errorf("%s requires separate repository management", name)
		}
	}
	for _, name := range []string{"installer_item_location", "installer_item_hash"} {
		if value, ok := fields[name]; ok {
			var text string
			if bytes.Equal(bytes.TrimSpace(value), []byte("null")) || json.Unmarshal(value, &text) != nil {
				return fmt.Errorf("%s must be a string", name)
			}
			if name == "installer_item_hash" {
				if strings.TrimSpace(text) == "" {
					return fmt.Errorf("installer_item_hash must not be blank")
				}
				result.InstallerItemHash = text
			}
			delete(fields, name)
		}
	}
	if value, ok := fields["installer_item_size"]; ok {
		var size int64
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) || json.Unmarshal(value, &size) != nil || size < 0 {
			return fmt.Errorf("installer_item_size must be a nonnegative integer")
		}
		delete(fields, "installer_item_size")
	}
	return importPkginfoRemovals(fields)
}

func pkginfoObject(data json.RawMessage) (map[string]json.RawMessage, error) {
	var result map[string]json.RawMessage
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("expected an object")
	}
	return result, nil
}

func pkginfoArray(data json.RawMessage, target any) error {
	if data = bytes.TrimSpace(data); len(data) == 0 || data[0] != '[' {
		return fmt.Errorf("expected an array")
	}
	return json.Unmarshal(data, target)
}

func renamePkginfoField(fields map[string]json.RawMessage, from, to string) {
	if value, ok := fields[from]; ok {
		fields[to] = value
		delete(fields, from)
	}
}

func pkginfoJSON(value any) json.RawMessage {
	data, _ := json.Marshal(value)
	return data
}

func importPkginfoReferences(fields map[string]json.RawMessage, result *PkginfoImport) error {
	for name, target := range map[string]*[]PkginfoReference{"requires": &result.Requires, "update_for": &result.UpdateFor} {
		if value, ok := fields[name]; ok {
			var references []string
			if err := pkginfoArray(value, &references); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			*target = make([]PkginfoReference, 0, len(references))
			for _, reference := range references {
				refName, version, pinned := strings.Cut(reference, "--")
				identity := software.CreateMutation{Name: refName}
				identity.Normalize()
				if err := identity.Validate(); err != nil {
					return fmt.Errorf("%s reference must use a software name or name--version: %w", name, err)
				}
				if pinned && strings.TrimSpace(version) == "" {
					return fmt.Errorf("%s reference has a blank version", name)
				}
				*target = append(*target, PkginfoReference{Name: identity.Name, Version: strings.TrimSpace(version)})
			}
			delete(fields, name)
		}
	}
	return nil
}

func importPkginfoLists(fields map[string]json.RawMessage) error {
	for name, aliases := range map[string]map[string]string{
		"receipts":              {"packageid": "package_id"},
		"installs":              {"CFBundleIdentifier": "bundle_identifier", "CFBundleName": "bundle_name", "CFBundleShortVersionString": "bundle_short_version", "CFBundleVersion": "bundle_version"},
		"installer_choices_xml": {"choiceIdentifier": "choice_identifier", "choiceAttribute": "choice_attribute", "attributeSetting": "attribute_setting"},
	} {
		if value, ok := fields[name]; ok {
			var entries []json.RawMessage
			if err := pkginfoArray(value, &entries); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			for i, entry := range entries {
				mapped, err := pkginfoObject(entry)
				if err != nil {
					return fmt.Errorf("%s[%d]: %w", name, i, err)
				}
				for from, to := range aliases {
					if _, exists := mapped[to]; exists {
						return fmt.Errorf("%s[%d].%s is not a native pkginfo field", name, i, to)
					}
					renamePkginfoField(mapped, from, to)
				}
				entries[i] = pkginfoJSON(mapped)
			}
			fields[name] = pkginfoJSON(entries)
		}
	}
	return nil
}

func importPkginfoEnvironment(fields map[string]json.RawMessage) error {
	if value, ok := fields["installer_environment"]; ok {
		environment, err := pkginfoObject(value)
		if err != nil {
			return fmt.Errorf("installer_environment: %w", err)
		}
		names := make([]string, 0, len(environment))
		for name := range environment {
			names = append(names, name)
		}
		slices.Sort(names)
		entries := make([]packages.PackageInstallerEnvironmentVariable, 0, len(names))
		for _, name := range names {
			var value string
			if bytes.Equal(environment[name], []byte("null")) || json.Unmarshal(environment[name], &value) != nil {
				return fmt.Errorf("installer_environment.%s must be a string", name)
			}
			entries = append(entries, packages.PackageInstallerEnvironmentVariable{Name: name, Value: value})
		}
		fields["installer_environment"] = pkginfoJSON(entries)
	}
	return nil
}

func importPkginfoAlerts(fields map[string]json.RawMessage) error {
	for _, name := range []string{"preinstall_alert", "preuninstall_alert"} {
		if value, ok := fields[name]; ok {
			if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
				fields[name] = json.RawMessage(`{"enabled":false}`)
				continue
			}
			alert, err := pkginfoObject(value)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			for field := range alert {
				if !slices.Contains([]string{"alert_title", "alert_detail", "ok_label", "cancel_label"}, field) {
					return fmt.Errorf("%s: unknown field %q", name, field)
				}
			}
			renamePkginfoField(alert, "alert_title", "title")
			renamePkginfoField(alert, "alert_detail", "detail")
			alert["enabled"] = json.RawMessage(`true`)
			fields[name] = pkginfoJSON(alert)
		}
	}
	return nil
}

func importPkginfoRemovals(fields map[string]json.RawMessage) error {
	value, ok := fields["items_to_remove"]
	if !ok {
		return nil
	}
	var removals []map[string]json.RawMessage
	if err := pkginfoArray(value, &removals); err != nil {
		return fmt.Errorf("items_to_remove: %w", err)
	}
	delete(fields, "items_to_remove")
	var copies []map[string]json.RawMessage
	if err := pkginfoArray(fields["items_to_copy"], &copies); err != nil {
		return fmt.Errorf("items_to_remove requires corresponding items_to_copy")
	}
	for _, item := range copies {
		for key := range item {
			if !slices.Contains([]string{"source_item", "destination_path", "destination_item"}, key) {
				delete(item, key)
			}
		}
	}
	if !bytes.Equal(pkginfoJSON(copies), pkginfoJSON(removals)) || string(fields["uninstall_method"]) != `"remove_copied_items"` {
		return fmt.Errorf("items_to_remove must match the items_to_copy managed by remove_copied_items")
	}
	return nil
}
