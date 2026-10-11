package pkginfo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"path"
	"reflect"
	"strings"

	"github.com/woodleighschool/stemma/plugin"

	"github.com/woodleighschool/woodstar/internal/munki/packages"
)

// Derived is the pkginfo a declaration manages for one artifact.
type Derived struct {
	Values  map[string]any
	Origins map[string]string
}

// Derive maps prepared evidence to native fields. Declared values win; owned
// fields without evidence clear. Detection requires an installed path.
func Derive[C any](request plugin.ReconcileRequest[C], declared json.RawMessage) (Derived, error) {
	var explicit map[string]any
	if err := json.Unmarshal(declared, &explicit); err != nil {
		return Derived{}, err
	}
	if err := resolveSubjects(explicit); err != nil {
		return Derived{}, err
	}
	d := nativeDefaults{values: maps.Clone(explicit), explicit: explicit, origins: map[string]string{}}
	for key := range explicit {
		d.origins["pkginfo."+key] = "explicit"
	}
	// The name needs no artifact, so its rules hold before preparation.
	d.put("name", request.Identity.Resource.Name, "software.name")
	if !request.Prepared && request.Artifact.Path == "" {
		return d.result(""), nil
	}
	if request.Artifact.Version != "" {
		d.put("version", request.Artifact.Version, "artifact.version")
	}
	if minimum := request.MinimumOS; minimum != nil {
		d.put("minimum_os_version", minimum.Version, minimum.Origin)
	} else {
		d.values["minimum_os_version"] = nil
	}
	kind := d.installerType(request.Artifact)
	if kind == "nopkg" {
		if request.Artifact.Path != "" || request.Artifact.SHA256 != "" {
			return Derived{}, errors.New("nopkg must not include installer content")
		}
		return d.result(kind), nil
	}
	d.packageDefaults(request.Artifact.Facts, kind)
	selected, versionKey, err := macEvidence(request.Artifact)
	if err != nil {
		return Derived{}, err
	}
	if selected != nil {
		if err := d.application(*selected, versionKey, kind); err != nil {
			return Derived{}, err
		}
	}
	if _, exists := d.values["version"]; !exists && (kind == "pkg" || kind == "copy_from_dmg") {
		return Derived{}, fmt.Errorf("the prepared installer has no managed version%s; select an application or set pkginfo.version%s", componentVersions(request.Artifact.Facts), sourceVersion(request.Artifact))
	}
	d.uninstall(kind)
	return d.result(kind), nil
}

// componentVersions lists the versions of a package's components, which leave
// it without one of its own when they differ.
func componentVersions(facts plugin.Facts) string {
	var listed []string
	for _, subject := range facts.Subjects {
		if pkg := subject.Package; pkg != nil && pkg.Identifier != "" {
			listed = append(listed, strings.TrimSpace(pkg.Identifier+" "+pkg.Version))
		}
	}
	const shown = 6
	switch {
	case len(listed) < 2:
		return ""
	case len(listed) > shown:
		return fmt.Sprintf(": its components carry %s and %d more", strings.Join(listed[:shown], ", "), len(listed)-shown)
	}
	return ": its components carry " + strings.Join(listed, ", ")
}

// sourceVersion names the version the source states for its input, which a
// declaration can adopt.
func sourceVersion(artifact plugin.Artifact) string {
	var source struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(artifact.Evidence["source"], &source) != nil || source.Version == "" {
		return ""
	}
	return fmt.Sprintf("; the source states %s as evidence.source.version", source.Version)
}

// resolveSubjects translates the inspected applications listed under installs
// into native application items, as derivation does for the selected
// application. Native items pass through unchanged.
func resolveSubjects(explicit map[string]any) error {
	items, _ := explicit["installs"].([]any)
	var copies []packages.PackageItemToCopy
	_ = json.Unmarshal(raw(explicit["items_to_copy"]), &copies)
	for i, item := range items {
		fields, object := item.(map[string]any)
		if _, native := fields["type"]; !object || native {
			continue
		}
		var subject plugin.Subject
		decoder := json.NewDecoder(bytes.NewReader(raw(item)))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&subject); err != nil || subject.Kind != "app" || subject.App == nil {
			return fmt.Errorf("installs item %d: expected a native installs item or an inspected application, such as {{ facts.name }}", i+1)
		}
		endpoint, err := installedPath(subject.Path, subject.InstalledPath, copies)
		if err != nil {
			return fmt.Errorf("installs item %d: %w", i+1, err)
		}
		if endpoint == "" {
			return fmt.Errorf("installs item %d: application %s has no known installed path; write a native installs item", i+1, subject.ID)
		}
		native, err := applicationItem(*subject.App, endpoint, "")
		if err != nil {
			return fmt.Errorf("installs item %d: application %s: %w", i+1, subject.ID, err)
		}
		items[i] = native
	}
	return nil
}

func applicationItem(app plugin.AppFacts, endpoint, versionKey string) (map[string]any, error) {
	if versionKey == "" {
		versionKey = app.VersionKey()
	}
	version := app.Version
	if versionKey == "CFBundleVersion" {
		version = app.Build
	}
	if app.BundleID == "" || version == "" {
		return nil, errors.New("bundle identifier and comparison version are required")
	}
	return map[string]any{"type": "application", "path": endpoint, "CFBundleIdentifier": app.BundleID, "CFBundleName": app.Name, "CFBundleShortVersionString": app.Version, "CFBundleVersion": app.Build, "version_comparison_key": versionKey, "minosversion": app.MinimumOS}, nil
}

type nativeDefaults struct {
	values   map[string]any
	explicit map[string]any
	origins  map[string]string
}

func (d *nativeDefaults) declared(key string) bool {
	_, exists := d.explicit[key]
	return exists
}

func (d *nativeDefaults) put(key string, value any, origin string) {
	if d.declared(key) {
		return
	}
	d.values[key], d.origins["pkginfo."+key] = value, origin
}

// detects reports whether derivation supplies the installs list: declared
// detection of any kind replaces it.
func (d *nativeDefaults) detects() bool {
	return !d.declared("installcheck_script") && !d.declared("receipts") && !d.declared("installs")
}

// result clears owned fields when neither the declaration nor current evidence
// supplies a value. Other omitted fields remain unmanaged.
func (d *nativeDefaults) result(kind string) Derived {
	defaults := map[string]any{}
	switch kind {
	case "pkg", "copy_from_dmg":
		defaults = map[string]any{
			"receipts": []any{}, "installed_size": int64(0), "RestartAction": nil, "items_to_copy": []any{},
			"uninstallable": false, "uninstall_method": nil,
		}
	}
	if len(defaults) > 0 && d.detects() {
		defaults["installs"] = []any{}
	}
	for key, value := range defaults {
		if _, present := d.values[key]; !present {
			d.values[key] = value
		}
	}
	return Derived{Values: d.values, Origins: d.origins}
}

func (d *nativeDefaults) installerType(artifact plugin.Artifact) string {
	kind, _ := d.values["installer_type"].(string)
	if kind != "" {
		return kind
	}
	switch {
	case artifact.Format == "pkg" || strings.HasSuffix(strings.ToLower(artifact.Filename), ".pkg"):
		kind = "pkg"
	case artifact.Format == "dmg" || strings.HasSuffix(strings.ToLower(artifact.Filename), ".dmg"):
		kind = "copy_from_dmg"
	}
	if kind != "" {
		d.put("installer_type", kind, "installer.format")
	}
	return kind
}

func (d *nativeDefaults) packageDefaults(facts plugin.Facts, kind string) {
	var receipts []map[string]any
	var installedSize int64
	var installer plugin.InstallerFacts
	for _, subject := range facts.Subjects {
		if subject.Installer != nil {
			installer = *subject.Installer
		}
		pkg := subject.Package
		if pkg == nil {
			continue
		}
		if pkg.HasPayload && pkg.Identifier != "" {
			receipts = append(receipts, map[string]any{"packageid": pkg.Identifier, "version": pkg.Version, "installed_size": pkg.InstalledSize})
			installedSize += pkg.InstalledSize
		}
	}
	if kind == "pkg" {
		if len(receipts) > 0 {
			d.put("receipts", receipts, "installer.receipts")
		}
		if installedSize > 0 {
			d.put("installed_size", installedSize, "installer.receipts")
		}
		if installer.RestartAction != "" {
			d.put("RestartAction", installer.RestartAction, "installer.restart_action")
		}
	}
}

// uninstall settles each removal field on its own: the method follows the
// installer, and an item with a method is uninstallable unless the declaration
// says otherwise.
func (d *nativeDefaults) uninstall(kind string) {
	method, origin := "", ""
	switch {
	case d.declared("uninstall_method"):
		method, _ = d.explicit["uninstall_method"].(string)
		origin = "pkginfo.uninstall_method"
	case kind == "pkg" && hasEntries(d.values["receipts"]):
		method, origin = "removepackages", "installer.receipts"
	case kind == "copy_from_dmg" && hasEntries(d.values["items_to_copy"]):
		method, origin = "remove_copied_items", "app.archive_path"
	}
	if removable, _ := d.explicit["uninstallable"].(bool); method != "" && (removable || !d.declared("uninstallable")) {
		d.put("uninstallable", true, origin)
		d.put("uninstall_method", method, origin)
	}
}

func hasEntries(value any) bool {
	list := reflect.ValueOf(value)
	return list.Kind() == reflect.Slice && list.Len() > 0
}

func (d *nativeDefaults) application(subject plugin.Subject, versionKey, kind string) error {
	endpoint := subject.InstalledPath
	if kind == "copy_from_dmg" {
		copied, err := d.copyDestination(subject.Path, endpoint)
		if err != nil {
			return err
		}
		endpoint = copied
	}
	if !d.detects() {
		return nil
	}
	if endpoint == "" {
		return errors.New("selected application has no known installed path; set application.installed_path or installs")
	}
	item, err := applicationItem(*subject.App, endpoint, versionKey)
	if err != nil {
		return fmt.Errorf("selected application: %w", err)
	}
	d.put("installs", []map[string]any{item}, "app.installed_path")
	return nil
}

func (d *nativeDefaults) copyDestination(source, endpoint string) (string, error) {
	if _, exists := d.values["items_to_copy"]; !exists {
		if !safeLocation(source) {
			return "", errors.New("selected DMG application requires a safe relative archive path")
		}
		destination, item := "/Applications", ""
		if endpoint != "" {
			destination = path.Dir(endpoint)
			if path.Base(endpoint) != path.Base(source) {
				item = path.Base(endpoint)
			}
		}
		d.put("items_to_copy", []packages.PackageItemToCopy{{SourceItem: source, DestinationPath: destination, DestinationItem: item}}, "app.archive_path")
	}
	if !d.detects() {
		return endpoint, nil
	}
	var copies []packages.PackageItemToCopy
	if err := json.Unmarshal(raw(d.values["items_to_copy"]), &copies); err != nil {
		return "", err
	}
	return installedPath(source, endpoint, copies)
}

func installedPath(source, observed string, copies []packages.PackageItemToCopy) (string, error) {
	var copied string
	source = path.Clean(source)
	for _, action := range copies {
		item := path.Clean(action.SourceItem)
		if source != item && !strings.HasPrefix(source, item+"/") {
			continue
		}
		if copied != "" {
			return "", errors.New("application has multiple copy destinations; write a native installs item")
		}
		name := action.DestinationItem
		if name == "" {
			name = path.Base(item)
		}
		copied = path.Join(action.DestinationPath, name, strings.TrimPrefix(strings.TrimPrefix(source, item), "/"))
	}
	if observed != "" && copied != "" && observed != copied {
		return "", errors.New("application.installed_path disagrees with items_to_copy")
	}
	if copied != "" {
		return copied, nil
	}
	return observed, nil
}

func safeLocation(value string) bool {
	clean := path.Clean(value)
	return value != "" && !path.IsAbs(value) && clean != "." && clean != ".." && !strings.HasPrefix(clean, "../") && !strings.ContainsAny(value, "\\\x00")
}
