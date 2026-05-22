package reports

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/platforms"
	"github.com/woodleighschool/woodstar/internal/scope"
)

// Store persists saved reports and their per-host result snapshots.
type Store struct {
	db     *database.DB
	q      *sqlc.Queries
	scopes *scope.Store
}

// NewStore returns a report store backed by db.
func NewStore(db *database.DB) *Store {
	return &Store{db: db, q: sqlc.New(db.Pool()), scopes: scope.NewStore(db)}
}

// List returns saved reports matching params.
func (s *Store) List(ctx context.Context, params ReportListParams) ([]Report, int, error) {
	params = cleanReportListParams(params)
	where, args := reportListWhere(params)

	var count int
	if err := s.db.Pool().QueryRow(ctx, "SELECT count(*) FROM reports "+where, args...).Scan(&count); err != nil {
		return nil, 0, err
	}

	query, args, err := reportListSQL(where, args, params)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	reports := make([]Report, 0)
	reportIDs := make([]int64, 0)
	for rows.Next() {
		report, err := scanReport(rows)
		if err != nil {
			return nil, 0, err
		}
		reports = append(reports, *report)
		reportIDs = append(reportIDs, report.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	scopes, err := s.scopes.LoadReports(ctx, reportIDs)
	if err != nil {
		return nil, 0, err
	}
	for i := range reports {
		reports[i].LabelScope = scopes[reports[i].ID]
	}
	return reports, count, nil
}

// GetByID returns a saved report by database ID.
func (s *Store) GetByID(ctx context.Context, id int64) (*Report, error) {
	report, err := s.getByID(ctx, id)
	if err != nil {
		return nil, err
	}
	labelScope, err := s.scopes.LoadReport(ctx, report.ID)
	if err != nil {
		return nil, err
	}
	report.LabelScope = labelScope
	return report, nil
}

func (s *Store) getByID(ctx context.Context, id int64) (*Report, error) {
	row, err := s.q.GetReportByID(ctx, sqlc.GetReportByIDParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return reportFromSQLC(row), nil
}

// Create inserts a saved report.
func (s *Store) Create(ctx context.Context, params ReportCreate) (*Report, error) {
	params, err := cleanReportCreate(params)
	if err != nil {
		return nil, err
	}

	var created *Report
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		row, err := s.q.WithTx(tx).CreateReport(ctx, sqlc.CreateReportParams{
			Name:              params.Name,
			Description:       params.Description,
			Query:             params.Query,
			Platforms:         toSQLCPlatforms(params.Platforms),
			MinOsqueryVersion: params.MinOsqueryVersion,
			ScheduleInterval:  int32(params.ScheduleInterval),
			CreatedByUserID:   params.CreatedByUserID,
		})
		if err != nil {
			if dbutil.IsUniqueViolation(err) {
				return dbutil.ErrAlreadyExists
			}
			return err
		}
		report := reportFromSQLC(row)
		if err := s.scopes.ReplaceReport(ctx, tx, report.ID, params.LabelScope); err != nil {
			return err
		}
		report.LabelScope = scope.NormalizeLabelScope(params.LabelScope)
		created = report
		return nil
	})
	return created, err
}

// Update replaces a saved report.
func (s *Store) Update(ctx context.Context, id int64, params ReportUpdate) (*Report, error) {
	cleaned, err := cleanReportUpdate(params)
	if err != nil {
		return nil, err
	}

	var updated *Report
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		row, err := s.q.WithTx(tx).UpdateReport(ctx, sqlc.UpdateReportParams{
			Name:              cleaned.Name,
			Description:       cleaned.Description,
			Query:             cleaned.Query,
			Platforms:         toSQLCPlatforms(cleaned.Platforms),
			MinOsqueryVersion: cleaned.MinOsqueryVersion,
			ScheduleInterval:  int32(cleaned.ScheduleInterval),
			ID:                id,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return dbutil.ErrNotFound
		}
		if err != nil {
			if dbutil.IsUniqueViolation(err) {
				return dbutil.ErrAlreadyExists
			}
			return err
		}
		report := reportFromSQLC(row)
		if err := s.scopes.ReplaceReport(ctx, tx, report.ID, cleaned.LabelScope); err != nil {
			return err
		}
		report.LabelScope = scope.NormalizeLabelScope(cleaned.LabelScope)
		updated = report
		return nil
	})
	return updated, err
}

// Delete removes a saved report.
func (s *Store) Delete(ctx context.Context, id int64) error {
	_, err := s.q.DeleteReport(ctx, sqlc.DeleteReportParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	if err != nil {
		return err
	}
	return nil
}

// DeleteMany removes multiple saved reports. Missing IDs are ignored for bulk idempotency.
func (s *Store) DeleteMany(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	deletedIDs, err := s.q.DeleteReports(ctx, sqlc.DeleteReportsParams{Ids: ids})
	if err != nil {
		return 0, err
	}
	return len(deletedIDs), nil
}

// ScheduledForHost returns scheduled reports applicable to host.
func (s *Store) ScheduledForHost(ctx context.Context, host *hosts.Host) ([]Report, error) {
	rows, err := s.q.ListScheduledReportsForHost(ctx, sqlc.ListScheduledReportsForHostParams{HostID: host.ID})
	if err != nil {
		return nil, err
	}
	reports := make([]Report, 0, len(rows))
	for _, row := range rows {
		reports = append(reports, *reportFromSQLC(row))
	}
	return reports, nil
}

func cleanReportCreate(params ReportCreate) (ReportCreate, error) {
	params.Name = strings.TrimSpace(params.Name)
	params.Description = strings.TrimSpace(params.Description)
	params.Query = strings.TrimSpace(params.Query)
	targets, err := platforms.CleanTargets(params.Platforms)
	if err != nil {
		return ReportCreate{}, fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	params.Platforms = targets
	params.MinOsqueryVersion = dbutil.CleanStringPtr(params.MinOsqueryVersion)
	params.LabelScope = scope.NormalizeLabelScope(params.LabelScope)
	if params.Name == "" {
		return ReportCreate{}, fmt.Errorf("%w: name is required", dbutil.ErrInvalidInput)
	}
	if params.Query == "" {
		return ReportCreate{}, fmt.Errorf("%w: query is required", dbutil.ErrInvalidInput)
	}
	if params.ScheduleInterval < 0 {
		return ReportCreate{}, fmt.Errorf("%w: schedule interval cannot be negative", dbutil.ErrInvalidInput)
	}
	return params, nil
}

func cleanReportUpdate(params ReportUpdate) (ReportUpdate, error) {
	cleaned, err := cleanReportCreate(ReportCreate{
		Name:              params.Name,
		Description:       params.Description,
		Query:             params.Query,
		Platforms:         params.Platforms,
		MinOsqueryVersion: params.MinOsqueryVersion,
		ScheduleInterval:  params.ScheduleInterval,
		LabelScope:        params.LabelScope,
	})
	if err != nil {
		return ReportUpdate{}, err
	}
	return ReportUpdate{
		Name:              cleaned.Name,
		Description:       cleaned.Description,
		Query:             cleaned.Query,
		Platforms:         cleaned.Platforms,
		MinOsqueryVersion: cleaned.MinOsqueryVersion,
		ScheduleInterval:  cleaned.ScheduleInterval,
		LabelScope:        cleaned.LabelScope,
	}, nil
}

func cleanReportListParams(params ReportListParams) ReportListParams {
	params.ListParams = dbutil.CleanListParams(params.ListParams)
	params.Platform = platforms.CleanPlatform(params.Platform)
	return params
}

func scanReport(row pgx.Row) (*Report, error) {
	var report Report
	var sqlcPlatforms []sqlc.Platform
	err := row.Scan(
		&report.ID,
		&report.Name,
		&report.Description,
		&report.Query,
		&sqlcPlatforms,
		&report.MinOsqueryVersion,
		&report.ScheduleInterval,
		&report.CreatedByUserID,
		&report.CreatedAt,
		&report.UpdatedAt,
	)
	report.Platforms = platformsFromSQLC(sqlcPlatforms)
	return &report, err
}

func reportFromSQLC(row sqlc.Report) *Report {
	return &Report{
		ID:                row.ID,
		Name:              row.Name,
		Description:       row.Description,
		Query:             row.Query,
		Platforms:         platformsFromSQLC(row.Platforms),
		MinOsqueryVersion: row.MinOsqueryVersion,
		ScheduleInterval:  int(row.ScheduleInterval),
		CreatedByUserID:   row.CreatedByUserID,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func toSQLCPlatforms(values []platforms.Platform) []sqlc.Platform {
	out := make([]sqlc.Platform, len(values))
	for i, value := range values {
		out[i] = sqlc.Platform(value)
	}
	return out
}

func platformsFromSQLC(values []sqlc.Platform) []platforms.Platform {
	out := make([]platforms.Platform, len(values))
	for i, value := range values {
		out[i] = platforms.Platform(value)
	}
	return out
}

func reportListWhere(params ReportListParams) (string, []any) {
	clauses := make([]string, 0, 2)
	args := make([]any, 0, 2)
	if params.Q != "" {
		args = append(args, "%"+strings.ToLower(params.Q)+"%")
		placeholder := fmt.Sprintf("$%d", len(args))
		clauses = append(clauses, "(lower(name) LIKE "+placeholder+" OR lower(description) LIKE "+placeholder+")")
	}
	if params.Platform != "" {
		args = append(args, params.Platform)
		clauses = append(clauses, fmt.Sprintf("$%d = ANY(platforms::text[])", len(args)))
	}
	if len(clauses) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func reportListSQL(where string, args []any, params ReportListParams) (string, []any, error) {
	return dbutil.ListQuery{
		SelectSQL: reportSelectSQL,
		WhereSQL:  where,
		Args:      args,
		OrderKeys: map[string]dbutil.OrderExpr{
			"name":                {SQL: "name"},
			"created_at":          {SQL: "created_at"},
			dbutil.OrderUpdatedAt: {SQL: dbutil.OrderUpdatedAt},
			"schedule_interval":   {SQL: "schedule_interval"},
		},
		DefaultOrder: []dbutil.OrderExpr{{SQL: dbutil.OrderUpdatedAt}, {SQL: "id"}},
		Params:       params.ListParams,
	}.Build()
}

const reportSelectSQL = `
SELECT id, name, description, query, platforms, min_osquery_version, schedule_interval,
       created_by_user_id, created_at, updated_at
FROM reports`
