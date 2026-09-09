package destination

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

func (remote *client) upload(ctx context.Context, artifact plugin.Artifact) (int64, error) {
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
	if _, err := io.Copy(hash, file); err != nil {
		return 0, errors.New("hash leased installer artifact")
	}
	if hex.EncodeToString(hash.Sum(nil)) != artifact.SHA256 {
		return 0, errors.New("leased installer artifact digest changed")
	}
	var upload struct {
		ObjectID int64 `json:"object_id"`
		Upload   struct {
			Strategy string       `json:"strategy"`
			Target   uploadTarget `json:"target"`
		} `json:"upload"`
	}
	if err := remote.request(ctx, http.MethodPost, "/api/munki/package-installers", map[string]any{"filename": artifact.Filename, "size_bytes": artifact.Size}, &upload); err != nil {
		return 0, err
	}
	if upload.ObjectID <= 0 {
		return 0, errors.New("upload reservation has no object id")
	}
	endpoint := "/api/munki/package-installers/" + strconv.FormatInt(upload.ObjectID, 10)
	finalizedSuccessfully := false
	defer func() {
		if !finalizedSuccessfully {
			remote.cleanupUpload(ctx, upload.ObjectID)
		}
	}()
	switch upload.Upload.Strategy {
	case "direct-put":
		if _, err := remote.transferBytes(ctx, upload.Upload.Target, io.NewSectionReader(file, 0, artifact.Size), artifact.Size); err != nil {
			return 0, err
		}
	case "multipart":
		if err := remote.uploadMultipart(ctx, endpoint, file, artifact.Size); err != nil {
			return 0, err
		}
	default:
		return 0, errors.New("unsupported installer upload strategy")
	}
	var finalized struct {
		ID        int64  `json:"id"`
		SHA256    string `json:"sha256"`
		SizeBytes int64  `json:"size_bytes"`
	}
	if err := remote.request(ctx, http.MethodPut, endpoint, nil, &finalized); err != nil {
		return 0, err
	}
	if finalized.ID != upload.ObjectID || finalized.SHA256 != artifact.SHA256 || finalized.SizeBytes != artifact.Size {
		return 0, errors.New("finalized installer does not match the prepared content")
	}
	finalizedSuccessfully = true
	return finalized.ID, nil
}

func (remote *client) uploadMultipart(ctx context.Context, endpoint string, file *os.File, size int64) error {
	// S3 permits at most 10,000 parts, so large installers need larger chunks.
	partSize := max(int64(64<<20), (size+9999)/10000)
	var parts []map[string]any
	for offset, part := int64(0), 1; offset < size; offset, part = offset+partSize, part+1 {
		var target uploadTarget
		if err := remote.request(ctx, http.MethodPost, endpoint+"/multipart/parts/"+strconv.Itoa(part), nil, &target); err != nil {
			return err
		}
		length := min(partSize, size-offset)
		headers, err := remote.transferBytes(ctx, target, io.NewSectionReader(file, offset, length), length)
		if err != nil {
			return err
		}
		etag := headers.Get("ETag")
		if etag == "" {
			return errors.New("multipart transfer returned no ETag")
		}
		parts = append(parts, map[string]any{"part_number": part, "etag": etag})
	}
	return remote.request(ctx, http.MethodPut, endpoint+"/multipart", map[string]any{"parts": parts}, nil)
}

type uploadTarget struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
}

func (remote *client) transferBytes(ctx context.Context, target uploadTarget, body io.ReadSeeker, size int64) (http.Header, error) {
	parsed, err := url.Parse(target.URL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || strings.ToUpper(target.Method) != http.MethodPut {
		return nil, errors.New("upload target must be an HTTPS PUT")
	}
	ctx, cancel := context.WithTimeout(ctx, time.Hour)
	defer cancel()
	response, err := remote.transfer.R().SetContext(ctx).
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

func (remote *client) cleanupUpload(ctx context.Context, objectID int64) {
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	_ = remote.request(cleanup, http.MethodDelete, "/api/munki/package-installers/"+strconv.FormatInt(objectID, 10), nil, nil)
}
