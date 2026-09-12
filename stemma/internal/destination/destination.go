// Package destination publishes prepared installers through the administrative API.
package destination

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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
		return plugin.ReconcileResponse{Origins: metadata.origins}, nil
	}
	remote, err := newClient(cfg)
	if err != nil {
		return plugin.ReconcileResponse{}, err
	}
	defer func() { _ = remote.api.Close(); _ = remote.transfer.Close() }()
	remote.fingerprint, remote.origins = metadata.installer.SHA256, metadata.origins
	if cfg.InstallerType == "nopkg" {
		remote.fingerprint = "nopkg:" + cfg.Version
	}
	observed, err := remote.observe(ctx, request.Binding)
	if err != nil {
		return remote.response(observed, nil), err
	}
	if err := remote.resolveReferences(ctx, &metadata); err != nil {
		return remote.response(observed, nil), err
	}
	if err := remote.manageIcon(&metadata, observed); err != nil {
		return remote.response(observed, nil), err
	}
	if err := remote.manageDerived(&metadata); err != nil {
		return remote.response(observed, nil), err
	}
	installer := metadata.installer
	plan, err := remote.plan(installer, metadata, observed)
	if err != nil {
		return remote.response(observed, nil), err
	}
	if request.Method == "apply" && len(plan.changes) != 0 {
		if err := remote.apply(ctx, installer, metadata, plan, &observed); err != nil {
			return remote.response(observed, nil), err
		}
	}
	if request.Method == "apply" {
		if metadata.icon.Path == "" && (remote.state.Icon == nil || remote.state.Icon.Complete) {
			remote.state.Icon = nil
		}
		remote.recordPackage(metadata, observed.Package)
		remote.state.Publications.Record(remote.fingerprint)
	}
	if request.Method == "plan" && observed.Software != nil {
		prospective := *observed.Software
		prospective.Targets = plan.targets
		observed.Software = &prospective
	}
	pruned, err := remote.prune(ctx, metadata.controls.Retention, observed, request.Method == "apply")
	return remote.response(observed, append(plan.changes, pruned...)), err
}

func (remote *client) apply(ctx context.Context, artifact plugin.Artifact, metadata metadata, plan desired, observed *observation) error {
	if observed.Software == nil {
		if err := remote.createSoftware(ctx, observed); err != nil {
			return err
		}
	}
	var pendingObject int64
	if plan.content {
		object, err := remote.upload(ctx, artifact)
		if err != nil {
			return err
		}
		pendingObject = object
	}
	if observed.Package == nil || changed(plan.changes, "package") {
		if err := remote.savePackage(ctx, artifact, metadata, plan, pendingObject, observed); err != nil {
			return err
		}
	}
	if observed.Package != nil && observed.Package.InstallerObjectID != nil && *observed.Package.InstallerObjectID == pendingObject {
		remote.state.Upload = nil
	}
	if changed(plan.changes, "software") {
		if err := remote.saveSoftware(ctx, artifact, metadata, observed); err != nil {
			return err
		}
	}
	if plan.icon {
		if err := remote.saveIcon(ctx, metadata.icon, observed); err != nil {
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
	remote.state.Creating = true
	if err := remote.request(ctx, http.MethodPost, "/api/munki/software", body, &created); err != nil {
		remote.creationFailed(err)
		return err
	}
	if created.ID <= 0 || created.Name != remote.config.Name {
		return errors.New("created software does not match its requested identity")
	}
	remote.state.Creating = false
	remote.state.SoftwareID = created.ID
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
	if method == http.MethodPost {
		remote.state.Creating = true
	}
	if err := remote.request(ctx, method, endpoint, body, &saved); err != nil {
		if method == http.MethodPost {
			remote.creationFailed(err)
		}
		if recovered, recoveryErr := remote.recoverPackage(ctx, observed, objectID); recoveryErr == nil {
			saved = recovered
		} else {
			return err
		}
	}
	if saved.ID <= 0 || saved.Software.ID != observed.Software.ID || saved.Version != remote.config.Version || (artifact.SHA256 != "" && (saved.InstallerFile == nil || saved.InstallerFile.SHA256 != artifact.SHA256)) {
		return errors.New("saved package does not match the intended identity and installer")
	}
	remote.state.Creating = false
	observed.Package = &saved
	remote.recordPackage(metadata, &saved)
	return nil
}

func (remote *client) creationFailed(err error) {
	var status httpError
	if errors.As(err, &status) && status.status >= 400 && status.status < 500 && status.status != http.StatusRequestTimeout {
		remote.state.Creating = false
	}
}

func (remote *client) recoverPackage(ctx context.Context, observed *observation, objectID int64) (packages.Package, error) {
	var result packages.Package
	if observed.Package != nil {
		err := remote.request(ctx, http.MethodGet, "/api/munki/packages/"+strconv.FormatInt(observed.Package.ID, 10), nil, &result)
		return result, err
	}
	if objectID == 0 {
		return result, errors.New("uncertain package create has no owned installer evidence")
	}
	items, err := list[packages.Package](ctx, remote, "/api/munki/packages", url.Values{"software_id": {strconv.FormatInt(observed.Software.ID, 10)}, "q": {remote.config.Version}})
	if err != nil {
		return result, err
	}
	for _, item := range items {
		if item.Software.ID == observed.Software.ID && item.Version == remote.config.Version && item.InstallerObjectID != nil && *item.InstallerObjectID == objectID {
			if result.ID != 0 {
				return result, errors.New("ambiguous owned installer reference")
			}
			result = item
		}
	}
	if result.ID == 0 {
		return result, errors.New("owned installer has no package")
	}
	return result, nil
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
