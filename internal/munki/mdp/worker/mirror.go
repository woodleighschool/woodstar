package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/woodleighschool/woodstar/internal/munki/mdp/wire"
)

var (
	errSizeMismatch   = errors.New("size mismatch")
	errSHA256Mismatch = errors.New("sha256 mismatch")
)

type packageState struct {
	SHA256    string
	SizeBytes int64
	info      os.FileInfo
}

// Verified entries only accelerate checks within this process. The desired set
// and the directory determine what is retained, including after a crash.
type mirror struct {
	dataDir  string
	mu       sync.RWMutex
	packages map[int64]packageState

	allocate func(*os.File, int64) error
}

func openMirror(dataDir string) (*mirror, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create cache directory: %w", err)
	}
	return &mirror{
		dataDir:  dataDir,
		packages: make(map[int64]packageState),
		allocate: allocateFile,
	}, nil
}

func (m *mirror) localPath(packageID int64, sha string) string {
	return filepath.Join(m.dataDir, strconv.FormatInt(packageID, 10)+"-"+sha)
}

func (m *mirror) get(packageID int64) (packageState, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	state, ok := m.packages[packageID]
	return state, ok
}

func (m *mirror) put(packageID int64, state packageState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.packages[packageID] = state
}

// reconcile runs only in the session's coordinator. Obsolete jobs have stopped
// before it is called; unchanged jobs may still be writing their partial files.
func (m *mirror) reconcile(ctx context.Context, desired []wire.DesiredPackage, active map[int64]bool) error {
	keep := make(map[string]bool, len(desired)*2)
	m.mu.Lock()
	for _, pkg := range desired {
		name := filepath.Base(m.localPath(pkg.PackageID, pkg.SHA256))
		keep[name] = true
		if active[pkg.PackageID] {
			keep[name+".download"] = true
		}
	}
	for id, state := range m.packages {
		if !keep[filepath.Base(m.localPath(id, state.SHA256))] {
			delete(m.packages, id)
		}
	}
	m.mu.Unlock()
	var failures []error
	if err := m.sweep(ctx, keep); err != nil {
		failures = append(failures, err)
	}
	for _, pkg := range desired {
		if active[pkg.PackageID] {
			continue
		}
		if err := m.check(ctx, pkg); err != nil {
			failures = append(failures, fmt.Errorf("check package %d: %w", pkg.PackageID, err))
		}
	}
	return errors.Join(failures...)
}

func (m *mirror) sweep(ctx context.Context, keep map[string]bool) error {
	entries, err := os.ReadDir(m.dataDir)
	if err != nil {
		return fmt.Errorf("read cache directory: %w", err)
	}
	var failures []error
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		// An ext filesystem owns this directory when the cache is a mount root.
		if entry.Name() == "lost+found" && entry.IsDir() {
			continue
		}
		if keep[entry.Name()] && entry.Type().IsRegular() {
			continue
		}
		if err := removeFile(filepath.Join(m.dataDir, entry.Name())); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func (m *mirror) check(ctx context.Context, pkg wire.DesiredPackage) error {
	path := m.localPath(pkg.PackageID, pkg.SHA256)
	info, err := os.Lstat(path)
	state, ok := m.get(pkg.PackageID)
	if err == nil && info.Mode().IsRegular() && ok && state.SHA256 == pkg.SHA256 && state.SizeBytes == pkg.SizeBytes &&
		state.info != nil && os.SameFile(state.info, info) &&
		info.Size() == pkg.SizeBytes && info.ModTime().Equal(state.info.ModTime()) {
		return nil
	}
	m.mu.Lock()
	delete(m.packages, pkg.PackageID)
	m.mu.Unlock()
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("cache path is not a regular file: %s", path)
	}
	if err := verifyFile(ctx, path, pkg.SizeBytes, pkg.SHA256); err != nil {
		if errors.Is(err, errSizeMismatch) || errors.Is(err, errSHA256Mismatch) {
			return removeFile(path)
		}
		return err
	}
	m.put(pkg.PackageID, packageState{SHA256: pkg.SHA256, SizeBytes: pkg.SizeBytes, info: info})
	return nil
}

// fetch owns the file from capacity admission through verification and publish.
// The callback only transfers bytes; it cannot expose an unverified installer.
func (m *mirror) fetch(ctx context.Context, pkg wire.DesiredPackage, transfer func(io.Writer) (int64, error)) error {
	file, err := m.begin(pkg)
	if err != nil {
		return err
	}
	hash := sha256.New()
	n, transferErr := transfer(io.MultiWriter(file, hash))
	err = errors.Join(transferErr, file.Close(), ctx.Err())
	if err == nil {
		switch {
		case n != pkg.SizeBytes:
			err = errSizeMismatch
		case hex.EncodeToString(hash.Sum(nil)) != pkg.SHA256:
			err = errSHA256Mismatch
		default:
			err = m.publish(pkg, file.Name())
		}
	}
	if err != nil {
		cleanupErr := removeFile(file.Name())
		return errors.Join(err, cleanupErr)
	}
	return nil
}

func (m *mirror) begin(pkg wire.DesiredPackage) (*os.File, error) {
	path := m.localPath(pkg.PackageID, pkg.SHA256) + ".download"
	if err := removeFile(path); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) //nolint:gosec // Path is derived from a validated package ID and digest in the owned cache directory.
	if err != nil {
		return nil, err
	}
	// Reserve real blocks before any network request. The filesystem arbitrates
	// concurrent downloads and quotas, including after a reconnect or restart.
	if err := m.allocate(file, pkg.SizeBytes); err != nil {
		return nil, errors.Join(fmt.Errorf("reserve %d cache bytes: %w", pkg.SizeBytes, err), file.Close(), removeFile(path))
	}
	return file, nil
}

func (m *mirror) publish(pkg wire.DesiredPackage, temporary string) error {
	path := m.localPath(pkg.PackageID, pkg.SHA256)
	if err := os.Rename(temporary, path); err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	m.put(pkg.PackageID, packageState{SHA256: pkg.SHA256, SizeBytes: pkg.SizeBytes, info: info})
	return nil
}

func removeFile(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove cache file: %w", err)
	}
	return nil
}

func verifyFile(ctx context.Context, path string, wantSize int64, wantSHA256 string) error {
	file, err := os.Open(path) //nolint:gosec // The caller selects a regular file in the owned cache directory.
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Size() != wantSize {
		return errSizeMismatch
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, contextReader{ctx: ctx, reader: file}); err != nil {
		return err
	}
	if hex.EncodeToString(hash.Sum(nil)) != wantSHA256 {
		return errSHA256Mismatch
	}
	return nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
