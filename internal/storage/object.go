package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// Object is a row in the storage registry: one stored (or pending) blob. The
// byte key is derived, never stored, so the path format lives in one place.
type Object struct {
	ID          int64
	Prefix      string
	Filename    string
	ContentType string
	SizeBytes   *int64
	SHA256      *string
	AvailableAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Key builds a storage key from its parts: <prefix>/<id>/<filename>. This is the
// one place the key format lives.
func Key(prefix string, id int64, filename string) string {
	return fmt.Sprintf("%s/%d/%s", prefix, id, filename)
}

// Key is the object's storage key.
func (o Object) Key() string {
	return Key(o.Prefix, o.ID, o.Filename)
}

// Available reports whether the bytes have been confirmed present.
func (o Object) Available() bool {
	return o.AvailableAt != nil
}

// ObjectStore is the database registry of stored objects.
type ObjectStore struct {
	db      *database.DB
	q       *sqlc.Queries
	backend Store
}

// NewObjectStore returns a registry backed by db.
func NewObjectStore(db *database.DB, backend Store) *ObjectStore {
	return &ObjectStore{db: db, q: db.Queries(), backend: backend}
}

// CreatePending inserts a pending object and returns it with its assigned id.
// The caller uploads to Object.Key() and then calls Confirm.
func (s *ObjectStore) CreatePending(ctx context.Context, prefix, filename, contentType string) (*Object, error) {
	if !prefixPattern.MatchString(prefix) {
		return nil, fmt.Errorf("%w: invalid storage prefix %q", dbutil.ErrInvalidInput, prefix)
	}
	filename, err := cleanUploadFilename(filename)
	if err != nil {
		return nil, err
	}
	row, err := s.q.CreateStorageObject(ctx, sqlc.CreateStorageObjectParams{
		Prefix:      prefix,
		Filename:    filename,
		ContentType: contentType,
	})
	if err != nil {
		return nil, dbutil.MutationError(err)
	}
	obj := objectFromSQLC(row)
	return &obj, nil
}

// Confirm records the landed object's size, sha, and content type, and marks it
// available. A content type of "" keeps whatever was set at creation.
func (s *ObjectStore) Confirm(ctx context.Context, id, size int64, contentType, sha256 string) (*Object, error) {
	row, err := s.q.ConfirmStorageObject(ctx, sqlc.ConfirmStorageObjectParams{
		ID:          id,
		SizeBytes:   &size,
		Sha256:      &sha256,
		ContentType: contentType,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, dbutil.MutationError(err)
	}
	obj := objectFromSQLC(row)
	return &obj, nil
}

// ConfirmUploaded verifies the bytes in the configured backend and marks the
// object available with server-derived size and SHA-256 metadata.
func (s *ObjectStore) ConfirmUploaded(ctx context.Context, id int64) (*Object, error) {
	if s.backend == nil {
		return nil, errors.New("storage backend is not configured")
	}
	obj, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// Re-read the whole object to compute its SHA-256; pkginfo needs a real one.
	reader, info, err := s.backend.Open(ctx, obj.Key())
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	hash := sha256.New()
	size, err := io.Copy(hash, reader)
	if err != nil {
		return nil, fmt.Errorf("hash %q: %w", obj.Key(), err)
	}
	contentType := info.ContentType
	if contentType == "" {
		contentType = obj.ContentType
	}
	return s.Confirm(ctx, obj.ID, size, contentType, hex.EncodeToString(hash.Sum(nil)))
}

// GetByID returns one object.
func (s *ObjectStore) GetByID(ctx context.Context, id int64) (*Object, error) {
	row, err := s.q.GetStorageObjectByID(ctx, sqlc.GetStorageObjectByIDParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	obj := objectFromSQLC(row)
	return &obj, nil
}

// ListByIDs returns objects keyed by id. Missing IDs are ignored.
func (s *ObjectStore) ListByIDs(ctx context.Context, ids []int64) (map[int64]Object, error) {
	rows, err := s.q.ListStorageObjectsByIDs(ctx, sqlc.ListStorageObjectsByIDsParams{Ids: ids})
	if err != nil {
		return nil, err
	}
	objects := make(map[int64]Object, len(rows))
	for _, row := range rows {
		obj := objectFromSQLC(row)
		objects[obj.ID] = obj
	}
	return objects, nil
}

// ListByPrefix returns available objects under a prefix, newest first.
func (s *ObjectStore) ListByPrefix(
	ctx context.Context,
	prefix string,
	params dbutil.ListParams,
) ([]Object, int, error) {
	params = dbutil.CleanListParams(params)
	count, err := s.q.CountStorageObjectsByPrefix(ctx, sqlc.CountStorageObjectsByPrefixParams{Prefix: prefix})
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.q.ListStorageObjectsByPrefix(ctx, sqlc.ListStorageObjectsByPrefixParams{
		Prefix:     prefix,
		OffsetRows: params.PageIndex * params.PageSize,
		LimitRows:  params.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}
	objects := make([]Object, len(rows))
	for i, row := range rows {
		objects[i] = objectFromSQLC(row)
	}
	return objects, int(count), nil
}

// DeleteByID removes an object row. It fails with a conflict if a consumer FK
// still references it.
func (s *ObjectStore) DeleteByID(ctx context.Context, id int64) error {
	rows, err := s.q.DeleteStorageObject(ctx, sqlc.DeleteStorageObjectParams{ID: id})
	if err != nil {
		return dbutil.DeleteConflict(err, "storage object is still referenced")
	}
	if rows == 0 {
		return dbutil.ErrNotFound
	}
	return nil
}

// DeleteUnreferenced deletes backend bytes and rows for objects that no Munki
// resource references anymore. Failed backend deletes leave rows in place so the
// database does not claim cleanup happened when bytes still exist.
func (s *ObjectStore) DeleteUnreferenced(ctx context.Context, ids ...int64) error {
	rows, err := s.q.ListUnreferencedStorageObjects(
		ctx,
		sqlc.ListUnreferencedStorageObjectsParams{Ids: ids},
	)
	if err != nil {
		return err
	}
	for _, row := range rows {
		key := Key(row.Prefix, row.ID, row.Filename)
		if s.backend != nil {
			if err := s.backend.Delete(ctx, key); err != nil {
				return err
			}
		}
		if err := s.DeleteByID(ctx, row.ID); err != nil && !errors.Is(err, dbutil.ErrNotFound) {
			return err
		}
	}
	return nil
}

func objectFromSQLC(row sqlc.StorageObject) Object {
	return Object{
		ID:          row.ID,
		Prefix:      row.Prefix,
		Filename:    row.Filename,
		ContentType: row.ContentType,
		SizeBytes:   row.SizeBytes,
		SHA256:      row.Sha256,
		AvailableAt: row.AvailableAt,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

// prefixPattern constrains a storage prefix to the lowercase slash-separated
// segments that make up a key namespace.
var prefixPattern = regexp.MustCompile(`^[a-z0-9]+(/[a-z0-9]+)*$`)

// cleanUploadFilename reduces a client filename to a safe key segment. It takes
// the base name (tolerating directory components and Windows separators) and
// trims surrounding space, then rejects what cannot be a usable single segment.
func cleanUploadFilename(name string) (string, error) {
	name = strings.ReplaceAll(name, `\`, `/`)
	name = path.Base(name)
	name = strings.TrimSpace(name)

	if name == "" || name == "." || name == ".." || name == "/" {
		return "", fmt.Errorf("%w: invalid upload filename", dbutil.ErrInvalidInput)
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return "", fmt.Errorf("%w: invalid upload filename", dbutil.ErrInvalidInput)
		}
	}
	return name, nil
}
