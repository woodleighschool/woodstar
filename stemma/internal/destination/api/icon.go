package api

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/woodleighschool/goodies/bloby"
)

// SetIcon uploads an icon to a new object and attaches it to the software in
// place of any icon it had. Attaching publishes the object, which the
// repository refuses unless storage holds the declared bytes.
func (c *Client) SetIcon(ctx context.Context, softwareID int64, filename string, content []byte) error {
	ctx, cancel := context.WithTimeout(ctx, time.Hour)
	defer cancel()
	declared, err := bloby.Digest(bytes.NewReader(content))
	if err != nil {
		return err
	}
	var upload reservation
	if err := c.request(ctx, http.MethodPost, "/api/munki/icons", declaredUpload{Content: declared, Filename: filename}, &upload); err != nil {
		return err
	}
	if upload.ObjectID <= 0 || upload.Upload.Strategy != "direct-put" {
		return errors.New("icon upload requires an owned object and a direct PUT target")
	}
	body := func() io.ReadSeeker { return bytes.NewReader(content) }
	if err := c.transferBytes(ctx, upload.Upload.Target, body, declared.SizeBytes); err != nil {
		return err
	}
	endpoint := "/api/munki/software/" + strconv.FormatInt(softwareID, 10) + "/icon"
	return c.request(ctx, http.MethodPut, endpoint, map[string]int64{"object_id": upload.ObjectID}, nil)
}
