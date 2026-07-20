package worker

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const snapshotFilename = "state.json"

var (
	errSizeMismatch   = errors.New("size mismatch")
	errSHA256Mismatch = errors.New("sha256 mismatch")
)

// pointIdentity is the distribution point a worker learns from Woodstar.
type pointIdentity struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// packageState is a verified local mirror entry. The map of these is an
// optimization to skip re-hashing on boot; the filesystem and the desired set
// remain the source of truth each cycle.
type packageState struct {
	Filename   string    `json:"filename"`
	SHA256     string    `json:"sha256"`
	SizeBytes  int64     `json:"size_bytes"`
	VerifiedAt time.Time `json:"verified_at"`
}

type snapshot struct {
	DistributionPoint pointIdentity          `json:"distribution_point"`
	Packages          map[int64]packageState `json:"packages"`
}

// mirror is the worker's local package store: an in-memory map guarded for the
// concurrent serve node, snapshotted to a 0600 JSON file under the data dir.
type mirror struct {
	dataDir  string
	mu       sync.RWMutex
	identity pointIdentity
	packages map[int64]packageState
}

// loadMirror restores a worker's mirror from the data dir, starting empty when
// no snapshot exists yet.
func loadMirror(dataDir string) (*mirror, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	m := &mirror{dataDir: dataDir, packages: map[int64]packageState{}}

	raw, err := os.ReadFile(filepath.Join(dataDir, snapshotFilename)) //nolint:gosec // dataDir is an administrator-configured storage root.
	if errors.Is(err, os.ErrNotExist) {
		return m, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read snapshot: %w", err)
	}
	var snap snapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		return nil, fmt.Errorf("decode snapshot: %w", err)
	}
	m.identity = snap.DistributionPoint
	if snap.Packages != nil {
		m.packages = snap.Packages
	}
	return m, nil
}

// localPath is the flat on-disk path for a package: <data_dir>/<id>-<filename>.
func (m *mirror) localPath(packageID int64, filename string) string {
	return filepath.Join(m.dataDir, fmt.Sprintf("%d-%s", packageID, filepath.Base(filename)))
}

func (m *mirror) get(packageID int64) (packageState, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	state, ok := m.packages[packageID]
	return state, ok
}

func (m *mirror) setIdentity(identity pointIdentity) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.identity = identity
}

func (m *mirror) put(packageID int64, state packageState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.packages[packageID] = state
}

func (m *mirror) remove(packageID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.packages, packageID)
}

func (m *mirror) packageIDs() []int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]int64, 0, len(m.packages))
	for id := range m.packages {
		ids = append(ids, id)
	}
	return ids
}

// satisfies reports whether the mirror already holds the wanted bytes for a
// package: a matching recorded hash and size, with a present file of that size.
func (m *mirror) satisfies(packageID int64, sha256 string, sizeBytes int64) bool {
	state, ok := m.get(packageID)
	if !ok || state.SHA256 != sha256 || state.SizeBytes != sizeBytes {
		return false
	}
	info, err := os.Stat(m.localPath(packageID, state.Filename))
	return err == nil && info.Size() == sizeBytes
}

// save writes the in-memory mirror to a 0600 snapshot via a temp-file rename so
// a crash never leaves a half-written snapshot.
func (m *mirror) save() error {
	m.mu.RLock()
	snap := snapshot{DistributionPoint: m.identity, Packages: maps.Clone(m.packages)}
	m.mu.RUnlock()

	raw, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(m.dataDir, snapshotFilename+".tmp")
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(m.dataDir, snapshotFilename))
}

// verifyFile reports whether path holds exactly wantSize bytes hashing to
// wantSHA256. It reads the whole file and is used once per download, never on
// the serve path.
func verifyFile(path string, wantSize int64, wantSHA256 string) error {
	f, err := os.Open(path) //nolint:gosec // Callers provide a mirror path under the configured data directory.
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		return err
	}
	if info.Size() != wantSize {
		return errSizeMismatch
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return err
	}
	if hex.EncodeToString(hash.Sum(nil)) != wantSHA256 {
		return errSHA256Mismatch
	}
	return nil
}
