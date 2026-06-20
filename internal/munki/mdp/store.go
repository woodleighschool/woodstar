package mdp

import (
	"context"
	"fmt"
	"log/slog"
	"net/netip"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/storage"
)

// Store persists distribution points and their per-package mirror state.
type Store struct {
	db       *database.DB
	presence *Presence
	logger   *slog.Logger
}

// NewStore returns a distribution point store backed by db. The presence set is
// shared with the hub: the hub writes it, the store reads it to gate redirects.
func NewStore(db *database.DB, presence *Presence, logger *slog.Logger) *Store {
	return &Store{db: db, presence: presence, logger: logger}
}

// List returns distribution points in admin order with live presence.
func (s *Store) List(
	ctx context.Context,
	params DistributionPointListParams,
) ([]DistributionPoint, int, error) {
	where, args := distributionPointListWhere(params)
	listQuery := distributionPointListQuery(params, where, args)

	records, count, err := dbutil.ListWithCount[distributionPointRow](
		ctx,
		s.db.Pool(),
		listQuery,
	)
	if err != nil {
		return nil, 0, err
	}

	points := make([]DistributionPoint, len(records))
	for i, record := range records {
		points[i] = s.distributionPointFromRow(record)
	}
	return points, count, nil
}

// GetByID returns one distribution point with its per-package mirror state.
func (s *Store) GetByID(ctx context.Context, id int64) (*DistributionPointDetail, error) {
	row, err := dbutil.GetOne[distributionPointRow](
		ctx,
		s.db.Pool(),
		distributionPointSelectSQL+"\nWHERE c.id = $1",
		id,
	)
	if err != nil {
		return nil, err
	}

	qrows, err := s.db.Pool().Query(ctx, packageStatesSQL, id)
	if err != nil {
		return nil, err
	}
	stateRows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[packageStateRow])
	if err != nil {
		return nil, err
	}

	detail := DistributionPointDetail{
		DistributionPoint: s.distributionPointFromRow(row),
		Packages:          make([]PackageState, len(stateRows)),
	}
	for i, state := range stateRows {
		detail.Packages[i] = packageStateFromRow(state)
	}
	return &detail, nil
}

// Create persists a new distribution point with key and returns it. The caller
// generates key and reveals it once; it is never read back through this model.
func (s *Store) Create(
	ctx context.Context,
	mutation DistributionPointMutation,
	key string,
) (*DistributionPoint, error) {
	if err := mutation.Validate(); err != nil {
		return nil, err
	}
	write := distributionPointCreateWrite{
		Name:          mutation.Name,
		Enabled:       mutation.Enabled,
		ClientCidrs:   clientCIDRs(mutation.ClientCIDRs),
		ClientBaseURL: mutation.ClientBaseURL,
		Key:           key,
	}
	row, err := dbutil.GetOne[distributionPointRow](
		ctx,
		s.db.Pool(),
		insertDistributionPointSQL,
		pgx.StructArgs(write),
	)
	if err != nil {
		return nil, dbutil.MutationError(err)
	}
	point := s.distributionPointFromRow(row)
	return &point, nil
}

// Update changes the admin-writable fields of a distribution point.
func (s *Store) Update(
	ctx context.Context,
	id int64,
	mutation DistributionPointMutation,
) (*DistributionPoint, error) {
	if err := mutation.Validate(); err != nil {
		return nil, err
	}
	write := distributionPointUpdateWrite{
		ID:            id,
		Name:          mutation.Name,
		Enabled:       mutation.Enabled,
		ClientCidrs:   clientCIDRs(mutation.ClientCIDRs),
		ClientBaseURL: mutation.ClientBaseURL,
	}
	row, err := dbutil.GetOne[distributionPointRow](
		ctx,
		s.db.Pool(),
		updateDistributionPointSQL,
		pgx.StructArgs(write),
	)
	if err != nil {
		return nil, dbutil.MutationError(err)
	}
	point := s.distributionPointFromRow(row)
	return &point, nil
}

// Delete removes a distribution point and its package states.
func (s *Store) Delete(ctx context.Context, id int64) error {
	tag, err := s.db.Pool().Exec(
		ctx,
		`DELETE FROM munki_distribution_points WHERE id = $1`,
		id,
	)
	if err != nil {
		return dbutil.DeleteConflict(err, "distribution point is still referenced")
	}
	if tag.RowsAffected() == 0 {
		return dbutil.ErrNotFound
	}
	return nil
}

// RotateKey replaces a distribution point's key, invalidating the old one.
func (s *Store) RotateKey(ctx context.Context, id int64, key string) error {
	tag, err := s.db.Pool().Exec(
		ctx,
		`UPDATE munki_distribution_points SET "key" = $1, updated_at = now() WHERE id = $2`,
		key,
		id,
	)
	if err != nil {
		return dbutil.MutationError(err)
	}
	if tag.RowsAffected() == 0 {
		return dbutil.ErrNotFound
	}
	return nil
}

// Reorder sets distribution point order from an exact permutation of the
// existing ids, persisted two-phase to satisfy the unique position constraint.
func (s *Store) Reorder(ctx context.Context, orderedIDs []int64) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT id FROM munki_distribution_points ORDER BY position, id`)
		if err != nil {
			return err
		}
		currentIDs, err := pgx.CollectRows(rows, pgx.RowTo[int64])
		if err != nil {
			return err
		}
		if !dbutil.SameInt64Set(orderedIDs, currentIDs) {
			return fmt.Errorf(
				"%w: ordered_ids must exactly match existing distribution point IDs",
				dbutil.ErrInvalidInput,
			)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE munki_distribution_points c
			SET position = -ordered.position
			FROM unnest($1::bigint[]) WITH ORDINALITY AS ordered(id, position)
			WHERE c.id = ordered.id`,
			orderedIDs,
		); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE munki_distribution_points SET position = -position - 1`)
		return err
	})
}

// AuthenticateWorker resolves a bearer key to its distribution point identity.
func (s *Store) AuthenticateWorker(ctx context.Context, key string) (*DistributionPoint, error) {
	row, err := dbutil.GetOne[distributionPointRow](
		ctx,
		s.db.Pool(),
		distributionPointSelectSQL+"\nWHERE c.\"key\" = $1",
		key,
	)
	if err != nil {
		return nil, err
	}
	point := s.distributionPointFromRow(row)
	return &point, nil
}

// ResolveForClient returns the first eligible, online distribution point for a
// client IP and package, or nil when Woodstar should serve the file itself.
// Eligibility is a database filter; liveness is the in-memory presence set, so a
// just-disconnected point is skipped before its stored state reflects the drop.
func (s *Store) ResolveForClient(
	ctx context.Context,
	clientIP netip.Addr,
	packageID int64,
) (*ResolvedPoint, error) {
	qrows, err := s.db.Pool().Query(ctx, eligibleDistributionPointsSQL, clientIP, packageID)
	if err != nil {
		return nil, err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[resolvedPointRow])
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if s.presence.Online(row.ID) {
			return &ResolvedPoint{ID: row.ID, Key: row.Key, ClientBaseURL: row.ClientBaseURL}, nil
		}
	}
	return nil, nil
}

// DesiredPackages returns every package whose installer is available to mirror.
func (s *Store) DesiredPackages(ctx context.Context) ([]DesiredPackage, error) {
	qrows, err := s.db.Pool().Query(ctx, desiredPackagesSQL)
	if err != nil {
		return nil, err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[desiredPackageRow])
	if err != nil {
		return nil, err
	}
	packages := make([]DesiredPackage, len(rows))
	for i, row := range rows {
		pkg := DesiredPackage{
			PackageID: row.PackageID,
			Filename:  row.Filename,
		}
		if row.Sha256 != nil {
			pkg.SHA256 = *row.Sha256
		}
		if row.SizeBytes != nil {
			pkg.SizeBytes = *row.SizeBytes
		}
		packages[i] = pkg
	}
	return packages, nil
}

// InstallerObjectKey returns the storage key of a package's installer object.
func (s *Store) InstallerObjectKey(ctx context.Context, packageID int64) (string, error) {
	type installerObjectRow struct {
		Prefix   string `db:"prefix"`
		ID       int64  `db:"object_id"`
		Filename string `db:"filename"`
	}
	row, err := dbutil.GetOne[installerObjectRow](ctx, s.db.Pool(), installerObjectSQL, packageID)
	if err != nil {
		return "", err
	}
	return storage.Key(row.Prefix, row.ID, row.Filename), nil
}

// RecordPackageState upserts one package's mirror state for a distribution
// point. Eligibility is derived at read and redirect time by comparing the
// reported hash against Woodstar's current desired installer, so a stale or
// removed package stops matching on its own and needs no separate cleanup.
func (s *Store) RecordPackageState(
	ctx context.Context,
	distributionPointID int64,
	packageID int64,
	status PackageStatus,
	sha256 string,
	errMessage string,
) error {
	_, err := s.db.Pool().Exec(ctx, upsertPackageStateSQL, pgx.NamedArgs{
		"distribution_point_id": distributionPointID,
		"package_id":            packageID,
		"status":                string(status),
		"reported_sha256":       reportedSHA256(sha256),
		"error":                 errMessage,
	})
	return err
}

// clientCIDRs coerces a nil slice to empty so the not-null text[] column takes
// an empty array rather than SQL NULL.
func clientCIDRs(cidrs []string) []string {
	if cidrs == nil {
		return []string{}
	}
	return cidrs
}

func (s *Store) distributionPointFromRow(row distributionPointRow) DistributionPoint {
	return DistributionPoint{
		ID:            row.ID,
		Name:          row.Name,
		Enabled:       row.Enabled,
		Position:      row.Position,
		ClientCIDRs:   row.ClientCidrs,
		ClientBaseURL: row.ClientBaseURL,
		Online:        s.presence.Online(row.ID),
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}

func packageStateFromRow(row packageStateRow) PackageState {
	return PackageState{
		PackageID:       row.PackageID,
		SoftwareID:      row.SoftwareID,
		DisplayName:     row.DisplayName,
		Version:         row.Version,
		SoftwareIconURL: softwareIconURL(row.SoftwareID, row.SoftwareIconObjectID),
		Status:          PackageStatus(row.Status),
		Error:           row.Error,
	}
}

func softwareIconURL(softwareID int64, objectID *int64) string {
	if objectID == nil {
		return ""
	}
	return "/api/munki/software/" + strconv.FormatInt(softwareID, 10) + "/icon"
}

func reportedSHA256(sha256 string) *string {
	if sha256 == "" {
		return nil
	}
	return &sha256
}

// distributionPointRow is the scan target for the munki_distribution_points projection.
type distributionPointRow struct {
	ID            int64     `db:"id"`
	Name          string    `db:"name"`
	Enabled       bool      `db:"enabled"`
	Position      int32     `db:"position"`
	ClientCidrs   []string  `db:"client_cidrs"`
	ClientBaseURL string    `db:"client_base_url"`
	Key           string    `db:"key"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

// distributionPointCreateWrite carries all admin-writable fields for INSERT.
type distributionPointCreateWrite struct {
	Name          string   `db:"name"`
	Enabled       bool     `db:"enabled"`
	ClientCidrs   []string `db:"client_cidrs"`
	ClientBaseURL string   `db:"client_base_url"`
	Key           string   `db:"key"`
}

// distributionPointUpdateWrite carries the admin-writable fields for UPDATE (no key rotation).
type distributionPointUpdateWrite struct {
	ID            int64    `db:"id"`
	Name          string   `db:"name"`
	Enabled       bool     `db:"enabled"`
	ClientCidrs   []string `db:"client_cidrs"`
	ClientBaseURL string   `db:"client_base_url"`
}

// packageStateRow is the scan target for the per-distribution-point package state query.
type packageStateRow struct {
	PackageID            int64  `db:"package_id"`
	SoftwareID           int64  `db:"software_id"`
	DisplayName          string `db:"display_name"`
	Version              string `db:"version"`
	SoftwareIconObjectID *int64 `db:"software_icon_object_id"`
	Status               string `db:"status"`
	Error                string `db:"error"`
}

// resolvedPointRow is the minimal scan target for the client-resolution query.
type resolvedPointRow struct {
	ID            int64  `db:"id"`
	Key           string `db:"key"`
	ClientBaseURL string `db:"client_base_url"`
}

// desiredPackageRow is the scan target for the desired-packages query.
type desiredPackageRow struct {
	PackageID int64   `db:"package_id"`
	Filename  string  `db:"filename"`
	Sha256    *string `db:"sha256"`
	SizeBytes *int64  `db:"size_bytes"`
}

const insertDistributionPointSQL = `
INSERT INTO munki_distribution_points (
	name,
	enabled,
	position,
	client_cidrs,
	client_base_url,
	"key"
)
VALUES (
	@name,
	@enabled,
	(SELECT COALESCE(MAX(position) + 1, 0) FROM munki_distribution_points),
	@client_cidrs::text[],
	@client_base_url,
	@key
)
RETURNING
	id,
	name,
	enabled,
	position,
	client_cidrs,
	client_base_url,
	"key",
	created_at,
	updated_at`

const updateDistributionPointSQL = `
UPDATE munki_distribution_points
SET
	name = @name,
	enabled = @enabled,
	client_cidrs = @client_cidrs::text[],
	client_base_url = @client_base_url,
	updated_at = now()
WHERE id = @id
RETURNING
	id,
	name,
	enabled,
	position,
	client_cidrs,
	client_base_url,
	"key",
	created_at,
	updated_at`

const eligibleDistributionPointsSQL = `
SELECT
	c.id,
	c."key",
	c.client_base_url
FROM munki_distribution_points c
WHERE c.enabled
  AND c.client_base_url <> ''
  AND $1::inet <<= ANY (c.client_cidrs::inet[])
  AND EXISTS (
      SELECT 1
      FROM munki_distribution_package_states s
      JOIN munki_packages p ON p.id = s.package_id
      JOIN storage_objects o ON o.id = p.installer_object_id
      WHERE s.distribution_point_id = c.id
        AND s.package_id = $2
        AND s.status = 'current'
        AND o.available_at IS NOT NULL
        AND o.sha256 = s.reported_sha256
  )
ORDER BY c.position, c.id`

const packageStatesSQL = `
SELECT
	p.id AS package_id,
	sw.id AS software_id,
	sw.name AS display_name,
	p.version,
	sw.icon_object_id AS software_icon_object_id,
	CASE
		WHEN s.package_id IS NULL THEN 'pending'
		WHEN s.status = 'error' THEN 'error'
		WHEN o.sha256 = s.reported_sha256 THEN 'current'
		ELSE 'syncing'
	END::text AS status,
	COALESCE(s.error, '') AS error
FROM munki_packages p
JOIN munki_software sw ON sw.id = p.software_id
JOIN storage_objects o ON o.id = p.installer_object_id
LEFT JOIN munki_distribution_package_states s
	ON s.package_id = p.id
	AND s.distribution_point_id = $1
WHERE o.available_at IS NOT NULL
  AND o.sha256 IS NOT NULL
  AND o.size_bytes IS NOT NULL
ORDER BY sw.name, p.version`

const desiredPackagesSQL = `
SELECT
	p.id AS package_id,
	o.filename,
	o.sha256,
	o.size_bytes
FROM munki_packages p
JOIN storage_objects o ON o.id = p.installer_object_id
WHERE o.available_at IS NOT NULL
  AND o.sha256 IS NOT NULL
  AND o.size_bytes IS NOT NULL
ORDER BY p.id`

const installerObjectSQL = `
SELECT
	o.prefix,
	o.id AS object_id,
	o.filename
FROM munki_packages p
JOIN storage_objects o ON o.id = p.installer_object_id
WHERE p.id = $1
  AND o.available_at IS NOT NULL`

const upsertPackageStateSQL = `
INSERT INTO munki_distribution_package_states (
	distribution_point_id,
	package_id,
	status,
	reported_sha256,
	error
)
VALUES (
	@distribution_point_id,
	@package_id,
	@status,
	@reported_sha256,
	@error
)
ON CONFLICT (distribution_point_id, package_id) DO UPDATE
SET status = EXCLUDED.status,
    reported_sha256 = EXCLUDED.reported_sha256,
    error = EXCLUDED.error,
    updated_at = now()`
