package pkginfo

import (
	"cmp"
	"encoding/json"
	"errors"
	"maps"
	"path"
	"reflect"
	"strconv"
	"strings"

	"github.com/woodleighschool/stemma/plugin"

	"github.com/woodleighschool/woodstar/internal/munki/packages"
)

// Derivation selects observed evidence used to fill omitted native fields.
type Derivation struct {
	App *AppDerivation `json:"app,omitempty"`
}

// AppDerivation selects a named application subject and its endpoint detection path.
type AppDerivation struct {
	Subject       string `json:"subject" jsonschema:"minLength=1"`
	InstalledPath string `json:"installed_path,omitempty"`
	VersionKey    string `json:"version_key,omitempty" jsonschema:"enum=CFBundleShortVersionString,enum=CFBundleVersion"`
}

// Validate checks the selection without requiring prepared artifacts.
func (d Derivation) Validate() error {
	app := d.App
	if app == nil {
		return nil
	}
	if app.Subject == "" {
		return errors.New("derive.app.subject is required")
	}
	if app.InstalledPath != "" && !path.IsAbs(app.InstalledPath) {
		return errors.New("derive.app.installed_path must be absolute")
	}
	if app.VersionKey != "" && app.VersionKey != "CFBundleVersion" && app.VersionKey != "CFBundleShortVersionString" {
		return errors.New("unsupported derive.app.version_key")
	}
	return nil
}

// Derived is the pkginfo a declaration manages for one artifact.
type Derived struct {
	Values  map[string]any
	Origins map[string]string
}

// Derive fills omitted native fields from artifact evidence. Declared values
// win; owned fields without evidence clear. Detection requires an established
// installed path, not just a location inside an archive.
func Derive(request plugin.ReconcileRequest, declared json.RawMessage, derivation Derivation) (Derived, error) {
	var explicit map[string]any
	if err := json.Unmarshal(declared, &explicit); err != nil {
		return Derived{}, err
	}
	d := nativeDefaults{values: maps.Clone(explicit), explicit: explicit, origins: map[string]string{}}
	for key := range explicit {
		d.origins["pkginfo."+key] = "explicit"
	}
	if !request.Prepared && request.Artifact.Path == "" {
		return d.result(""), nil
	}
	d.put("name", request.Identity.Software, "software.name")
	if request.Artifact.Version != "" {
		d.put("version", request.Artifact.Version, "installer.version")
	}
	kind := d.installerType(request.Artifact)
	if kind == "nopkg" {
		if request.Artifact.Path != "" || request.Artifact.SHA256 != "" {
			return Derived{}, errors.New("nopkg must not include installer content")
		}
		return d.result(kind), nil
	}
	facts := request.Facts
	if len(facts.Subjects) == 0 {
		facts = request.Artifact.Facts
	}
	versions, installer := d.packageDefaults(facts, kind)
	selected, versionKey, err := macEvidence(request.Artifact)
	if err != nil {
		return Derived{}, err
	}
	options := derivation.App
	if options != nil || selected == nil {
		selected, err = selectApplication(facts, request.Subjects, options)
	} else {
		options = &AppDerivation{VersionKey: versionKey}
	}
	if err != nil {
		return Derived{}, err
	}
	minimumOS, minimumOrigin := installer.MinimumOS, "installer.minimum_os"
	if selected != nil {
		if err := d.application(*selected, options, kind); err != nil {
			return Derived{}, err
		}
		// Munki takes the later of the installer and application requirements.
		if compareVersions(selected.App.MinimumOS, minimumOS) > 0 {
			minimumOS, minimumOrigin = selected.App.MinimumOS, "app.minimum_os"
		}
	} else if _, exists := d.values["version"]; !exists {
		switch {
		case installer.Version != "":
			d.put("version", installer.Version, "installer.version")
		case len(versions) == 1:
			for version := range versions {
				d.put("version", version, "installer.receipts")
			}
		case len(versions) > 1:
			return Derived{}, errors.New("PKG components declare different versions; select an application or author pkginfo.version")
		}
	}
	if minimumOS != "" {
		d.put("minimum_os_version", minimumOS, minimumOrigin)
	}
	d.uninstall(kind)
	return d.result(kind), nil
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
			"minimum_os_version": nil, "uninstallable": false, "uninstall_method": nil,
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

func (d *nativeDefaults) packageDefaults(facts plugin.Facts, kind string) (map[string]bool, plugin.InstallerFacts) {
	var receipts []map[string]any
	var installedSize int64
	var installer plugin.InstallerFacts
	versions := map[string]bool{}
	for _, subject := range facts.Subjects {
		if subject.Installer != nil {
			installer = *subject.Installer
		}
		pkg := subject.Package
		if pkg == nil {
			continue
		}
		if pkg.Version != "" {
			versions[pkg.Version] = true
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
	return versions, installer
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

// compareVersions orders dotted numeric versions such as macOS releases as Munki
// does: empty components are ignored and missing components compare as zero.
func compareVersions(a, b string) int {
	dot := func(r rune) bool { return r == '.' }
	left, right := strings.FieldsFunc(a, dot), strings.FieldsFunc(b, dot)
	for i := range max(len(left), len(right)) {
		x, y := "0", "0"
		if i < len(left) {
			x = left[i]
		}
		if i < len(right) {
			y = right[i]
		}
		order := strings.Compare(x, y)
		m, errM := strconv.ParseUint(x, 10, 64)
		n, errN := strconv.ParseUint(y, 10, 64)
		if errM == nil && errN == nil {
			order = cmp.Compare(m, n)
		}
		if order != 0 {
			return order
		}
	}
	return 0
}

func (d *nativeDefaults) application(subject plugin.Subject, options *AppDerivation, kind string) error {
	app := subject.App
	version, versionKey, endpoint := app.Version, app.VersionKey(), subject.InstalledPath
	if options != nil {
		if options.VersionKey != "" {
			versionKey = options.VersionKey
		}
		if options.InstalledPath != "" {
			endpoint = options.InstalledPath
		}
	}
	if versionKey == "CFBundleVersion" {
		version = app.Build
	}
	if version != "" {
		d.put("version", version, "app."+versionKey)
	}
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
		if options != nil || kind == "copy_from_dmg" {
			return errors.New("selected application has no known installed path; author derive.app.installed_path or installs")
		}
		return nil
	}
	if app.BundleID == "" || version == "" {
		return errors.New("selected application requires a bundle identifier and comparison version")
	}
	d.put("installs", []map[string]any{{"type": "application", "path": endpoint, "CFBundleIdentifier": app.BundleID, "CFBundleName": app.Name, "CFBundleShortVersionString": app.Version, "CFBundleVersion": app.Build, "version_comparison_key": versionKey, "minosversion": app.MinimumOS}}, "app.installed_path")
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
	var copies []packages.PackageItemToCopy
	if err := json.Unmarshal(raw(d.values["items_to_copy"]), &copies); err != nil {
		return "", err
	}
	var copied string
	for _, action := range copies {
		if path.Clean(action.SourceItem) != path.Clean(source) {
			continue
		}
		if copied != "" {
			return "", errors.New("selected app has multiple copy destinations; author installs explicitly")
		}
		item := action.DestinationItem
		if item == "" {
			item = path.Base(action.SourceItem)
		}
		copied = path.Join(action.DestinationPath, item)
	}
	if endpoint != "" && copied != "" && endpoint != copied {
		return "", errors.New("derive.app.installed_path disagrees with items_to_copy")
	}
	if copied != "" {
		return copied, nil
	}
	return endpoint, nil
}

func safeLocation(value string) bool {
	clean := path.Clean(value)
	return value != "" && !path.IsAbs(value) && clean != "." && clean != ".." && !strings.HasPrefix(clean, "../") && !strings.ContainsAny(value, "\\\x00")
}
