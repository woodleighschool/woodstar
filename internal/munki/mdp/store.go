package mdp

import (
	"context"
	"errors"
	"fmt"
	"net/netip"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/storage"
)

// Store persists distribution points and their per-package mirror state.
type Store struct {
	db       *database.DB
	q        *sqlc.Queries
	presence Presence
}

// NewStore returns a distribution point store backed by db.
func NewStore(db *database.DB) *Store {
	return &Store{db: db, q: db.Queries()}
}

// SetPresence wires the connection hub the store consults for live presence and
// client-redirect eligibility. It is set once during wiring, before serving.
func (s *Store) SetPresence(presence Presence) {
	s.presence = presence
}

// List returns distribution points in admin order with live presence.
func (s *Store) List(
	ctx context.Context,
	params DistributionPointListParams,
) ([]DistributionPoint, int, error) {
	where, args := distributionPointListWhere(params)
	listQuery := distributionPointListQuery(params, where, args)

	var count int
	countSQL, countArgs := listQuery.BuildCount()
	if err := s.db.Pool().QueryRow(ctx, countSQL, countArgs...).Scan(&count); err != nil {
		return nil, 0, err
	}
	query, args, err := listQuery.Build()
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[sqlc.MunkiDistributionPoint])
	if err != nil {
		return nil, 0, err
	}

	points := make([]DistributionPoint, len(records))
	for i, record := range records {
		points[i] = s.distributionPoint(record)
	}
	return points, count, nil
}

// GetByID returns one distribution point with its per-package mirror state.
func (s *Store) GetByID(ctx context.Context, id int64) (*DistributionPointDetail, error) {
	row, err := s.q.GetMunkiDistributionPointByID(ctx, sqlc.GetMunkiDistributionPointByIDParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	states, err := s.q.ListMunkiDistributionPackageStates(
		ctx,
		sqlc.ListMunkiDistributionPackageStatesParams{DistributionPointID: id},
	)
	if err != nil {
		return nil, err
	}
	detail := DistributionPointDetail{
		DistributionPoint: s.distributionPoint(row),
		Packages:          make([]PackageState, len(states)),
	}
	for i, state := range states {
		detail.Packages[i] = packageStateFromSQLC(state)
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
	row, err := s.q.CreateMunkiDistributionPoint(ctx, sqlc.CreateMunkiDistributionPointParams{
		Name:          mutation.Name,
		Enabled:       mutation.Enabled,
		ClientCidrs:   clientCIDRs(mutation.ClientCIDRs),
		ClientBaseURL: mutation.ClientBaseURL,
		Key:           key,
	})
	if err != nil {
		return nil, dbutil.MutationError(err)
	}
	point := s.distributionPoint(row)
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
	row, err := s.q.UpdateMunkiDistributionPoint(ctx, sqlc.UpdateMunkiDistributionPointParams{
		ID:            id,
		Name:          mutation.Name,
		Enabled:       mutation.Enabled,
		ClientCidrs:   clientCIDRs(mutation.ClientCIDRs),
		ClientBaseURL: mutation.ClientBaseURL,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, dbutil.MutationError(err)
	}
	point := s.distributionPoint(row)
	return &point, nil
}

// Delete removes a distribution point and its package states.
func (s *Store) Delete(ctx context.Context, id int64) error {
	_, err := s.q.DeleteMunkiDistributionPoint(ctx, sqlc.DeleteMunkiDistributionPointParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	return err
}

// RotateKey replaces a distribution point's key, invalidating the old one.
func (s *Store) RotateKey(ctx context.Context, id int64, key string) error {
	_, err := s.q.RotateMunkiDistributionPointKey(ctx, sqlc.RotateMunkiDistributionPointKeyParams{
		ID:  id,
		Key: key,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	if err != nil {
		return dbutil.MutationError(err)
	}
	return nil
}

// Reorder sets distribution point order from an exact permutation of the
// existing ids, persisted two-phase to satisfy the unique position constraint.
func (s *Store) Reorder(ctx context.Context, orderedIDs []int64) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		currentIDs, err := q.ListMunkiDistributionPointIDsByPosition(ctx)
		if err != nil {
			return err
		}
		if !dbutil.SameInt64Set(orderedIDs, currentIDs) {
			return fmt.Errorf(
				"%w: ordered_ids must exactly match existing distribution point IDs",
				dbutil.ErrInvalidInput,
			)
		}
		if err := q.SetMunkiDistributionPointPositions(
			ctx,
			sqlc.SetMunkiDistributionPointPositionsParams{OrderedIds: orderedIDs},
		); err != nil {
			return err
		}
		return q.NormalizeMunkiDistributionPointPositions(ctx)
	})
}

// AuthenticateWorker resolves a bearer key to its distribution point identity.
func (s *Store) AuthenticateWorker(ctx context.Context, key string) (*DistributionPoint, error) {
	row, err := s.q.GetMunkiDistributionPointByKey(ctx, sqlc.GetMunkiDistributionPointByKeyParams{Key: key})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	point := s.distributionPoint(row)
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
	rows, err := s.q.ListEligibleMunkiDistributionPointsForClient(
		ctx,
		sqlc.ListEligibleMunkiDistributionPointsForClientParams{ClientIP: clientIP, PackageID: packageID},
	)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if s.presence != nil && s.presence.Online(row.ID) {
			return &ResolvedPoint{ID: row.ID, Key: row.Key, ClientBaseURL: row.ClientBaseURL}, nil
		}
	}
	return nil, nil
}

// DesiredPackages returns every package whose installer is available to mirror.
func (s *Store) DesiredPackages(ctx context.Context) ([]DesiredPackage, error) {
	rows, err := s.q.ListDesiredMunkiPackages(ctx)
	if err != nil {
		return nil, err
	}
	packages := make([]DesiredPackage, len(rows))
	for i, row := range rows {
		packages[i] = desiredPackageFromSQLC(row)
	}
	return packages, nil
}

// InstallerObjectKey returns the storage key of a package's installer object.
func (s *Store) InstallerObjectKey(ctx context.Context, packageID int64) (string, error) {
	row, err := s.q.GetMunkiPackageInstallerObject(
		ctx,
		sqlc.GetMunkiPackageInstallerObjectParams{PackageID: packageID},
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", dbutil.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	obj := row.StorageObject
	return storage.Key(obj.Prefix, obj.ID, obj.Filename), nil
}

// RecordState persists the worker's package mirror state. Current eligibility is
// still checked against Woodstar's desired hash at read time and redirect time.
func (s *Store) RecordState(
	ctx context.Context,
	distributionPointID int64,
	report StateReport,
) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if _, err := q.GetMunkiDistributionPointByID(ctx, sqlc.GetMunkiDistributionPointByIDParams{
			ID: distributionPointID,
		}); errors.Is(err, pgx.ErrNoRows) {
			return dbutil.ErrNotFound
		} else if err != nil {
			return err
		}

		reportedIDs := make([]int64, len(report.Packages))
		for i, pkg := range report.Packages {
			reportedIDs[i] = pkg.PackageID
		}
		if err := q.DeleteMunkiDistributionPackageStatesNotIn(
			ctx,
			sqlc.DeleteMunkiDistributionPackageStatesNotInParams{
				DistributionPointID: distributionPointID,
				PackageIds:          reportedIDs,
			},
		); err != nil {
			return err
		}

		for _, pkg := range report.Packages {
			if err := q.UpsertMunkiDistributionPackageState(ctx, sqlc.UpsertMunkiDistributionPackageStateParams{
				DistributionPointID: distributionPointID,
				PackageID:           pkg.PackageID,
				Status:              string(reportedPackageStatus(pkg)),
				ReportedSha256:      reportedSHA256(pkg.SHA256),
				Error:               pkg.Error,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

// clientCIDRs coerces a nil slice to empty so the not-null text[] column takes
// an empty array rather than SQL NULL.
func clientCIDRs(cidrs []string) []string {
	if cidrs == nil {
		return []string{}
	}
	return cidrs
}

func (s *Store) distributionPoint(row sqlc.MunkiDistributionPoint) DistributionPoint {
	return DistributionPoint{
		ID:            row.ID,
		Name:          row.Name,
		Enabled:       row.Enabled,
		Position:      row.Position,
		ClientCIDRs:   row.ClientCidrs,
		ClientBaseURL: row.ClientBaseURL,
		Online:        online(s.presence, row.ID),
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}

func packageStateFromSQLC(row sqlc.ListMunkiDistributionPackageStatesRow) PackageState {
	return PackageState{
		PackageID:   row.PackageID,
		DisplayName: row.DisplayName,
		Version:     row.Version,
		IconURL:     munkiIconURL(row.IconObjectID),
		Status:      PackageStatus(row.Status),
		Error:       row.Error,
	}
}

func desiredPackageFromSQLC(row sqlc.ListDesiredMunkiPackagesRow) DesiredPackage {
	pkg := DesiredPackage{
		PackageID:   row.PackageID,
		Filename:    row.Filename,
		DisplayName: row.DisplayName,
		Version:     row.Version,
	}
	if row.Sha256 != nil {
		pkg.SHA256 = *row.Sha256
	}
	if row.SizeBytes != nil {
		pkg.SizeBytes = *row.SizeBytes
	}
	return pkg
}

func reportedPackageStatus(pkg ReportedPackage) PackageStatus {
	if pkg.Status != "" {
		return pkg.Status
	}
	if pkg.Error != "" {
		return PackageStatusError
	}
	return PackageStatusCurrent
}

func reportedSHA256(sha256 string) *string {
	if sha256 == "" {
		return nil
	}
	return &sha256
}
