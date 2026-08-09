package mdp

import (
	"context"
	"fmt"
	"log/slog"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/listing"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/storage"
)

// Store persists distribution points and their per-package mirror state.
type Store struct {
	db       *database.DB
	objects  *storage.ObjectStore
	presence *Presence
	logger   *slog.Logger
}

// NewStore returns a distribution point store backed by db.
func NewStore(db *database.DB, objects *storage.ObjectStore, logger *slog.Logger) *Store {
	return &Store{db: db, objects: objects, presence: NewPresence(), logger: logger}
}

// Presence returns the live worker presence set shared by selection and the
// worker protocol hub.
func (s *Store) Presence() *Presence {
	return s.presence
}

// List returns distribution points in admin order with live presence.
func (s *Store) List(
	ctx context.Context,
	params DistributionPointListParams,
) ([]DistributionPoint, int, error) {
	params.ListParams = listing.Normalize(params.ListParams)
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
		distributionPointSelectSQL()+"\nWHERE c.id = $1",
		id,
	)
	if err != nil {
		return nil, err
	}

	qrows, err := s.db.Pool().Query(ctx, `
SELECT
	p.id AS package_id,
	sw.id AS software_id,
	sw.name,
	p.version,
	sw.icon_object_id AS software_icon_object_id,
	CASE
		WHEN s.package_id IS NULL THEN 'pending'
		WHEN s.status = 'error' THEN 'error'
		WHEN o.sha256 = s.reported_sha256 THEN 'current'
		ELSE 'syncing'
	END::text AS status,
	COALESCE(s.error, '') AS error,
	(
		o.available_at IS NOT NULL
		AND o.sha256 IS NOT NULL
		AND o.size_bytes IS NOT NULL
	) AS installer_finalized
FROM munki_packages p
JOIN munki_software sw ON sw.id = p.software_id
JOIN storage_objects o ON o.id = p.installer_object_id
LEFT JOIN munki_distribution_package_states s
	ON s.package_id = p.id
	AND s.distribution_point_id = $1
ORDER BY sw.name, p.version`,
		id,
	)
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
		if !state.InstallerFinalized {
			return nil, fmt.Errorf("munki package %d installer object is not finalized", state.PackageID)
		}
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
	mutation.normalize()
	if err := mutation.validate(); err != nil {
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
		`
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
	updated_at`,
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
	mutation.normalize()
	if err := mutation.validate(); err != nil {
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
		`
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
	updated_at`,
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
		return fault.ErrNotFound
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
		return fault.ErrNotFound
	}
	return nil
}

// Reorder sets distribution point order from an exact permutation of the
// existing ids, persisted two-phase to satisfy the unique position constraint.
func (s *Store) Reorder(ctx context.Context, orderedIDs []int64) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		var updated, total int
		if err := tx.QueryRow(ctx, `
WITH ordered AS (
	SELECT id, position::int
	FROM unnest($1::bigint[]) WITH ORDINALITY AS input(id, position)
),
stats AS (
	SELECT
		(SELECT count(*) FROM munki_distribution_points) AS total,
		(SELECT count(*) FROM ordered) AS requested,
		(SELECT count(DISTINCT id) FROM ordered) AS distinct_requested,
		(
			SELECT count(*)
			FROM ordered
			JOIN munki_distribution_points c ON c.id = ordered.id
		) AS matched
),
updated AS (
	UPDATE munki_distribution_points c
	SET position = -ordered.position
	FROM ordered, stats
	WHERE stats.total = stats.requested
	  AND stats.requested = stats.distinct_requested
	  AND stats.requested = stats.matched
	  AND c.id = ordered.id
	RETURNING c.id
)
SELECT (SELECT count(*) FROM updated), (SELECT total FROM stats)`,
			orderedIDs,
		).Scan(&updated, &total); err != nil {
			return err
		}
		if updated != total {
			return fmt.Errorf(
				"%w: ordered_ids must exactly match existing distribution point IDs",
				fault.ErrInvalidInput,
			)
		}
		_, err := tx.Exec(ctx, `UPDATE munki_distribution_points SET position = -position - 1`)
		return err
	})
}

// AuthenticateWorker resolves a bearer key to its distribution point identity.
func (s *Store) AuthenticateWorker(ctx context.Context, key string) (*DistributionPoint, error) {
	row, err := dbutil.GetOne[distributionPointRow](
		ctx,
		s.db.Pool(),
		distributionPointSelectSQL()+"\nWHERE c.\"key\" = $1",
		key,
	)
	if err != nil {
		return nil, err
	}
	point := s.distributionPointFromRow(row)
	return &point, nil
}

// CandidatesForClients returns enabled distribution points whose client CIDRs
// contain each address. Every candidate slice is ordered by position and ID.
func (s *Store) CandidatesForClients(
	ctx context.Context,
	clientIPs []netip.Addr,
) (map[netip.Addr][]ClientCandidate, error) {
	addresses := make([]string, 0, len(clientIPs))
	candidates := make(map[netip.Addr][]ClientCandidate, len(clientIPs))
	seen := make(map[netip.Addr]struct{}, len(clientIPs))
	for _, clientIP := range clientIPs {
		clientIP = clientIP.Unmap()
		if !clientIP.IsValid() {
			continue
		}
		if _, ok := seen[clientIP]; ok {
			continue
		}
		seen[clientIP] = struct{}{}
		addresses = append(addresses, clientIP.String())
		candidates[clientIP] = []ClientCandidate{}
	}
	if len(addresses) == 0 {
		return candidates, nil
	}

	qrows, err := s.db.Pool().Query(ctx, `
SELECT
	input.client_ip,
	c.id,
	c.name,
	c."key",
	c.client_base_url
FROM unnest($1::text[]) WITH ORDINALITY AS input(client_ip, input_order)
JOIN munki_distribution_points c
	ON input.client_ip::inet <<= ANY (c.client_cidrs::inet[])
WHERE c.enabled
ORDER BY input.input_order, c.position, c.id`, addresses)
	if err != nil {
		return nil, err
	}
	defer qrows.Close()
	for qrows.Next() {
		var address string
		var candidate ClientCandidate
		if err := qrows.Scan(
			&address,
			&candidate.ID,
			&candidate.Name,
			&candidate.key,
			&candidate.clientBaseURL,
		); err != nil {
			return nil, err
		}
		clientIP, err := netip.ParseAddr(address)
		if err != nil {
			return nil, fmt.Errorf("parse distribution point candidate address %q: %w", address, err)
		}
		clientIP = clientIP.Unmap()
		candidates[clientIP] = append(candidates[clientIP], candidate)
	}
	if err := qrows.Err(); err != nil {
		return nil, err
	}
	return candidates, nil
}

// CandidatesForClient returns ordered enabled CIDR matches for one address.
func (s *Store) CandidatesForClient(
	ctx context.Context,
	clientIP netip.Addr,
) ([]ClientCandidate, error) {
	clientIP = clientIP.Unmap()
	byAddress, err := s.CandidatesForClients(ctx, []netip.Addr{clientIP})
	if err != nil {
		return nil, err
	}
	return byAddress[clientIP], nil
}

// ResolveForClient returns the first matching candidate eligible to serve the
// requested package, or nil when Woodstar should serve the file itself.
func (s *Store) ResolveForClient(
	ctx context.Context,
	clientIP netip.Addr,
	packageID int64,
) (*ResolvedPoint, error) {
	candidates, err := s.CandidatesForClient(ctx, clientIP)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.clientBaseURL != "" {
			ids = append(ids, candidate.ID)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}

	qrows, err := s.db.Pool().Query(ctx, `
SELECT s.distribution_point_id
FROM munki_distribution_package_states s
JOIN munki_packages p ON p.id = s.package_id
JOIN storage_objects o ON o.id = p.installer_object_id
WHERE s.distribution_point_id = ANY($1::bigint[])
  AND s.package_id = $2
  AND s.status = 'current'
  AND o.sha256 = s.reported_sha256`, ids, packageID)
	if err != nil {
		return nil, err
	}
	current := make(map[int64]struct{}, len(ids))
	for qrows.Next() {
		var id int64
		if err := qrows.Scan(&id); err != nil {
			qrows.Close()
			return nil, err
		}
		current[id] = struct{}{}
	}
	if err := qrows.Err(); err != nil {
		qrows.Close()
		return nil, err
	}
	qrows.Close()

	for _, candidate := range candidates {
		if candidate.clientBaseURL == "" {
			continue
		}
		if _, ok := current[candidate.ID]; !ok {
			continue
		}
		if worker, ok := s.presence.Worker(candidate.ID); ok && worker.Compatible {
			return &ResolvedPoint{
				ID:            candidate.ID,
				Key:           candidate.key,
				ClientBaseURL: candidate.clientBaseURL,
			}, nil
		}
	}
	return nil, nil
}

// DesiredPackages returns every installer-backed package to mirror.
func (s *Store) DesiredPackages(ctx context.Context) ([]DesiredPackage, error) {
	qrows, err := s.db.Pool().Query(ctx, `
SELECT
	p.id AS package_id,
	o.filename,
	o.sha256,
	o.size_bytes
FROM munki_packages p
JOIN storage_objects o ON o.id = p.installer_object_id
ORDER BY p.id`)
	if err != nil {
		return nil, err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[desiredPackageRow])
	if err != nil {
		return nil, err
	}
	packages := make([]DesiredPackage, len(rows))
	for i, row := range rows {
		if row.Sha256 == nil || row.SizeBytes == nil {
			return nil, fmt.Errorf("munki package %d installer object is not finalized", row.PackageID)
		}
		pkg := DesiredPackage{
			PackageID: row.PackageID,
			Filename:  row.Filename,
			SHA256:    *row.Sha256,
			SizeBytes: *row.SizeBytes,
		}
		packages[i] = pkg
	}
	return packages, nil
}

// InstallerObject returns the stored content for a package's installer.
func (s *Store) InstallerObject(ctx context.Context, packageID int64) (storage.Object, error) {
	var objectID *int64
	err := s.db.Pool().QueryRow(ctx, `
SELECT installer_object_id
FROM munki_packages
WHERE id = $1`, packageID).Scan(&objectID)
	if err != nil {
		return storage.Object{}, dbutil.GetError(err)
	}
	if objectID == nil {
		return storage.Object{}, fault.ErrNotFound
	}
	object, err := s.objects.GetByID(ctx, *objectID)
	if err != nil {
		return storage.Object{}, err
	}
	if !object.Available() || object.SizeBytes == nil || object.SHA256 == nil {
		return storage.Object{}, fmt.Errorf("munki package %d installer object is not finalized", packageID)
	}
	return *object, nil
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
	_, err := s.db.Pool().Exec(ctx, `
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
    updated_at = now()`,
		pgx.NamedArgs{
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
	var worker *DistributionPointWorker
	if current, ok := s.presence.Worker(row.ID); ok {
		worker = &current
	}
	return DistributionPoint{
		ID:            row.ID,
		Name:          row.Name,
		Enabled:       row.Enabled,
		Position:      row.Position,
		ClientCIDRs:   row.ClientCidrs,
		ClientBaseURL: row.ClientBaseURL,
		Worker:        worker,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}

func packageStateFromRow(row packageStateRow) PackageState {
	return PackageState{
		PackageID:       row.PackageID,
		SoftwareID:      row.SoftwareID,
		Name:            row.Name,
		Version:         row.Version,
		SoftwareIconURL: munkisoftware.IconURL(row.SoftwareIconObjectID),
		Status:          PackageStatus(row.Status),
		Error:           row.Error,
	}
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
	Name                 string `db:"name"`
	Version              string `db:"version"`
	SoftwareIconObjectID *int64 `db:"software_icon_object_id"`
	Status               string `db:"status"`
	Error                string `db:"error"`
	InstallerFinalized   bool   `db:"installer_finalized"`
}

// desiredPackageRow is the scan target for the desired-packages query.
type desiredPackageRow struct {
	PackageID int64   `db:"package_id"`
	Filename  string  `db:"filename"`
	Sha256    *string `db:"sha256"`
	SizeBytes *int64  `db:"size_bytes"`
}
