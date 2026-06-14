package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// fileStore keeps blobs under a local directory. Keys map to paths beneath root;
// woodstar proxies transfers, so it implements Store but not Presigner.
type fileStore struct {
	root string
}

func newFileStore(root string) (*fileStore, error) {
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
	return &fileStore{root: abs}, nil
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
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return fmt.Errorf("chmod %q: %w", key, err)
	}
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
