package storage

import (
	"context"
	"errors"
	"fmt"
	"mime"
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
	ID                int64      `db:"id"`
	Prefix            string     `db:"prefix"`
	Filename          string     `db:"filename"`
	ContentType       string     `db:"content_type"`
	SizeBytes         *int64     `db:"size_bytes"`
	SHA256            *string    `db:"sha256"`
	AvailableAt       *time.Time `db:"available_at"`
	MultipartUploadID *string    `db:"multipart_upload_id"`
	CreatedAt         time.Time  `db:"created_at"`
	UpdatedAt         time.Time  `db:"updated_at"`
}

const objectSelectSQL = `SELECT id, prefix, filename, content_type, size_bytes, sha256, available_at, multipart_upload_id, created_at, updated_at
FROM storage_objects`

// Key builds a storage key from its parts: <prefix>/<id>/<filename>. This is the
// one place the key format lives.
func Key(prefix string, id int64, filename string) string {
	return fmt.Sprintf("%s/%d/%s", prefix, id, filename)
}

// Key is the object's storage key.
func (o Object) Key() string {
	return Key(o.Prefix, o.ID, o.Filename)
}

// Available reports whether the bytes have been finalized.
func (o Object) Available() bool {
	return o.AvailableAt != nil
}

// SHA256Value returns the recorded hash, or "" while the object is pending.
func (o Object) SHA256Value() string {
	if o.SHA256 == nil {
		return ""
	}
	return *o.SHA256
}

// SizeBytesValue returns the recorded byte length, or 0 while the object is pending.
func (o Object) SizeBytesValue() int64 {
	if o.SizeBytes == nil {
		return 0
	}
	return *o.SizeBytes
}

// SizeKBValue returns the rounded-up recorded size in KiB.
func (o Object) SizeKBValue() int64 {
	sizeBytes := o.SizeBytesValue()
	if sizeBytes <= 0 {
		return 0
	}
	return (sizeBytes + 1023) / 1024
}

// ObjectStore is the database registry of stored objects.
type ObjectStore struct {
	db      *database.DB
	backend objectBackend
}

type objectBackend interface {
	Delete(ctx context.Context, key string) error
	PresignGet(ctx context.Context, key string, ttl time.Duration, opts GetOptions) (string, error)
}

// NewObjectStore returns a registry backed by db.
func NewObjectStore(db *database.DB, backend objectBackend) *ObjectStore {
	return &ObjectStore{db: db, backend: backend}
}

// CreatePending reserves an object in the registry without classifying content.
func (s *ObjectStore) CreatePending(ctx context.Context, prefix, filename string) (*Object, error) {
	if !prefixPattern.MatchString(prefix) {
		return nil, fmt.Errorf("%w: invalid storage prefix %q", dbutil.ErrInvalidInput, prefix)
	}
	filename = normalizeUploadFilename(filename)
	if err := validateUploadFilename(filename); err != nil {
		return nil, err
	}
	const sql = `INSERT INTO storage_objects (prefix, filename)
VALUES (@prefix, @filename)
RETURNING id, prefix, filename, content_type, size_bytes, sha256, available_at, multipart_upload_id, created_at, updated_at`
	obj, err := dbutil.GetOne[Object](ctx, s.db.Pool(), sql, pgx.NamedArgs{
		"prefix":   prefix,
		"filename": filename,
	})
	if err != nil {
		return nil, dbutil.MutationError(err)
	}
	return &obj, nil
}

// MarkAvailable records application-derived representation metadata for an object.
func (s *ObjectStore) MarkAvailable(
	ctx context.Context,
	id int64,
	sizeBytes int64,
	contentType string,
	sha256sum string,
) (*Object, error) {
	contentType, err := normalizeContentType(contentType)
	if err != nil {
		return nil, err
	}
	if sizeBytes < 0 || sha256sum == "" {
		return nil, fmt.Errorf("%w: incomplete storage object metadata", dbutil.ErrInvalidInput)
	}
	const sql = `UPDATE storage_objects
SET size_bytes = @size_bytes,
    sha256 = @sha256,
    content_type = @content_type,
    available_at = now(),
    updated_at = now()
WHERE id = @id
RETURNING id, prefix, filename, content_type, size_bytes, sha256, available_at, multipart_upload_id, created_at, updated_at`
	obj, err := dbutil.GetOne[Object](ctx, s.db.Pool(), sql, pgx.NamedArgs{
		"id":           id,
		"size_bytes":   &sizeBytes,
		"sha256":       &sha256sum,
		"content_type": contentType,
	})
	if err != nil {
		return nil, dbutil.MutationError(err)
	}
	return &obj, nil
}

// RecordMultipartUploadID records the provider upload ID, or returns the ID
// already recorded by a concurrent or retried creation request.
func (s *ObjectStore) RecordMultipartUploadID(
	ctx context.Context,
	id int64,
	uploadID string,
) (string, bool, error) {
	uploadID = strings.TrimSpace(uploadID)
	if uploadID == "" {
		return "", false, fmt.Errorf("%w: multipart upload ID is blank", dbutil.ErrInvalidInput)
	}
	var recorded string
	err := s.db.Pool().QueryRow(ctx, `
UPDATE storage_objects
SET multipart_upload_id = $2,
    updated_at = now()
WHERE id = $1
  AND available_at IS NULL
  AND multipart_upload_id IS NULL
RETURNING multipart_upload_id`, id, uploadID).Scan(&recorded)
	if err == nil {
		return recorded, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", false, dbutil.MutationError(err)
	}
	object, err := s.GetByID(ctx, id)
	if err != nil {
		return "", false, err
	}
	if object.MultipartUploadID != nil {
		return *object.MultipartUploadID, false, nil
	}
	return "", false, fmt.Errorf("%w: storage object is already finalized", dbutil.ErrInvalidInput)
}

// ClearMultipartUploadID closes the recorded provider upload after assembly.
func (s *ObjectStore) ClearMultipartUploadID(ctx context.Context, id int64, uploadID string) error {
	tag, err := s.db.Pool().Exec(ctx, `
UPDATE storage_objects
SET multipart_upload_id = NULL,
    updated_at = now()
WHERE id = $1
  AND multipart_upload_id = $2`, id, uploadID)
	if err != nil {
		return dbutil.MutationError(err)
	}
	if tag.RowsAffected() > 0 {
		return nil
	}
	object, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if object.MultipartUploadID == nil {
		return nil
	}
	return fmt.Errorf("%w: multipart upload ID changed", dbutil.ErrConflict)
}

// ContentURL returns a direct read URL with the registry's content type.
func (s *ObjectStore) ContentURL(ctx context.Context, object Object) (string, error) {
	if s.backend == nil {
		return "", errors.New("storage backend is not configured")
	}
	return s.backend.PresignGet(ctx, object.Key(), 0, GetOptions{ContentType: object.ContentType})
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
	params = dbutil.NormalizeListParams(params)
	listQuery := dbutil.ListQuery{
		SelectSQL: objectSelectSQL,
		WhereSQL:  "WHERE prefix = $1 AND available_at IS NOT NULL",
		Args:      []any{prefix},
		DefaultOrder: []dbutil.OrderExpr{
			{SQL: "created_at", Descending: true},
			{SQL: "id", Descending: true},
		},
		Params: params,
	}
	return dbutil.ListWithCount[Object](ctx, s.db.Pool(), listQuery)
}

// Delete removes one registry row and its bytes. Foreign keys decide whether
// another row still uses the object.
func (s *ObjectStore) Delete(ctx context.Context, id int64) error {
	const sql = `DELETE FROM storage_objects
WHERE id = $1
RETURNING id, prefix, filename, content_type, size_bytes, sha256, available_at, multipart_upload_id, created_at, updated_at`
	object, err := dbutil.GetOne[Object](ctx, s.db.Pool(), sql, id)
	if err != nil {
		return dbutil.DeleteConflict(err, "storage object is still referenced")
	}
	if s.backend == nil {
		return nil
	}
	if err := s.backend.Delete(ctx, object.Key()); err != nil {
		return fmt.Errorf("delete %q: %w", object.Key(), err)
	}
	return nil
}

func normalizeContentType(value string) (string, error) {
	mediaType, params, err := mime.ParseMediaType(value)
	if err != nil {
		return "", fmt.Errorf("%w: invalid content type: %w", dbutil.ErrInvalidInput, err)
	}
	value = mime.FormatMediaType(mediaType, params)
	if value == "" {
		return "", fmt.Errorf("%w: invalid content type", dbutil.ErrInvalidInput)
	}
	return value, nil
}

// prefixPattern constrains a storage prefix to the lowercase slash-separated
// segments that make up a key namespace.
var prefixPattern = regexp.MustCompile(`^[a-z0-9]+(/[a-z0-9]+)*$`)

func normalizeUploadFilename(name string) string {
	name = strings.ReplaceAll(name, `\`, `/`)
	name = path.Base(name)
	return strings.TrimSpace(name)
}

func validateUploadFilename(name string) error {
	if name == "" || name == "." || name == ".." || name == "/" {
		return fmt.Errorf("%w: invalid upload filename", dbutil.ErrInvalidInput)
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("%w: invalid upload filename", dbutil.ErrInvalidInput)
		}
	}
	return nil
}
