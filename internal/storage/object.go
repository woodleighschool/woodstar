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
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// Object is a row in the storage registry: one stored (or pending) blob. The
// byte key is derived, never stored, so the path format lives in one place.
type Object struct {
	ID          int64      `db:"id"`
	Prefix      string     `db:"prefix"`
	Filename    string     `db:"filename"`
	ContentType string     `db:"content_type"`
	SizeBytes   *int64     `db:"size_bytes"`
	SHA256      *string    `db:"sha256"`
	AvailableAt *time.Time `db:"available_at"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}

const objectSelectSQL = `SELECT id, prefix, filename, content_type, size_bytes, sha256, available_at, created_at, updated_at
FROM storage_objects`

// objectUnrefRow is the minimal projection used by DeleteUnreferenced.
type objectUnrefRow struct {
	Prefix   string `db:"prefix"`
	ID       int64  `db:"id"`
	Filename string `db:"filename"`
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
	backend Store
}

// NewObjectStore returns a registry backed by db.
func NewObjectStore(db *database.DB, backend Store) *ObjectStore {
	return &ObjectStore{db: db, backend: backend}
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
	const sql = `INSERT INTO storage_objects (prefix, filename, content_type)
VALUES (@prefix, @filename, @content_type)
RETURNING id, prefix, filename, content_type, size_bytes, sha256, available_at, created_at, updated_at`
	obj, err := dbutil.GetOne[Object](ctx, s.db.Pool(), sql, pgx.NamedArgs{
		"prefix":       prefix,
		"filename":     filename,
		"content_type": contentType,
	})
	if err != nil {
		return nil, dbutil.MutationError(err)
	}
	return &obj, nil
}

// Confirm records the landed object's size, sha, and content type, and marks it
// available. A content type of "" keeps whatever was set at creation.
func (s *ObjectStore) Confirm(ctx context.Context, id, size int64, contentType, sha256sum string) (*Object, error) {
	const sql = `UPDATE storage_objects
SET size_bytes = @size_bytes,
    sha256 = @sha256,
    content_type = COALESCE(NULLIF(@content_type::text, ''), content_type),
    available_at = now(),
    updated_at = now()
WHERE id = @id
RETURNING id, prefix, filename, content_type, size_bytes, sha256, available_at, created_at, updated_at`
	obj, err := dbutil.GetOne[Object](ctx, s.db.Pool(), sql, pgx.NamedArgs{
		"id":           id,
		"size_bytes":   &size,
		"sha256":       &sha256sum,
		"content_type": contentType,
	})
	if err != nil {
		return nil, dbutil.MutationError(err)
	}
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
	obj, err := dbutil.GetOne[Object](ctx, s.db.Pool(), objectSelectSQL+"\nWHERE id = $1", id)
	if err != nil {
		return nil, dbutil.GetError(err)
	}
	return &obj, nil
}

// ListByIDs returns objects keyed by id. Missing IDs are ignored.
func (s *ObjectStore) ListByIDs(ctx context.Context, ids []int64) (map[int64]Object, error) {
	rows, err := s.db.Pool().Query(ctx,
		objectSelectSQL+"\nWHERE id = ANY($1::bigint[])", ids)
	if err != nil {
		return nil, err
	}
	objects, err := pgx.CollectRows(rows, pgx.RowToStructByName[Object])
	if err != nil {
		return nil, err
	}
	result := make(map[int64]Object, len(objects))
	for _, obj := range objects {
		result[obj.ID] = obj
	}
	return result, nil
}

// ListByPrefix returns available objects under a prefix, newest first.
func (s *ObjectStore) ListByPrefix(
	ctx context.Context,
	prefix string,
	params dbutil.ListParams,
) ([]Object, int, error) {
	params = dbutil.CleanListParams(params)
	listQuery := dbutil.ListQuery{
		SelectSQL:    objectSelectSQL,
		WhereSQL:     "WHERE prefix = $1 AND available_at IS NOT NULL",
		Args:         []any{prefix},
		DefaultOrder: []dbutil.OrderExpr{{SQL: "created_at DESC"}, {SQL: "id DESC"}},
		Params:       params,
	}
	return dbutil.ListWithCount[Object](ctx, s.db.Pool(), listQuery)
}

// DeleteByID removes an object row. It fails with a conflict if a consumer FK
// still references it.
func (s *ObjectStore) DeleteByID(ctx context.Context, id int64) error {
	tag, err := s.db.Pool().Exec(ctx, `DELETE FROM storage_objects WHERE id = $1`, id)
	if err != nil {
		return dbutil.DeleteConflict(err, "storage object is still referenced")
	}
	if tag.RowsAffected() == 0 {
		return dbutil.ErrNotFound
	}
	return nil
}

// DeleteUnreferenced deletes backend bytes and rows for objects that no Munki
// resource references anymore. Failed backend deletes leave rows in place so the
// database does not claim cleanup happened when bytes still exist.
func (s *ObjectStore) DeleteUnreferenced(ctx context.Context, ids ...int64) error {
	const sql = `SELECT o.prefix, o.id, o.filename
FROM storage_objects o
WHERE o.id = ANY($1::bigint[])
  AND NOT EXISTS (SELECT 1 FROM munki_software s WHERE s.icon_object_id = o.id)
  AND NOT EXISTS (
      SELECT 1 FROM munki_packages p
      WHERE p.installer_object_id = o.id
  )`
	rows, err := s.db.Pool().Query(ctx, sql, ids)
	if err != nil {
		return err
	}
	unref, err := pgx.CollectRows(rows, pgx.RowToStructByName[objectUnrefRow])
	if err != nil {
		return err
	}
	for _, row := range unref {
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

// ReplacedObjectIDs returns the old object id when a pointer field now points
// at a different object or has been cleared.
func ReplacedObjectIDs(oldID, newID *int64) []int64 {
	if oldID == nil {
		return nil
	}
	if newID != nil && *oldID == *newID {
		return nil
	}
	return []int64{*oldID}
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
