package destination

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/invopop/jsonschema"
	"github.com/woodleighschool/stemma/plugin"

	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/munki/software"
)

// controls separates native pkginfo from provider-owned derivation and retention.
type controls struct {
	targets *targets

	Targets   json.RawMessage   `json:"targets,omitempty"`
	Derive    Derivation        `json:"derive,omitzero"`
	Pkginfo   json.RawMessage   `json:"pkginfo,omitempty"`
	Retention *plugin.Retention `json:"retention,omitempty"`
}

// metadata is a declaration resolved against its artifacts: the publication's
// native identity and the sparse changes it makes to the software and package.
type metadata struct {
	name      string
	version   string
	software  software.Patch
	pkg       packages.Patch
	requires  []munki.PkginfoReference
	updateFor []munki.PkginfoReference
	links     map[string][]CatalogReference
	peers     map[string]json.RawMessage
	icon      plugin.Artifact
	installer plugin.Artifact
	controls  controls
	origins   map[string]string
}

func readRequest(ctx context.Context, request plugin.ReconcileRequest) (Config, metadata, error) {
	var cfg Config
	if err := decode(request.Config, &cfg); err != nil {
		return cfg, metadata{}, fmt.Errorf("configuration: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return cfg, metadata{}, err
	}
	settings, err := decodeControls(request.Metadata)
	if err != nil {
		return cfg, metadata{}, err
	}
	derived, err := Derive(request, settings.Pkginfo, settings.Derive)
	if err != nil {
		return cfg, metadata{}, err
	}
	static := !request.Prepared && request.Artifact.Path == ""
	if static {
		if _, ok := derived.Values["name"]; !ok {
			derived.Values["name"] = "PendingSoftware"
		}
		if _, ok := derived.Values["version"]; !ok {
			derived.Values["version"] = "0"
		}
	}
	imported, err := Import(derived.Values, request.Identity.Software)
	if err != nil {
		return cfg, metadata{}, err
	}
	identity, err := imported.Package.Apply(packages.PackageMutation{})
	if err != nil {
		return cfg, metadata{}, err
	}
	identity.Normalize()
	if !static {
		if err := validateInstaller(ctx, request.Artifact, identity.InstallerType, imported.InstallerItemHash); err != nil {
			return cfg, metadata{}, err
		}
	}
	icon := request.Inputs["icon"]
	if !static && icon.Path != "" {
		if err := validateIcon(ctx, icon); err != nil {
			return cfg, metadata{}, fmt.Errorf("icon: %w", err)
		}
		derived.Origins["software.icon"] = "input.icon"
	}
	return cfg, metadata{
		name: imported.Name, version: identity.Version,
		software: imported.Software, pkg: imported.Package,
		requires: imported.Requires, updateFor: imported.UpdateFor, links: imported.Links,
		peers: request.Peers, icon: icon, installer: request.Artifact, controls: settings, origins: derived.Origins,
	}, nil
}

// decodeControls validates declared fields without requiring prepared artifacts.
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
	var err error
	if metadata.targets, err = decodeTargets(metadata.Targets); err != nil {
		return metadata, err
	}
	if metadata.Retention != nil {
		if err := metadata.Retention.Validate(); err != nil {
			return metadata, err
		}
	}
	return metadata, metadata.Derive.Validate()
}

func metadataSchema() *jsonschema.Schema {
	r := jsonschema.Reflector{DoNotReference: true, RequiredFromJSONSchemaTags: true}
	schema := r.Reflect(controls{})
	schema.ID = ""
	schema.Properties.Set("pkginfo", Schema())
	declared := r.Reflect(targets{})
	declared.ID = ""
	if entries, ok := declared.Properties.Get("include"); ok {
		if field, ok := entries.Items.Properties.Get("actions"); ok {
			field.Items = &jsonschema.Schema{Type: "string"}
			for _, action := range actions {
				field.Items.Enum = append(field.Items.Enum, string(action))
			}
		}
	}
	schema.Properties.Set("targets", declared)
	return schema
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
