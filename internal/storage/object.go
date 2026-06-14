package storage

import (
	"context"
	"errors"
	"fmt"
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

// Key is the object's storage key: <prefix>/<id>/<filename>.
func (o Object) Key() string {
	return fmt.Sprintf("%s/%d/%s", o.Prefix, o.ID, o.Filename)
}

// Available reports whether the bytes have been confirmed present.
func (o Object) Available() bool {
	return o.AvailableAt != nil
}

// ObjectStore is the database registry of stored objects.
type ObjectStore struct {
	db *database.DB
	q  *sqlc.Queries
}

// NewObjectStore returns a registry backed by db.
func NewObjectStore(db *database.DB) *ObjectStore {
	return &ObjectStore{db: db, q: db.Queries()}
}

// CreatePending inserts a pending object and returns it with its assigned id.
// The caller uploads to Object.Key() and then calls Confirm.
func (s *ObjectStore) CreatePending(ctx context.Context, prefix, filename, contentType string) (*Object, error) {
	prefix = strings.Trim(strings.TrimSpace(prefix), "/")
	if !prefixPattern.MatchString(prefix) {
		return nil, fmt.Errorf("invalid storage prefix %q", prefix)
	}
	row, err := s.q.CreateStorageObject(ctx, sqlc.CreateStorageObjectParams{
		Prefix:      prefix,
		Filename:    sanitizeFilename(filename),
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

var (
	prefixPattern  = regexp.MustCompile(`^[a-z0-9]+(/[a-z0-9]+)*$`)
	filenameUnsafe = regexp.MustCompile(`[^A-Za-z0-9._-]+`)
	dashAroundDot  = regexp.MustCompile(`-*\.-*`)
)

// sanitizeFilename reduces a client filename to a safe key segment, keeping the
// extension readable. It never returns empty.
func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	name = filenameUnsafe.ReplaceAllString(name, "-")
	name = dashAroundDot.ReplaceAllString(name, ".")
	name = strings.Trim(name, "-.")
	if name == "" {
		return "file"
	}
	return name
}
