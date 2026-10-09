package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/woodleighschool/goodies/bloby"
	"github.com/woodleighschool/stemma/plugin"
	"golang.org/x/sync/errgroup"
	"resty.dev/v3"
)

// declaredUpload names an upload and declares the bytes it will carry. Storage
// accepts no other content for the object it reserves.
type declaredUpload struct {
	bloby.Content

	Filename string `json:"filename"`
}

// reservation is an object the repository allocated for content and how to
// send that content.
type reservation struct {
	ObjectID int64 `json:"object_id"`
	Upload   struct {
		Strategy string       `json:"strategy"`
		Target   uploadTarget `json:"target"`
	} `json:"upload"`
}

type uploadTarget struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
}

// concurrentParts is how many multipart parts are in flight at once.
// Throughput to S3 storage levels off around four.
const concurrentParts = 4

// Upload sends a leased installer to a new object and returns its id once the
// repository has published it. One read of the installer checks it against the
// prepared artifact and yields the declaration that storage holds the transfer
// to. A failed upload releases its object.
func (c *Client) Upload(ctx context.Context, artifact plugin.Artifact) (_ int64, runErr error) {
	ctx, cancel := context.WithTimeout(ctx, time.Hour)
	defer cancel()
	done := plugin.Stage(ctx, "Uploading installer", plugin.Detail(artifact.Filename))
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
	content, err := bloby.Digest(contextReader{Reader: file, ctx: ctx})
	if err != nil {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		return 0, errors.New("read leased installer artifact")
	}
	if content.SHA256 != artifact.SHA256 || content.SizeBytes != artifact.Size {
		return 0, errors.New("leased installer artifact digest changed")
	}
	var upload reservation
	if err := c.request(ctx, http.MethodPost, "/api/munki/package-installers", declaredUpload{Content: content, Filename: artifact.Filename}, &upload); err != nil {
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
	progress := &transferProgress{ctx: ctx, total: artifact.Size}
	switch upload.Upload.Strategy {
	case "direct-put":
		err = c.transferBytes(ctx, upload.Upload.Target, progress.body(file, 0, artifact.Size), artifact.Size)
	case "multipart":
		err = c.uploadMultipart(ctx, endpoint, file, artifact.Size, progress)
	default:
		err = errors.New("unsupported installer upload strategy")
	}
	if err != nil {
		return 0, err
	}
	progress.finish()
	if err := c.finalizeUpload(ctx, endpoint, artifact.Filename); err != nil {
		return 0, err
	}
	return upload.ObjectID, nil
}

// ReleaseUpload deletes an uploaded installer that no package references. It is
// best effort and outlives the caller's cancellation. The repository may have
// released the object already.
func (c *Client) ReleaseUpload(ctx context.Context, objectID int64) {
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	_ = c.request(cleanup, http.MethodDelete, "/api/munki/package-installers/"+strconv.FormatInt(objectID, 10), nil, nil)
}

// finalizeUpload publishes the object. The repository refuses an upload whose
// stored bytes are missing or differ from the declaration.
func (c *Client) finalizeUpload(ctx context.Context, endpoint, filename string) (runErr error) {
	done := plugin.Stage(ctx, "Finalizing upload", plugin.Detail(filename))
	defer func() { done(runErr) }()
	return c.request(ctx, http.MethodPut, endpoint, nil, nil)
}

// uploadMultipart sends the installer as parts, several at a time. The first
// part to fail stops those in flight and keeps the rest from starting.
func (c *Client) uploadMultipart(ctx context.Context, endpoint string, file *os.File, size int64, progress *transferProgress) error {
	// S3 permits at most 10,000 parts, so large installers need larger parts.
	partSize := max(int64(64<<20), (size+9999)/10000)
	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(concurrentParts)
	for offset, number := int64(0), 1; offset < size; offset, number = offset+partSize, number+1 {
		length := min(partSize, size-offset)
		group.Go(func() error {
			return c.uploadPart(ctx, endpoint+"/multipart/parts/"+strconv.Itoa(number), file, offset, length, progress)
		})
	}
	return group.Wait()
}

// uploadPart signs one part with the checksum of its bytes and sends them.
// Storage refuses a part whose bytes differ from that checksum.
func (c *Client) uploadPart(ctx context.Context, endpoint string, file *os.File, offset, length int64, progress *transferProgress) error {
	checksum, err := bloby.CRC64NVME(contextReader{Reader: io.NewSectionReader(file, offset, length), ctx: ctx})
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("read leased installer artifact")
	}
	// Signing reserves nothing, so it repeats like an idempotent request.
	sign := c.api.R().SetRetryAllowNonIdempotent(true)
	var target uploadTarget
	if err := send(ctx, sign, http.MethodPost, endpoint, map[string]string{"crc64nvme": checksum}, &target); err != nil {
		return err
	}
	return c.transferBytes(ctx, target, progress.body(file, offset, length), length)
}

// transferProgress reports how much of an installer its request bodies have
// handed to storage. Parts send concurrently and a request that is sent again
// starts over, so only the latest body of each range counts and their sum
// never passes the installer's size.
type transferProgress struct {
	ctx   context.Context
	total int64

	mu       sync.Mutex
	sent     int64
	reported time.Time
}

// body returns the source of request bodies for one range of the installer.
// Each body it opens replaces the one before.
func (p *transferProgress) body(file *os.File, offset, length int64) func() io.ReadSeeker {
	var current *installerBody
	return func() io.ReadSeeker {
		p.mu.Lock()
		defer p.mu.Unlock()
		if current != nil {
			current.replaced = true
			p.sent -= current.sent
		}
		current = &installerBody{SectionReader: io.NewSectionReader(file, offset, length), progress: p}
		return current
	}
}

// finish reports the end of a transfer that storage accepted in full.
func (p *transferProgress) finish() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.report(true)
}

func (p *transferProgress) report(final bool) {
	plugin.Logger(p.ctx).InfoContext(p.ctx, "Transfer progress", "progress", true, "current", p.sent, "total", p.total, "unit", "bytes", "progress_final", final)
	p.reported = time.Now()
}

// installerBody is one attempt at sending a range of an installer.
type installerBody struct {
	*io.SectionReader

	progress *transferProgress
	// The progress lock guards how much this body has counted and whether a
	// later attempt replaced it.
	sent     int64
	replaced bool
}

func (body *installerBody) Read(data []byte) (int, error) {
	n, err := body.SectionReader.Read(data)
	p := body.progress
	p.mu.Lock()
	defer p.mu.Unlock()
	if !body.replaced {
		body.sent += int64(n)
		p.sent += int64(n)
		if time.Since(p.reported) >= 250*time.Millisecond {
			p.report(false)
		}
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

// transferBytes sends content to a signed target with the headers it was signed
// with. Errors never carry the target, whose URL is a credential.
func (c *Client) transferBytes(ctx context.Context, target uploadTarget, body func() io.ReadSeeker, size int64) error {
	parsed, err := url.Parse(target.URL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || strings.ToUpper(target.Method) != http.MethodPut {
		return errors.New("upload target must be an HTTPS PUT")
	}
	request := c.transfer.R().SetContext(ctx).SetHeaders(target.Headers).SetContentLength(size).SetBody(body())
	// The transport can still be reading the body of an attempt that failed
	// when the next one starts, so each attempt reads a body of its own.
	request.AddRetryHooks(func(*resty.Response, error) { request.SetBody(body()) })
	response, err := request.Put(target.URL)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("installer transfer failed")
	}
	if !response.IsStatusSuccess() {
		return fmt.Errorf("installer transfer returned HTTP %d", response.StatusCode())
	}
	return nil
}
