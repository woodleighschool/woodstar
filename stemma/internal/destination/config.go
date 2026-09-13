package destination

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/url"
	"strings"

	"github.com/woodleighschool/stemma/plugin"

	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/munki/software"
)

type config struct {
	URL           string `json:"url"`
	APIKey        string `json:"api_key"`
	CAFile        string `json:"ca_file,omitempty"`
	Name          string `json:"-"`
	Version       string `json:"-"`
	InstallerType string `json:"-"`
}

type metadata struct {
	software     software.Patch
	pkg          packages.Patch
	requires     []munki.PkginfoReference
	updateFor    []munki.PkginfoReference
	icon         plugin.Artifact
	refreshIcons bool
	installer    plugin.Artifact
	controls     controls
	origins      map[string]string
}

func readRequest(ctx context.Context, request plugin.ReconcileRequest) (config, metadata, error) {
	var cfg config
	if err := decode(request.Config, &cfg); err != nil {
		return cfg, metadata{}, fmt.Errorf("configuration: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return cfg, metadata{}, err
	}
	settings, err := decodeControls(request.Metadata)
	if err != nil {
		return cfg, metadata{}, err
	}
	if request.Artifact.Format == "json" {
		document, err := readPkginfo(ctx, request.Artifact)
		if err != nil {
			return cfg, metadata{}, err
		}
		fields, err := object(document)
		if err != nil {
			return cfg, metadata{}, err
		}
		overrides, _ := object(settings.Pkginfo)
		maps.Copy(fields, overrides)
		settings.Pkginfo = raw(fields)
		request.Artifact = request.Inputs["installer"]
		request.Facts = request.Artifact.Facts
		request.Prepared = true
		request.Metadata = raw(settings)
	}
	values, origins, err := derive(request)
	if err != nil {
		return cfg, metadata{}, err
	}
	static := !request.Prepared && request.Artifact.Path == ""
	if static {
		if _, ok := values["name"]; !ok {
			values["name"] = "PendingSoftware"
		}
		if _, ok := values["version"]; !ok {
			values["version"] = "0"
		}
	}
	imported, err := importPkginfo(values, settings.Targets)
	if err != nil {
		return cfg, metadata{}, err
	}
	identity, err := imported.Package.Apply(packages.PackageMutation{})
	if err != nil {
		return cfg, metadata{}, err
	}
	identity.Normalize()
	cfg.Name, cfg.Version, cfg.InstallerType = imported.Name, identity.Version, string(identity.InstallerType)
	if !static {
		if err := validateInstaller(ctx, request, &imported, identity.InstallerType); err != nil {
			return cfg, metadata{}, err
		}
	}
	icon := request.Inputs["icon"]
	if !static && icon.Path != "" {
		if err := validateIcon(ctx, icon); err != nil {
			return cfg, metadata{}, fmt.Errorf("icon: %w", err)
		}
		origins["software.icon"] = "input.icon"
	}
	return cfg, metadata{icon: icon, refreshIcons: request.RefreshIcons, installer: request.Artifact, controls: settings, origins: origins, software: imported.Software, pkg: imported.Package, requires: imported.Requires, updateFor: imported.UpdateFor}, nil
}

func validateInstaller(ctx context.Context, request plugin.ReconcileRequest, imported *munki.PkginfoImport, installerType packages.InstallerType) error {
	if installerType == packages.InstallerTypeNoPkg {
		if request.Artifact.Path != "" || request.Artifact.SHA256 != "" {
			return errors.New("nopkg must not include installer content")
		}
		fields, _ := object(imported.Package.Bytes())
		fields["installer_object_id"] = raw(nil)
		return json.Unmarshal(raw(fields), &imported.Package)
	}
	installer := request.Artifact
	if installer.Path == "" {
		return errors.New("installer is required")
	}
	if imported.InstallerItemHash != "" && imported.InstallerItemHash != installer.SHA256 {
		return errors.New("pkginfo installer_item_hash does not match installer")
	}
	if err := verifyArtifact(ctx, installer, nil); err != nil {
		return fmt.Errorf("installer: %w", err)
	}
	return nil
}

func (cfg *config) validate() error {
	parsed, err := url.Parse(cfg.URL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return errors.New("url must be an HTTPS origin without credentials, query or path")
	}
	if strings.TrimSpace(cfg.APIKey) == "" || strings.ContainsAny(cfg.APIKey, "\r\n\x00") {
		return errors.New("api_key is required")
	}
	return nil
}

func object(data json.RawMessage) (map[string]json.RawMessage, error) {
	if len(data) == 0 {
		return map[string]json.RawMessage{}, nil
	}
	var fields map[string]json.RawMessage
	if err := decode(data, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

func decode(data json.RawMessage, target any) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return errors.New("null is not allowed")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("expected a single JSON value")
	}
	return nil
}

func raw(value any) json.RawMessage {
	data, _ := json.Marshal(value)
	return data
}

func importPkginfo(values map[string]any, targets json.RawMessage) (munki.PkginfoImport, error) {
	imported, err := munki.ImportPkginfo(raw(values))
	if err != nil {
		return munki.PkginfoImport{}, fmt.Errorf("pkginfo: %w", err)
	}
	if len(targets) > 0 {
		fields, err := object(imported.Software.Bytes())
		if err != nil {
			return munki.PkginfoImport{}, err
		}
		fields["targets"] = targets
		if err := json.Unmarshal(raw(fields), &imported.Software); err != nil {
			return munki.PkginfoImport{}, fmt.Errorf("targets: %w", err)
		}
	}
	return imported, nil
}
