package destination

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/woodleighschool/stemma/plugin"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
)

// controls separates native pkginfo from provider-owned derivation and retention.
type controls struct {
	Targets   json.RawMessage   `json:"targets,omitempty"`
	Unmanaged []string          `json:"unmanaged,omitempty" jsonschema:"description=Native pkginfo fields whose existing values remain unmanaged and are not derived."`
	Derive    Derivation        `json:"derive,omitzero"`
	Pkginfo   json.RawMessage   `json:"pkginfo,omitempty"`
	Retention *plugin.Retention `json:"retention,omitempty"`
}

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

// decodeControls validates authored fields without requiring prepared artifacts.
func decodeControls(data json.RawMessage) (controls, error) {
	var metadata controls
	if len(data) == 0 {
		data = json.RawMessage(`{}`)
	}
	if err := decode(data, &metadata); err != nil {
		return metadata, fmt.Errorf("metadata: %w", err)
	}
	if len(metadata.Pkginfo) == 0 {
		metadata.Pkginfo = json.RawMessage(`{}`)
	}
	if _, err := object(metadata.Pkginfo); err != nil {
		return metadata, err
	}
	var explicit map[string]any
	_ = json.Unmarshal(metadata.Pkginfo, &explicit)
	seen := map[string]bool{}
	schema := pkginfoSchema()
	for _, field := range metadata.Unmanaged {
		key, ok := strings.CutPrefix(field, "pkginfo.")
		if !ok {
			return metadata, errors.New("unmanaged fields must use pkginfo.<field>")
		}
		if _, exists := schema.Properties.Get(key); !exists {
			return metadata, fmt.Errorf("unsupported unmanaged field %q", field)
		}
		if key == "name" || key == "version" || key == "installer_type" {
			return metadata, fmt.Errorf("publication identity field %s cannot be unmanaged", field)
		}
		if _, exists := explicit[key]; exists {
			return metadata, fmt.Errorf("%s cannot be both authored and unmanaged", field)
		}
		if seen[field] {
			return metadata, fmt.Errorf("duplicate unmanaged field %q", field)
		}
		seen[field] = true
	}
	if metadata.Retention != nil && metadata.Retention.Keep < 1 {
		return metadata, errors.New("retention.keep must be at least 1")
	}
	if app := metadata.Derive.App; app != nil {
		if app.Subject == "" {
			return metadata, errors.New("derive.app.subject is required")
		}
		if app.InstalledPath != "" && !path.IsAbs(app.InstalledPath) {
			return metadata, errors.New("derive.app.installed_path must be absolute")
		}
		if app.VersionKey != "" && app.VersionKey != "CFBundleVersion" && app.VersionKey != "CFBundleShortVersionString" {
			return metadata, errors.New("unsupported derive.app.version_key")
		}
	}
	return metadata, nil
}

type nativeDefaults struct {
	values    map[string]any
	explicit  map[string]any
	origins   map[string]string
	unmanaged []string
}

func (d *nativeDefaults) put(key string, value any, origin string) {
	if _, exists := d.explicit[key]; exists || slices.Contains(d.unmanaged, "pkginfo."+key) {
		return
	}
	d.values[key], d.origins["pkginfo."+key] = value, origin
}

// Authored native values win over observations. Archive paths become endpoint
// paths only when a copy action or an installed-path selector establishes them.
func derive(request plugin.ReconcileRequest) (map[string]any, map[string]string, error) {
	metadata, err := decodeControls(request.Metadata)
	if err != nil {
		return nil, nil, err
	}
	var explicit map[string]any
	if err := json.Unmarshal(metadata.Pkginfo, &explicit); err != nil {
		return nil, nil, err
	}
	d := nativeDefaults{values: maps.Clone(explicit), explicit: explicit, origins: map[string]string{}, unmanaged: metadata.Unmanaged}
	for key := range explicit {
		d.origins["pkginfo."+key] = "authored"
	}
	if !request.Prepared && request.Artifact.Path == "" {
		return d.values, d.origins, nil
	}
	d.put("name", request.Identity.Software, "software.name")
	if request.Artifact.Version != "" {
		d.put("version", request.Artifact.Version, "installer.version")
	}
	kind := d.installerType(request.Artifact)
	if kind == "nopkg" {
		if request.Artifact.Path != "" || request.Artifact.SHA256 != "" {
			return nil, nil, errors.New("nopkg must not include installer content")
		}
		return d.values, d.origins, nil
	}
	facts := request.Facts
	if len(facts.Subjects) == 0 {
		facts = request.Artifact.Facts
	}
	versions := d.receipts(facts, kind)
	selected, versionKey, err := macEvidence(request.Artifact)
	if err != nil {
		return nil, nil, err
	}
	options := metadata.Derive.App
	if options != nil || selected == nil {
		selected, err = selectApplication(facts, request.Subjects, options)
	} else {
		options = &AppDerivation{VersionKey: versionKey}
	}
	if err != nil {
		return nil, nil, err
	}
	if selected != nil {
		if err := d.application(*selected, options, kind); err != nil {
			return nil, nil, err
		}
	} else if len(versions) == 1 {
		for version := range versions {
			d.put("version", version, "installer.receipts")
		}
	}
	return d.values, d.origins, nil
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

func selectApplication(facts plugin.Facts, selectors map[string]plugin.SubjectSelector, options *AppDerivation) (*plugin.Subject, error) {
	if options != nil {
		selector, exists := selectors[options.Subject]
		if !exists {
			return nil, fmt.Errorf("derive.app references unknown subject %q", options.Subject)
		}
		subject, err := plugin.SelectSubject(facts, selector)
		if err != nil {
			return nil, err
		}
		if subject.App == nil {
			return nil, errors.New("derive.app requires an application subject")
		}
		return &subject, nil
	}
	var selected *plugin.Subject
	for _, subject := range facts.Subjects {
		if subject.App == nil {
			continue
		}
		if selected != nil {
			return nil, nil
		}
		selected = &subject
	}
	return selected, nil
}

func (d *nativeDefaults) receipts(facts plugin.Facts, kind string) map[string]bool {
	var receipts []map[string]any
	versions := map[string]bool{}
	for _, subject := range facts.Subjects {
		pkg := subject.Package
		if pkg == nil {
			continue
		}
		if pkg.Version != "" {
			versions[pkg.Version] = true
		}
		if pkg.HasPayload && pkg.Identifier != "" {
			receipts = append(receipts, map[string]any{"packageid": pkg.Identifier, "version": pkg.Version, "installed_size": pkg.InstalledSize})
		}
	}
	if len(receipts) > 0 && kind == "pkg" {
		d.put("receipts", receipts, "installer.receipts")
	}
	return versions
}

func (d *nativeDefaults) application(subject plugin.Subject, options *AppDerivation, kind string) error {
	app := subject.App
	version, versionKey, endpoint := app.Version, "CFBundleShortVersionString", subject.InstalledPath
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
	if app.MinimumOS != "" {
		d.put("minimum_os_version", app.MinimumOS, "app.minimum_os")
	}
	for _, key := range []string{"installcheck_script", "receipts", "installs"} {
		if _, authored := d.explicit[key]; authored {
			return nil
		}
	}
	if slices.Contains(d.unmanaged, "pkginfo.installs") {
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
	d.put("installs", []map[string]any{{"type": "application", "path": endpoint, "CFBundleIdentifier": app.BundleID, "CFBundleName": app.Name, "CFBundleShortVersionString": app.Version, "CFBundleVersion": app.Build, "version_comparison_key": versionKey, "minimum_os_version": app.MinimumOS}}, "app.installed_path")
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
