package destination

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	software  software.Patch
	pkg       packages.Patch
	requires  []munki.PkginfoReference
	updateFor []munki.PkginfoReference
}

func readRequest(ctx context.Context, request plugin.ReconcileRequest) (config, metadata, error) {
	var cfg config
	if err := decode(request.Config, &cfg); err != nil {
		return cfg, metadata{}, fmt.Errorf("configuration: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return cfg, metadata{}, err
	}
	data, err := readPkginfo(ctx, request.Artifact)
	if err != nil {
		return cfg, metadata{}, err
	}
	imported, err := munki.ImportPkginfo(data)
	if err != nil {
		return cfg, metadata{}, fmt.Errorf("pkginfo: %w", err)
	}
	var controls struct {
		Targets json.RawMessage `json:"targets"`
	}
	if len(request.Metadata) > 0 {
		if err := decode(request.Metadata, &controls); err != nil {
			return cfg, metadata{}, fmt.Errorf("metadata: %w", err)
		}
	}
	if len(controls.Targets) > 0 {
		fields, err := object(imported.Software.Bytes())
		if err != nil {
			return cfg, metadata{}, err
		}
		fields["targets"] = controls.Targets
		if err := json.Unmarshal(raw(fields), &imported.Software); err != nil {
			return cfg, metadata{}, fmt.Errorf("targets: %w", err)
		}
	}
	identity, err := imported.Package.Apply(packages.PackageMutation{})
	if err != nil {
		return cfg, metadata{}, err
	}
	identity.Normalize()
	cfg.Name, cfg.Version, cfg.InstallerType = imported.Name, identity.Version, string(identity.InstallerType)
	if err := validateInstaller(ctx, request, &imported, identity.InstallerType); err != nil {
		return cfg, metadata{}, err
	}
	return cfg, metadata{software: imported.Software, pkg: imported.Package, requires: imported.Requires, updateFor: imported.UpdateFor}, nil
}

func validateInstaller(ctx context.Context, request plugin.ReconcileRequest, imported *munki.PkginfoImport, installerType packages.InstallerType) error {
	if installerType == packages.InstallerTypeNoPkg {
		fields, _ := object(imported.Package.Bytes())
		fields["installer_object_id"] = raw(nil)
		return json.Unmarshal(raw(fields), &imported.Package)
	}
	installer, ok := request.Inputs["installer"]
	if !ok {
		return errors.New("inputs.installer is required")
	}
	if imported.InstallerItemHash != "" && imported.InstallerItemHash != installer.SHA256 {
		return errors.New("pkginfo installer_item_hash does not match inputs.installer")
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
