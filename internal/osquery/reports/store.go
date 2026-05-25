package reports

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

// Store persists saved reports and their per-host result snapshots.
type Store struct {
	db *database.DB
	q  *sqlc.Queries
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db, q: sqlc.New(db.Pool())}
}

func (s *Store) List(ctx context.Context, params ReportListParams) ([]Report, int, error) {
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
	scopes, err := s.loadReportScopes(ctx, reportIDs)
	if err != nil {
		return nil, 0, err
	}
	for i := range reports {
		reports[i].LabelScope = scopes[reports[i].ID]
	}
	return reports, count, nil
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Report, error) {
	report, err := s.getByID(ctx, id)
	if err != nil {
		return nil, err
	}
	labelScope, err := s.loadReportScope(ctx, report.ID)
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

func (s *Store) Create(ctx context.Context, params ReportCreate) (*Report, error) {
	var created *Report
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		row, err := s.q.WithTx(tx).CreateReport(ctx, sqlc.CreateReportParams{
			Name:              params.Name,
			Description:       params.Description,
			Query:             params.Query,
			Platforms:         params.Platforms,
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
		if err := replaceReportScope(ctx, tx, report.ID, params.LabelScope); err != nil {
			return err
		}
		report.LabelScope = params.LabelScope
		created = report
		return nil
	})
	return created, err
}

func (s *Store) Update(ctx context.Context, id int64, params ReportUpdate) (*Report, error) {
	var updated *Report
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		row, err := s.q.WithTx(tx).UpdateReport(ctx, sqlc.UpdateReportParams{
			Name:              params.Name,
			Description:       params.Description,
			Query:             params.Query,
			Platforms:         params.Platforms,
			MinOsqueryVersion: params.MinOsqueryVersion,
			ScheduleInterval:  int32(params.ScheduleInterval),
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
		if err := replaceReportScope(ctx, tx, report.ID, params.LabelScope); err != nil {
			return err
		}
		report.LabelScope = params.LabelScope
		updated = report
		return nil
	})
	return updated, err
}

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

func scanReport(row pgx.Row) (*Report, error) {
	var report Report
	err := row.Scan(
		&report.ID,
		&report.Name,
		&report.Description,
		&report.Query,
		&report.Platforms,
		&report.MinOsqueryVersion,
		&report.ScheduleInterval,
		&report.CreatedByUserID,
		&report.CreatedAt,
		&report.UpdatedAt,
	)
	return &report, err
}

func reportFromSQLC(row sqlc.Report) *Report {
	return &Report{
		ID:                row.ID,
		Name:              row.Name,
		Description:       row.Description,
		Query:             row.Query,
		Platforms:         row.Platforms,
		MinOsqueryVersion: row.MinOsqueryVersion,
		ScheduleInterval:  int(row.ScheduleInterval),
		CreatedByUserID:   row.CreatedByUserID,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func reportListWhere(params ReportListParams) (string, []any) {
	var where dbutil.WhereBuilder
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add("(name ILIKE " + search + " OR description ILIKE " + search + ")")
	}
	if params.Platform != "" {
		where.Add(where.Arg(params.Platform) + " = ANY(platforms::text[])")
	}
	return where.Build()
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
