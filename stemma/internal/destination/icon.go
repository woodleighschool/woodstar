package destination

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/woodleighschool/stemma/plugin"
)

func (remote *client) recoverIcon(observed observation) {
	pending := remote.state.Icon
	if pending != nil && observed.Software != nil && observed.Software.IconObjectID != nil && *observed.Software.IconObjectID == pending.ObjectID && observed.Software.IconFile != nil && observed.Software.IconFile.SHA256 == pending.SHA256 && observed.Software.IconFile.SizeBytes == pending.Size {
		remote.state.Icon = nil
	}
}

func (remote *client) saveIcon(ctx context.Context, artifact plugin.Artifact, observed *observation) (runErr error) {
	done := plugin.Stage(ctx, "Publishing icon")
	defer func() { done(runErr) }()
	var data bytes.Buffer
	if err := verifyArtifact(ctx, artifact, &data); err != nil {
		return fmt.Errorf("icon: %w", err)
	}
	softwareID := observed.Software.ID
	if pending := remote.state.Icon; pending != nil {
		err := remote.attachIcon(ctx, softwareID, pending)
		if err != nil && remote.state.Icon != nil {
			return err
		}
		if err == nil && pending.SHA256 == artifact.SHA256 && pending.Size == artifact.Size {
			return nil
		}
	}
	var upload struct {
		ObjectID int64 `json:"object_id"`
		Upload   struct {
			Strategy string       `json:"strategy"`
			Target   uploadTarget `json:"target"`
		} `json:"upload"`
	}
	if err := remote.request(ctx, http.MethodPost, "/api/munki/icons", map[string]string{"filename": artifact.Filename}, &upload); err != nil {
		return err
	}
	if upload.ObjectID <= 0 || upload.Upload.Strategy != "direct-put" {
		return errors.New("icon upload requires an owned object and a direct PUT target")
	}
	pending := &pendingUpload{ObjectID: upload.ObjectID, SHA256: artifact.SHA256, Size: artifact.Size}
	remote.state.Icon = pending
	if _, err := remote.transferBytes(ctx, upload.Upload.Target, bytes.NewReader(data.Bytes()), artifact.Size); err != nil {
		// The transfer may have completed without a reply; attaching resolves it.
		return err
	}
	return remote.attachIcon(ctx, softwareID, pending)
}

func (remote *client) attachIcon(ctx context.Context, softwareID int64, pending *pendingUpload) error {
	var attached struct {
		ID        int64  `json:"id"`
		SHA256    string `json:"sha256"`
		SizeBytes int64  `json:"size_bytes"`
	}
	endpoint := "/api/munki/software/" + strconv.FormatInt(softwareID, 10) + "/icon"
	if err := remote.request(ctx, http.MethodPut, endpoint, map[string]int64{"object_id": pending.ObjectID}, &attached); err != nil {
		// The native attach cleans invalid or missing bytes. Unknown outcomes
		// retain their object ID so a retry never needs the original local PNG.
		var status httpError
		if errors.As(err, &status) && (status.status == http.StatusBadRequest || status.status == http.StatusNotFound) {
			remote.state.Icon = nil
		}
		return err
	}
	if attached.ID != pending.ObjectID || attached.SHA256 != pending.SHA256 || attached.SizeBytes != pending.Size {
		return errors.New("attached icon differs from the prepared content")
	}
	remote.state.Icon = nil
	return nil
}

func validateIcon(ctx context.Context, artifact plugin.Artifact) error {
	if artifact.Format != "png" || artifact.Size <= 0 || artifact.Size > 32<<20 {
		return errors.New("icon input requires a PNG artifact no larger than 32 MiB")
	}
	return verifyArtifact(ctx, artifact, nil)
}
