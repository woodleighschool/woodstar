package destination

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"time"
)

// SetIcon uploads an icon to a new object and attaches it to the software in
// place of any icon it had.
func (c *Client) SetIcon(ctx context.Context, softwareID int64, filename string, content []byte) error {
	ctx, cancel := context.WithTimeout(ctx, time.Hour)
	defer cancel()
	var upload reservation
	if err := c.request(ctx, http.MethodPost, "/api/munki/icons", map[string]string{"filename": filename}, &upload); err != nil {
		return err
	}
	if upload.ObjectID <= 0 || upload.Upload.Strategy != "direct-put" {
		return errors.New("icon upload requires an owned object and a direct PUT target")
	}
	size := int64(len(content))
	if _, err := c.transferBytes(ctx, upload.Upload.Target, bytes.NewReader(content), size); err != nil {
		return err
	}
	var attached storedObject
	endpoint := "/api/munki/software/" + strconv.FormatInt(softwareID, 10) + "/icon"
	if err := c.request(ctx, http.MethodPut, endpoint, map[string]int64{"object_id": upload.ObjectID}, &attached); err != nil {
		return err
	}
	digest := sha256.Sum256(content)
	if attached.ID != upload.ObjectID || attached.SHA256 != hex.EncodeToString(digest[:]) || attached.SizeBytes != size {
		return errors.New("attached icon differs from the prepared content")
	}
	return nil
}
