package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/woodleighschool/woodstar/internal/storage/capability"
)

// fileStore keeps blobs under a local directory. Keys map to paths beneath root.
type fileStore struct {
	root          string
	publicURL     string
	capabilityKey []byte
	ttl           time.Duration
}

func newFileStore(root string, publicURL string, capabilityKey []byte, ttl time.Duration) (*fileStore, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, errors.New("storage file root is empty")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve storage file root: %w", err)
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("create storage file root: %w", err)
	}
	return &fileStore{
		root:          abs,
		publicURL:     publicURL,
		capabilityKey: slices.Clone(capabilityKey),
		ttl:           ttl,
	}, nil
}

// resolve maps a storage key to a path under root, rejecting traversal.
func (s *fileStore) resolve(key string) (string, error) {
	if slices.Contains(strings.Split(key, "/"), "..") {
		return "", fmt.Errorf("invalid storage key %q", key)
	}
	path := filepath.Join(s.root, filepath.FromSlash(key))
	if path != s.root && !strings.HasPrefix(path, s.root+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid storage key %q", key)
	}
	return path, nil
}

func (s *fileStore) Open(_ context.Context, key string) (io.ReadCloser, ObjectInfo, error) {
	path, err := s.resolve(key)
	if err != nil {
		return nil, ObjectInfo{}, err
	}
	// #nosec G703 -- path comes from resolve, which keeps reads under root.
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ObjectInfo{}, ErrObjectNotFound
	}
	if err != nil {
		return nil, ObjectInfo{}, fmt.Errorf("open %q: %w", key, err)
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, ObjectInfo{}, fmt.Errorf("stat %q: %w", key, err)
	}
	return f, ObjectInfo{Size: info.Size()}, nil
}

func (s *fileStore) Put(_ context.Context, key string, r io.Reader, _ PutOptions) error {
	path, err := s.resolve(key)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	// #nosec G703 -- path comes from resolve, which rejects traversal and
	// constrains keys to the configured storage root.
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create dir for %q: %w", key, err)
	}
	tmp, err := os.CreateTemp(dir, ".upload-*")
	if err != nil {
		return fmt.Errorf("create temp for %q: %w", key, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := io.Copy(tmp, r); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write %q: %w", key, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close %q: %w", key, err)
	}
	// #nosec G703 -- tmpName is created under the already-resolved storage dir.
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return fmt.Errorf("chmod %q: %w", key, err)
	}
	// #nosec G703 -- path comes from resolve, which keeps writes under root.
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("commit %q: %w", key, err)
	}
	return nil
}

func (s *fileStore) Delete(_ context.Context, key string) error {
	path, err := s.resolve(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete %q: %w", key, err)
	}
	// Prune the now-empty <id> directory; ignore if it has siblings.
	_ = os.Remove(filepath.Dir(path))
	return nil
}

func (s *fileStore) Stat(_ context.Context, key string) (ObjectInfo, error) {
	path, err := s.resolve(key)
	if err != nil {
		return ObjectInfo{}, err
	}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return ObjectInfo{}, ErrObjectNotFound
	}
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("stat %q: %w", key, err)
	}
	return ObjectInfo{Size: info.Size()}, nil
}

func (s *fileStore) PresignGet(
	_ context.Context,
	key string,
	ttl time.Duration,
	opts GetOptions,
) (string, error) {
	return s.blobURL(capability.Claims{
		Op:          capability.OpGet,
		Key:         key,
		Exp:         time.Now().Add(s.expires(ttl)).Unix(),
		ContentType: opts.ContentType,
	})
}

func (s *fileStore) PresignPut(
	_ context.Context,
	key string,
	ttl time.Duration,
	opts PutOptions,
) (UploadTarget, error) {
	url, err := s.blobURL(capability.Claims{
		Op:          capability.OpPut,
		Key:         key,
		Exp:         time.Now().Add(s.expires(ttl)).Unix(),
		ContentType: opts.ContentType,
	})
	if err != nil {
		return UploadTarget{}, err
	}
	return UploadTarget{
		URL:       url,
		Method:    http.MethodPut,
		Transport: UploadTransportWoodstar,
	}, nil
}

func (s *fileStore) blobURL(claims capability.Claims) (string, error) {
	token, err := capability.Sign(s.capabilityKey, claims)
	if err != nil {
		return "", err
	}
	blobURL, err := url.Parse(s.publicURL + "/storage/blob")
	if err != nil {
		return "", err
	}
	values := blobURL.Query()
	values.Set("cap", token)
	blobURL.RawQuery = values.Encode()
	return blobURL.String(), nil
}

func (s *fileStore) expires(ttl time.Duration) time.Duration {
	if ttl <= 0 {
		return s.ttl
	}
	return ttl
}
