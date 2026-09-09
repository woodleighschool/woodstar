// Package destination publishes prepared installers through the administrative API.
package destination

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/woodleighschool/stemma/plugin"

	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/munki/software"
)

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
		return plugin.ReconcileResponse{}, nil
	}
	remote, err := newClient(cfg)
	if err != nil {
		return plugin.ReconcileResponse{}, err
	}
	defer func() { _ = remote.api.Close(); _ = remote.transfer.Close() }()
	observed, err := remote.observe(ctx, request.Binding)
	if err != nil {
		return remote.response(observed, nil), err
	}
	if err := remote.resolveReferences(ctx, &metadata); err != nil {
		return remote.response(observed, nil), err
	}
	installer := request.Inputs["installer"]
	plan, err := remote.plan(installer, metadata, observed)
	if err != nil {
		return remote.response(observed, nil), err
	}
	if request.Method == "apply" && len(plan.changes) != 0 {
		if err := remote.apply(ctx, installer, metadata, plan, &observed); err != nil {
			return remote.response(observed, nil), err
		}
	}
	return remote.response(observed, plan.changes), nil
}

func (remote *client) apply(ctx context.Context, artifact plugin.Artifact, metadata metadata, plan desired, observed *observation) error {
	var pendingObject int64
	if plan.content {
		object, err := remote.upload(ctx, artifact)
		if err != nil {
			return err
		}
		pendingObject = object
		defer func() {
			if pendingObject != 0 {
				remote.cleanupUpload(ctx, pendingObject)
			}
		}()
	}
	if observed.Software == nil {
		if err := remote.createSoftware(ctx, observed); err != nil {
			return err
		}
	}
	if observed.Package == nil || changed(plan.changes, "package") {
		if err := remote.savePackage(ctx, artifact, metadata, plan, pendingObject, observed); err != nil {
			return err
		}
	}
	if observed.Package != nil && observed.Package.InstallerObjectID != nil && *observed.Package.InstallerObjectID == pendingObject {
		pendingObject = 0
	}
	if changed(plan.changes, "software") {
		if err := remote.saveSoftware(ctx, artifact, metadata, observed); err != nil {
			return err
		}
	}
	readback, err := remote.observe(ctx, nil)
	if err != nil {
		return err
	}
	*observed = readback
	remaining, err := remote.plan(artifact, metadata, readback)
	if err != nil {
		return err
	}
	if len(remaining.changes) != 0 {
		return errors.New("destination changed during publication; re-plan before retrying")
	}
	return nil
}

func (remote *client) createSoftware(ctx context.Context, observed *observation) error {
	// Targets can select a package that does not exist until the next write.
	body := software.CreateMutation{Name: remote.config.Name}
	body.Normalize()
	var created softwareDetail
	if err := remote.request(ctx, http.MethodPost, "/api/munki/software", body, &created); err != nil {
		// Exact-name discovery also recovers a committed create whose reply was lost.
		recovered, recoveryErr := remote.observe(ctx, nil)
		if recoveryErr != nil || recovered.Software == nil {
			return err
		}
		*observed = recovered
		return nil
	}
	if created.ID <= 0 || created.Name != remote.config.Name {
		return errors.New("created software does not match its requested identity")
	}
	observed.Software = &created
	return nil
}

func (remote *client) savePackage(ctx context.Context, artifact plugin.Artifact, metadata metadata, plan desired, objectID int64, observed *observation) error {
	method, endpoint := http.MethodPost, "/api/munki/packages"
	var body any
	if observed.Package == nil {
		mutation := plan.pkg
		if objectID != 0 {
			mutation.InstallerObjectID = &objectID
		}
		body = packages.PackageCreateMutation{PackageMutation: mutation, SoftwareID: observed.Software.ID}
	} else {
		fields := map[string]json.RawMessage{}
		if err := decode(metadata.pkg.Bytes(), &fields); err != nil {
			return err
		}
		if objectID != 0 {
			fields["installer_object_id"] = raw(objectID)
		}
		body = fields
		method, endpoint = http.MethodPatch, "/api/munki/packages/"+strconv.FormatInt(observed.Package.ID, 10)
	}
	var saved packages.Package
	if err := remote.request(ctx, method, endpoint, body, &saved); err != nil {
		return remote.recoverWrite(ctx, artifact, metadata, "package", observed, err)
	}
	observed.Package = &saved
	return nil
}

func (remote *client) saveSoftware(ctx context.Context, artifact plugin.Artifact, metadata metadata, observed *observation) error {
	var saved softwareDetail
	endpoint := "/api/munki/software/" + strconv.FormatInt(observed.Software.ID, 10)
	if err := remote.request(ctx, http.MethodPatch, endpoint, metadata.software, &saved); err != nil {
		return remote.recoverWrite(ctx, artifact, metadata, "software", observed, err)
	}
	observed.Software = &saved
	return nil
}

func (remote *client) recoverWrite(ctx context.Context, artifact plugin.Artifact, metadata metadata, resource string, observed *observation, writeErr error) error {
	recovered, err := remote.observe(ctx, nil)
	if err != nil {
		return writeErr
	}
	*observed = recovered
	remaining, err := remote.plan(artifact, metadata, recovered)
	if err != nil || changed(remaining.changes, resource) {
		return writeErr
	}
	return nil
}
