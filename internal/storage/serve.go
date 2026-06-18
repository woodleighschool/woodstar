package storage

import (
	"errors"
	"io"
	"net/http"
	"path"
	"time"
)

// ServeOptions carries HTTP metadata for a caller-authorized object read.
type ServeOptions struct {
	ContentType  string
	Filename     string
	CacheControl string
	ETag         string
}

// ServeObject streams an already-authorized object key from store.
func ServeObject(w http.ResponseWriter, r *http.Request, store Store, key string, opts ServeOptions) {
	reader, info, err := store.Open(r.Context(), key)
	if errors.Is(err, ErrObjectNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	seeker, ok := reader.(io.ReadSeeker)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	contentType := opts.ContentType
	if contentType == "" {
		contentType = info.ContentType
	}
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	if opts.CacheControl != "" {
		w.Header().Set("Cache-Control", opts.CacheControl)
	}
	if opts.ETag != "" {
		w.Header().Set("ETag", opts.ETag)
	}
	filename := opts.Filename
	if filename == "" {
		filename = path.Base(key)
	}
	http.ServeContent(w, r, filename, time.Time{}, seeker)
}
