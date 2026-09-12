package destination

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/woodleighschool/stemma/plugin"
)

func (remote *client) manageIcon(metadata *metadata, observed observation) error {
	owned := remote.state.Icon
	if owned == nil {
		return nil
	}
	if observed.Software != nil && observed.Software.IconObjectID != nil && *observed.Software.IconObjectID == owned.ObjectID && observed.Software.IconFile != nil && observed.Software.IconFile.SHA256 == owned.SHA256 && observed.Software.IconFile.SizeBytes == owned.Size {
		owned.Complete = true
	}
	if metadata.icon.Path != "" {
		return nil
	}
	if !owned.Complete {
		return errors.New("pending icon publication requires its original icon input")
	}
	if observed.Software == nil || observed.Software.IconObjectID == nil || *observed.Software.IconObjectID != owned.ObjectID {
		return nil
	}
	fields, err := object(metadata.software.Bytes())
	if err != nil {
		return err
	}
	fields["icon_object_id"] = raw(nil)
	return json.Unmarshal(raw(fields), &metadata.software)
}

func (remote *client) saveIcon(ctx context.Context, artifact plugin.Artifact, observed *observation) error {
	var data bytes.Buffer
	if err := verifyArtifact(ctx, artifact, &data); err != nil {
		return fmt.Errorf("icon: %w", err)
	}
	pending := remote.state.Icon
	if pending != nil && (pending.SHA256 != artifact.SHA256 || pending.Size != artifact.Size) {
		if !pending.Complete {
			return errors.New("pending icon belongs to different content; finish its publication first")
		}
		pending = nil
	}
	if pending == nil {
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
		pending = &pendingUpload{ObjectID: upload.ObjectID, SHA256: artifact.SHA256, Size: artifact.Size}
		remote.state.Icon = pending
		if _, err := remote.transferBytes(ctx, upload.Upload.Target, bytes.NewReader(data.Bytes()), artifact.Size); err != nil {
			// Keep the object ID: the transfer may have completed without a reply.
			return err
		}
	}
	var attached struct {
		ID        int64  `json:"id"`
		SHA256    string `json:"sha256"`
		SizeBytes int64  `json:"size_bytes"`
	}
	endpoint := "/api/munki/software/" + strconv.FormatInt(observed.Software.ID, 10) + "/icon"
	if err := remote.request(ctx, http.MethodPut, endpoint, map[string]int64{"object_id": pending.ObjectID}, &attached); err != nil {
		// The native attach finalizes the owned upload and cleans missing or
		// invalid bytes. Unknown outcomes retain the ID for readback and retry.
		var status httpError
		if errors.As(err, &status) && (status.status == http.StatusBadRequest || status.status == http.StatusNotFound) {
			remote.state.Icon = nil
		}
		return err
	}
	if attached.ID != pending.ObjectID || attached.SHA256 != artifact.SHA256 || attached.SizeBytes != artifact.Size {
		return errors.New("attached icon differs from the prepared content")
	}
	pending.Complete = true
	return nil
}

func validateIcon(ctx context.Context, artifact plugin.Artifact) error {
	if artifact.Format != "png" || artifact.Size <= 0 || artifact.Size > 32<<20 {
		return errors.New("icon input requires a PNG artifact no larger than 32 MiB")
	}
	return verifyArtifact(ctx, artifact, nil)
}
