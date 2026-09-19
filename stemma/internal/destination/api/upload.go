package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/woodleighschool/stemma/plugin"
)

// reservation is an object the repository allocated for content and how to
// send that content.
type reservation struct {
	ObjectID int64 `json:"object_id"`
	Upload   struct {
		Strategy string       `json:"strategy"`
		Target   uploadTarget `json:"target"`
	} `json:"upload"`
}

// storedObject is the content the repository verified for an object.
type storedObject struct {
	ID        int64  `json:"id"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

type uploadTarget struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
}

// Upload sends a leased installer to a new object and returns its id once the
// repository holds the prepared content. A failed upload releases its object.
func (c *Client) Upload(ctx context.Context, artifact plugin.Artifact) (_ int64, runErr error) {
	ctx, cancel := context.WithTimeout(ctx, time.Hour)
	defer cancel()
	done := plugin.Stage(ctx, "Uploading installer")
	defer func() { done(runErr) }()
	if !filepath.IsAbs(artifact.Path) {
		return 0, errors.New("artifact path must be an absolute leased path")
	}
	file, err := os.Open(artifact.Path)
	if err != nil {
		return 0, errors.New("open leased installer artifact")
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() != artifact.Size {
		return 0, errors.New("leased artifact size or file type changed")
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, contextReader{Reader: file, ctx: ctx}); err != nil {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		return 0, errors.New("hash leased installer artifact")
	}
	if hex.EncodeToString(hash.Sum(nil)) != artifact.SHA256 {
		return 0, errors.New("leased installer artifact digest changed")
	}
	var upload reservation
	if err := c.request(ctx, http.MethodPost, "/api/munki/package-installers", map[string]any{"filename": artifact.Filename, "size_bytes": artifact.Size}, &upload); err != nil {
		return 0, err
	}
	if upload.ObjectID <= 0 {
		return 0, errors.New("upload reservation has no object id")
	}
	endpoint := "/api/munki/package-installers/" + strconv.FormatInt(upload.ObjectID, 10)
	// A later run uploads afresh, so nothing about a failed attempt is worth keeping.
	defer func() {
		if runErr != nil {
			c.ReleaseUpload(ctx, upload.ObjectID)
		}
	}()
	switch upload.Upload.Strategy {
	case "direct-put":
		body := &installerBody{SectionReader: io.NewSectionReader(file, 0, artifact.Size), ctx: ctx, size: artifact.Size}
		if _, err := c.transferBytes(ctx, upload.Upload.Target, body, artifact.Size); err != nil {
			return 0, err
		}
	case "multipart":
		if err := c.uploadMultipart(ctx, endpoint, file, artifact.Size); err != nil {
			return 0, err
		}
	default:
		return 0, errors.New("unsupported installer upload strategy")
	}
	if err := c.finalizeUpload(ctx, artifact, upload.ObjectID); err != nil {
		return 0, err
	}
	return upload.ObjectID, nil
}

// ReleaseUpload deletes an uploaded installer that no package references. It is
// best effort and outlives the caller's cancellation.
func (c *Client) ReleaseUpload(ctx context.Context, objectID int64) {
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	_ = c.request(cleanup, http.MethodDelete, "/api/munki/package-installers/"+strconv.FormatInt(objectID, 10), nil, nil)
}

func (c *Client) finalizeUpload(ctx context.Context, artifact plugin.Artifact, objectID int64) (runErr error) {
	done := plugin.Stage(ctx, "Finalizing upload")
	defer func() { done(runErr) }()
	var finalized storedObject
	endpoint := "/api/munki/package-installers/" + strconv.FormatInt(objectID, 10)
	if err := c.request(ctx, http.MethodPut, endpoint, nil, &finalized); err != nil {
		return err
	}
	if finalized.ID != objectID || finalized.SHA256 != artifact.SHA256 || finalized.SizeBytes != artifact.Size {
		return errors.New("finalized installer does not match the prepared content")
	}
	return nil
}

func (c *Client) uploadMultipart(ctx context.Context, endpoint string, file *os.File, size int64) error {
	// S3 permits at most 10,000 parts, so large installers need larger chunks.
	partSize := max(int64(64<<20), (size+9999)/10000)
	var parts []map[string]any
	for offset, part := int64(0), 1; offset < size; offset, part = offset+partSize, part+1 {
		var target uploadTarget
		if err := c.request(ctx, http.MethodPost, endpoint+"/multipart/parts/"+strconv.Itoa(part), nil, &target); err != nil {
			return err
		}
		length := min(partSize, size-offset)
		body := &installerBody{SectionReader: io.NewSectionReader(file, offset, length), ctx: ctx, offset: offset, size: size}
		headers, err := c.transferBytes(ctx, target, body, length)
		if err != nil {
			return err
		}
		etag := headers.Get("ETag")
		if etag == "" {
			return errors.New("multipart transfer returned no ETag")
		}
		parts = append(parts, map[string]any{"part_number": part, "etag": etag})
	}
	return c.request(ctx, http.MethodPut, endpoint+"/multipart", map[string]any{"parts": parts}, nil)
}

// installerBody reads one request's range of an installer and reports progress
// across the whole upload. Retries rewind the body, so progress follows the read
// position instead of summing reads.
type installerBody struct {
	*io.SectionReader

	ctx          context.Context
	offset, size int64
	reported     time.Time
}

func (body *installerBody) Read(data []byte) (int, error) {
	if err := body.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := body.SectionReader.Read(data)
	if err != nil || time.Since(body.reported) >= 250*time.Millisecond {
		position, _ := body.Seek(0, io.SeekCurrent)
		current := body.offset + position
		plugin.Logger(body.ctx).InfoContext(body.ctx, "Transfer progress", "progress", true, "current", current, "total", body.size, "unit", "bytes", "progress_final", err != nil && current == body.size)
		body.reported = time.Now()
	}
	return n, err
}

type contextReader struct {
	io.Reader

	ctx context.Context
}

func (r contextReader) Read(data []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.Reader.Read(data)
}

func (c *Client) transferBytes(ctx context.Context, target uploadTarget, body io.ReadSeeker, size int64) (http.Header, error) {
	parsed, err := url.Parse(target.URL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || strings.ToUpper(target.Method) != http.MethodPut {
		return nil, errors.New("upload target must be an HTTPS PUT")
	}
	ctx, cancel := context.WithTimeout(ctx, time.Hour)
	defer cancel()
	response, err := c.transfer.R().SetContext(ctx).
		SetHeaders(target.Headers).SetContentLength(size).SetBody(body).
		Put(target.URL)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errors.New("installer transfer failed")
	}
	if !response.IsStatusSuccess() {
		return nil, fmt.Errorf("installer transfer returned HTTP %d", response.StatusCode())
	}
	return response.Header(), nil
}
