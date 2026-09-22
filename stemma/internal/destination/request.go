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
	"github.com/woodleighschool/woodstar/stemma/internal/destination/api"
	"github.com/woodleighschool/woodstar/stemma/internal/destination/pkginfo"
)

// controls separates native pkginfo from provider-owned targets and retention.
type controls struct {
	targets *targets

	Targets   json.RawMessage   `json:"targets,omitempty" jsonschema_description:"Assign software to labels by exact name. Omitted lists remain unchanged; empty lists clear their assignments."`
	Pkginfo   json.RawMessage   `json:"pkginfo,omitempty" jsonschema_description:"Native Munki pkginfo imported into software and packages. Only supported writable fields are accepted."`
	Retention *plugin.Retention `json:"retention,omitempty" jsonschema_description:"Prune older versions of this software after publication. Referenced packages remain protected."`
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
	links     map[string][]pkginfo.ResourceRelationship
	peers     map[string]json.RawMessage
	icon      plugin.Artifact
	installer plugin.Artifact
	controls  controls
	origins   map[string]string
}

func readRequest(ctx context.Context, request plugin.ReconcileRequest[api.Config]) (api.Config, metadata, error) {
	cfg := request.Config
	settings, err := decodeControls(request.Metadata)
	if err != nil {
		return cfg, metadata{}, err
	}
	derived, err := pkginfo.Derive(request, settings.Pkginfo)
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
	imported, err := pkginfo.Import(derived.Values, request.Identity.Resource)
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
	declared, err := object(metadata.Pkginfo)
	if err != nil {
		return metadata, err
	}
	if _, exists := declared["minimum_os_version"]; exists {
		return metadata, errors.New("pkginfo.minimum_os_version derives from the software; set minimum_os")
	}
	if metadata.targets, err = decodeTargets(metadata.Targets); err != nil {
		return metadata, err
	}
	if metadata.Retention != nil {
		if err := metadata.Retention.Validate(); err != nil {
			return metadata, err
		}
	}
	return metadata, nil
}

func metadataSchema() *jsonschema.Schema {
	r := jsonschema.Reflector{DoNotReference: true, RequiredFromJSONSchemaTags: true}
	schema := r.Reflect(controls{})
	schema.ID = ""
	native := pkginfo.Schema()
	field, _ := schema.Properties.Get("pkginfo")
	native.Description = field.Description
	schema.Properties.Set("pkginfo", native)
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
	field, _ = schema.Properties.Get("targets")
	declared.Description = field.Description
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
