// Package destination publishes prepared installers through the administrative API.
package destination

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/invopop/jsonschema"
	"github.com/woodleighschool/stemma/plugin"

	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/stemma/internal/destination/api"
)

// Register exposes the transport through Stemma's shared operation registry.
func Register(registry *plugin.Registry) error {
	reflector := jsonschema.Reflector{DoNotReference: true}
	return registry.Register(plugin.Operation{
		Name: "woodstar.munki", Kind: "reconcile", SideEffects: "remote", Methods: []string{"validate", "plan", "apply"},
		RequiresInspection: true,
		Content:            &plugin.ContentContract{Formats: []string{"pkg", "dmg"}, SourceFree: true},
		ConfigSchema:       json.RawMessage(`{"type":"object","properties":{"url":{"type":"string","pattern":"^https://"},"api_key":{"type":"string","minLength":1,"writeOnly":true},"ca_file":{"type":"string"}},"required":["url","api_key"],"additionalProperties":false}`),
		MetadataSchema:     raw(metadataSchema()),
		InputSchema:        raw(reflector.Reflect(plugin.ReconcileRequest{})),
		OutputSchema:       raw(reflector.Reflect(plugin.ReconcileResponse{})),
	}, func(ctx context.Context, envelope plugin.Request) (plugin.Response, error) {
		var request plugin.ReconcileRequest
		if err := decode(envelope.Input, &request); err != nil {
			return plugin.Response{}, err
		}
		request.Method = envelope.Method
		result, err := Handle(ctx, request)
		return plugin.Response{Output: raw(result)}, err
	})
}

// Handle plans without writes or applies the fields supplied by a destination.
func Handle(ctx context.Context, request plugin.ReconcileRequest) (plugin.ReconcileResponse, error) {
	if request.Method != "validate" && request.Method != "plan" && request.Method != "apply" {
		return plugin.ReconcileResponse{}, fmt.Errorf("unsupported method %q", request.Method)
	}
	cfg, metadata, err := readRequest(ctx, request)
	if err != nil {
		return plugin.ReconcileResponse{}, err
	}
	if request.Method == "validate" {
		return plugin.ReconcileResponse{Origins: metadata.origins, Requires: metadata.references()}, nil
	}
	remote, err := api.New(cfg)
	if err != nil {
		return plugin.ReconcileResponse{}, err
	}
	defer func() { _ = remote.Close() }()
	response := plugin.ReconcileResponse{Origins: metadata.origins}
	done := plugin.Stage(ctx, "Observing destination")
	observed, err := remote.Observe(ctx, metadata.name, metadata.version)
	done(err)
	if err != nil {
		return response, err
	}
	if err := resolveReferences(ctx, remote, &metadata); err != nil {
		return response, err
	}
	if err := resolveTargets(ctx, remote, &metadata); err != nil {
		return response, err
	}
	planned, err := plan(metadata, observed)
	if err != nil {
		return response, err
	}
	response.Changes = planned.changes
	if request.Method == "apply" && len(planned.changes) != 0 {
		if err := apply(ctx, remote, metadata, planned, &observed); err != nil {
			return response, err
		}
	}
	if request.Method == "plan" && observed.Software != nil {
		prospective := *observed.Software
		prospective.Targets = planned.targets
		observed.Software = &prospective
	}
	pruned, err := prune(ctx, remote, metadata, observed, request.Method == "apply")
	response.Changes = append(response.Changes, pruned...)
	return response, err
}

func apply(ctx context.Context, remote *api.Client, metadata metadata, planned desired, observed *api.Observation) (runErr error) {
	if observed.Software == nil {
		// Targets can select a package that does not exist until the next write,
		// so the title starts with only its name.
		created, err := remote.CreateSoftware(ctx, metadata.name)
		if err != nil {
			return err
		}
		observed.Software = created
	}
	var uploaded int64
	if planned.content {
		id, err := remote.Upload(ctx, metadata.installer)
		if err != nil {
			return err
		}
		uploaded = id
	}
	if observed.Package == nil || planned.content || metadataChanged(planned.changes, "package") {
		if err := savePackage(ctx, remote, metadata, planned, uploaded, observed); err != nil {
			// The repository refuses to delete an installer a package references.
			if uploaded != 0 {
				remote.ReleaseUpload(ctx, uploaded)
			}
			return err
		}
	}
	if metadataChanged(planned.changes, "software") {
		if err := saveSoftware(ctx, remote, metadata, observed); err != nil {
			return err
		}
	}
	if planned.icon {
		if err := publishIcon(ctx, remote, metadata.icon, observed.Software.ID); err != nil {
			return err
		}
	}
	done := plugin.Stage(ctx, "Verifying publication")
	defer func() { done(runErr) }()
	readback, err := remote.Observe(ctx, metadata.name, metadata.version)
	if err != nil {
		return err
	}
	*observed = readback
	remaining, err := plan(metadata, readback)
	if err != nil {
		return err
	}
	if len(remaining.changes) != 0 {
		return errors.New("destination changed during publication; re-plan before retrying")
	}
	return nil
}

// savePackage creates the package with its whole desired state, or patches the
// fields the declaration supplies onto the package the repository holds.
func savePackage(ctx context.Context, remote *api.Client, metadata metadata, planned desired, objectID int64, observed *api.Observation) (runErr error) {
	done := plugin.Stage(ctx, "Saving package")
	defer func() { done(runErr) }()
	var saved *packages.Package
	var writeErr error
	if observed.Package == nil {
		mutation := planned.pkg
		if objectID != 0 {
			mutation.InstallerObjectID = &objectID
		}
		saved, writeErr = remote.CreatePackage(ctx, packages.PackageCreateMutation{PackageMutation: mutation, SoftwareID: observed.Software.ID})
	} else {
		patch, err := withInstaller(metadata.pkg, objectID)
		if err != nil {
			return err
		}
		saved, writeErr = remote.UpdatePackage(ctx, observed.Package.ID, patch)
	}
	if writeErr != nil {
		if err := recoverWrite(ctx, remote, metadata, "package", observed, writeErr); err != nil {
			return err
		}
		saved = observed.Package
	}
	artifact := metadata.installer
	if saved.ID <= 0 || saved.Software.ID != observed.Software.ID || saved.Version != metadata.version || (artifact.SHA256 != "" && (saved.InstallerFile == nil || saved.InstallerFile.SHA256 != artifact.SHA256)) {
		return errors.New("saved package does not match the intended identity and installer")
	}
	observed.Package = saved
	return nil
}

// withInstaller points a package patch at a newly uploaded installer.
func withInstaller(patch packages.Patch, objectID int64) (packages.Patch, error) {
	if objectID == 0 {
		return patch, nil
	}
	fields, err := object(patch.Bytes())
	if err != nil {
		return patch, err
	}
	fields["installer_object_id"] = raw(objectID)
	err = json.Unmarshal(raw(fields), &patch)
	return patch, err
}

func saveSoftware(ctx context.Context, remote *api.Client, metadata metadata, observed *api.Observation) (runErr error) {
	done := plugin.Stage(ctx, "Saving software")
	defer func() { done(runErr) }()
	saved, err := remote.UpdateSoftware(ctx, observed.Software.ID, metadata.software)
	if err != nil {
		return recoverWrite(ctx, remote, metadata, "software", observed, err)
	}
	observed.Software = saved
	return nil
}

func publishIcon(ctx context.Context, remote *api.Client, icon plugin.Artifact, softwareID int64) (runErr error) {
	done := plugin.Stage(ctx, "Publishing icon")
	defer func() { done(runErr) }()
	var content bytes.Buffer
	if err := verifyArtifact(ctx, icon, &content); err != nil {
		return fmt.Errorf("icon: %w", err)
	}
	return remote.SetIcon(ctx, softwareID, icon.Filename, content.Bytes())
}

// recoverWrite accepts a write whose reply was lost once the repository holds
// its result, which the publication's native identity finds again.
func recoverWrite(ctx context.Context, remote *api.Client, metadata metadata, resource string, observed *api.Observation, writeErr error) error {
	recovered, err := remote.Observe(ctx, metadata.name, metadata.version)
	if err != nil || recovered.Software == nil {
		return writeErr
	}
	remaining, err := plan(metadata, recovered)
	if err != nil || metadataChanged(remaining.changes, resource) || resource == "package" && (recovered.Package == nil || remaining.content) {
		return writeErr
	}
	*observed = recovered
	return nil
}
