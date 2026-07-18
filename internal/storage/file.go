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
	baseURL       string
	capabilityKey []byte
	ttl           time.Duration
}

func (*fileStore) uploadMode() uploadMode {
	return uploadModeDirect
}

func newFileStore(root, baseURL string, capabilityKey []byte, ttl time.Duration) (*fileStore, error) {
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
		baseURL:       baseURL,
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

func (s *fileStore) Open(_ context.Context, key string) (ObjectReader, ObjectInfo, error) {
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
	return fileObjectReader{ReadSeeker: f, Closer: f}, ObjectInfo{Size: info.Size()}, nil
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

func (s *fileStore) Move(
	_ context.Context,
	sourceKey string,
	destinationKey string,
	_ PutOptions,
) error {
	sourcePath, err := s.resolve(sourceKey)
	if err != nil {
		return err
	}
	destinationPath, err := s.resolve(destinationKey)
	if err != nil {
		return err
	}
	// #nosec G703 -- both paths come from resolve, which keeps them under root.
	if err := os.MkdirAll(filepath.Dir(destinationPath), 0o750); err != nil {
		return fmt.Errorf("create dir for %q: %w", destinationKey, err)
	}
	// #nosec G703 -- both paths come from resolve, which keeps them under root.
	if err := os.Rename(sourcePath, destinationPath); errors.Is(err, os.ErrNotExist) {
		return ErrObjectNotFound
	} else if err != nil {
		return fmt.Errorf("move %q to %q: %w", sourceKey, destinationKey, err)
	}
	_ = os.Remove(filepath.Dir(sourcePath))
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
	// Prune the now-empty parent directory; ignore if it has siblings.
	_ = os.Remove(filepath.Dir(path))
	return nil
}

func (*fileStore) deliveryMode() deliveryMode {
	return deliveryStream
}

func (s *fileStore) PresignGet(
	_ context.Context,
	key string,
	ttl time.Duration,
	opts GetOptions,
) (string, error) {
	return s.blobURL(BlobCapabilityClaims{
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
) (UploadTarget, error) {
	url, err := s.blobURL(BlobCapabilityClaims{
		Op:  capability.OpPut,
		Key: key,
		Exp: time.Now().Add(s.expires(ttl)).Unix(),
	})
	if err != nil {
		return UploadTarget{}, err
	}
	return UploadTarget{
		URL:    url,
		Method: http.MethodPut,
	}, nil
}

func (s *fileStore) blobURL(claims BlobCapabilityClaims) (string, error) {
	token, err := capability.Sign(s.capabilityKey, claims)
	if err != nil {
		return "", err
	}
	blobURL, err := url.Parse(strings.TrimRight(s.baseURL, "/") + "/storage/" + escapeKeyPath(claims.Key))
	if err != nil {
		return "", err
	}
	values := blobURL.Query()
	values.Set("cap", token)
	blobURL.RawQuery = values.Encode()
	return blobURL.String(), nil
}

func escapeKeyPath(key string) string {
	parts := strings.Split(key, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func (s *fileStore) expires(ttl time.Duration) time.Duration {
	return ttlOrDefault(ttl, s.ttl)
}

// fileObjectReader exposes only the object-reader contract.
// Returning the concrete file would expose optional interfaces that allow net/http
// to select platform sendfile paths, which can break HTTP framing on some stacks.
type fileObjectReader struct {
	io.ReadSeeker
	io.Closer
}
