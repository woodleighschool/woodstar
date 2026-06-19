package reports

import (
	"context"

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
	return &Store{db: db, q: db.Queries()}
}

func (s *Store) List(ctx context.Context, params ReportListParams) ([]Report, int, error) {
	where, args := reportListWhere(params)
	listQuery := reportListQuery(where, args, params)

	reports, count, err := dbutil.ScanListWithCount(ctx, s.db.Pool(), listQuery, func(row pgx.Row) (Report, error) {
		report, err := scanReport(row)
		if err != nil {
			return Report{}, err
		}
		return *report, nil
	})
	if err != nil {
		return nil, 0, err
	}

	reportIDs := make([]int64, 0, len(reports))
	for _, report := range reports {
		reportIDs = append(reportIDs, report.ID)
	}
	targets, err := s.loadReportTargets(ctx, reportIDs)
	if err != nil {
		return nil, 0, err
	}
	for i := range reports {
		reports[i].Targets = targets[reports[i].ID]
	}
	return reports, count, nil
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Report, error) {
	report, err := s.getByID(ctx, id)
	if err != nil {
		return nil, err
	}
	targets, err := s.loadReportTarget(ctx, report.ID)
	if err != nil {
		return nil, err
	}
	report.Targets = targets
	return report, nil
}

func (s *Store) getByID(ctx context.Context, id int64) (*Report, error) {
	row, err := s.q.GetReportByID(ctx, sqlc.GetReportByIDParams{ID: id})
	if err != nil {
		return nil, dbutil.GetError(err)
	}
	return reportFromSQLC(row), nil
}

func (s *Store) Create(ctx context.Context, params ReportMutation, createdByUserID *int64) (*Report, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	var created *Report
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		row, err := s.q.WithTx(tx).CreateReport(ctx, sqlc.CreateReportParams{
			Name:              params.Name,
			Description:       params.Description,
			Query:             params.Query,
			MinOsqueryVersion: params.MinOsqueryVersion,
			ScheduleInterval:  params.ScheduleInterval,
			CreatedByUserID:   createdByUserID,
		})
		if err != nil {
			return err
		}
		report := reportFromSQLC(row)
		if err := replaceReportTargets(ctx, tx, report.ID, params.Targets); err != nil {
			return err
		}
		report.Targets = normalizeReportTargets(params.Targets)
		created = report
		return nil
	})
	if err != nil {
		return nil, dbutil.MutationError(err)
	}
	return created, nil
}

func (s *Store) Update(ctx context.Context, id int64, params ReportMutation) (*Report, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	var updated *Report
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		row, err := s.q.WithTx(tx).UpdateReport(ctx, sqlc.UpdateReportParams{
			Name:              params.Name,
			Description:       params.Description,
			Query:             params.Query,
			MinOsqueryVersion: params.MinOsqueryVersion,
			ScheduleInterval:  params.ScheduleInterval,
			ID:                id,
		})
		if err != nil {
			return dbutil.GetError(err)
		}
		report := reportFromSQLC(row)
		if err := replaceReportTargets(ctx, tx, report.ID, params.Targets); err != nil {
			return err
		}
		report.Targets = normalizeReportTargets(params.Targets)
		updated = report
		return nil
	})
	if err != nil {
		return nil, dbutil.MutationError(err)
	}
	return updated, nil
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	_, err := s.q.DeleteReport(ctx, sqlc.DeleteReportParams{ID: id})
	if err != nil {
		return dbutil.GetError(err)
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
		MinOsqueryVersion: row.MinOsqueryVersion,
		ScheduleInterval:  row.ScheduleInterval,
		CreatedByUserID:   row.CreatedByUserID,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func reportListWhere(params ReportListParams) (string, []any) {
	var where dbutil.WhereBuilder
	if params.Q != "" {
		where.Addf("(name ILIKE %s OR description ILIKE %s)", "%"+params.Q+"%", "%"+params.Q+"%")
	}
	return where.Build()
}

func reportListQuery(where string, args []any, params ReportListParams) dbutil.ListQuery {
	return dbutil.ListQuery{
		SelectSQL: reportSelectSQL,
		WhereSQL:  where,
		Args:      args,
		OrderKeys: map[string]dbutil.OrderExpr{
			"name":              {SQL: "name"},
			"created_at":        {SQL: "created_at"},
			"updated_at":        {SQL: "updated_at"},
			"schedule_interval": {SQL: "schedule_interval"},
		},
		DefaultOrder: []dbutil.OrderExpr{{SQL: "updated_at"}, {SQL: "id"}},
		Params:       params.ListParams,
	}
}

const reportSelectSQL = `
SELECT id, name, description, query, min_osquery_version, schedule_interval,
       created_by_user_id, created_at, updated_at
FROM reports`
