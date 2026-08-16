package reports

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/listing"
	"github.com/woodleighschool/woodstar/internal/postgres"
)

// Store persists saved reports and their per-host result snapshots.
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Create(ctx context.Context, in ReportCreateMutation) (*Report, error) {
	in.normalize()
	if err := in.Validate(); err != nil {
		return nil, err
	}
	write := newReportWrite(in.ReportMutation)
	write.CreatedByUserID = in.CreatedByUserID
	var id int64
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
			INSERT INTO osquery_reports (
				name,
				description,
				query,
				min_osquery_version,
				schedule_interval,
				created_by_user_id
			) VALUES (
				@name,
				@description,
				@query,
				@min_osquery_version,
				@schedule_interval,
				@created_by_user_id
			)
			RETURNING id`, pgx.StructArgs(write)).Scan(&id); err != nil {
			return postgres.MutationError(err)
		}
		return replaceReportTargets(ctx, tx, id, in.Targets)
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *Store) Update(ctx context.Context, id int64, params ReportMutation) (*Report, error) {
	params.normalize()
	if err := params.Validate(); err != nil {
		return nil, err
	}
	write := newReportWrite(params)
	write.ID = id
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var resultsInvalidated bool
		if err := tx.QueryRow(ctx, `
			WITH current AS (
				SELECT id, query, min_osquery_version
				FROM osquery_reports
				WHERE id = @id
				FOR UPDATE
			)
			UPDATE osquery_reports r
			SET
				name = @name,
				description = @description,
				query = @query,
				min_osquery_version = @min_osquery_version,
				schedule_interval = @schedule_interval,
				updated_at = now()
			FROM current
			WHERE r.id = current.id
			RETURNING
				current.query IS DISTINCT FROM @query
				OR current.min_osquery_version IS DISTINCT FROM @min_osquery_version`,
			pgx.StructArgs(write),
		).Scan(&resultsInvalidated); err != nil {
			return postgres.MutationError(err)
		}
		if err := replaceReportTargets(ctx, tx, id, params.Targets); err != nil {
			return err
		}
		// Query and minimum-version edits invalidate every snapshot.
		// Retargeting only removes hosts outside the completed assignment set.
		_, err := tx.Exec(ctx, `
				DELETE FROM osquery_report_snapshots snapshot
				WHERE snapshot.report_id = $1
				  AND (
					  $2
					  OR NOT EXISTS (
						  SELECT 1
						  FROM osquery_report_assignments assignment
						  WHERE assignment.report_id = snapshot.report_id
						    AND assignment.host_id = snapshot.host_id
					  )
				  )`,
			id,
			resultsInvalidated,
		)
		return err
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Report, error) {
	if id <= 0 {
		return nil, fault.ErrNotFound
	}
	row, err := postgres.GetOne[reportSummaryRow](ctx, s.pool, reportSummarySelectSQL()+"\nWHERE r.id = $1", id)
	if err != nil {
		return nil, err
	}
	report := reportFromSummaryRow(row)
	reports := []Report{report}
	if err := s.attachReportTargets(ctx, reports, []int64{report.ID}); err != nil {
		return nil, err
	}
	return &reports[0], nil
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM osquery_reports WHERE id = $1`, id)
	if err != nil {
		return postgres.DeleteConflict(err, "Report is still referenced")
	}
	if tag.RowsAffected() == 0 {
		return fault.ErrNotFound
	}
	return nil
}

// DeleteMany removes multiple saved reports. Missing IDs are ignored for bulk idempotency.
func (s *Store) DeleteMany(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	rows, err := s.pool.Query(
		ctx,
		`DELETE FROM osquery_reports WHERE id = ANY($1::bigint[]) RETURNING id`,
		ids,
	)
	if err != nil {
		return 0, err
	}
	deletedIDs, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return 0, err
	}
	return len(deletedIDs), nil
}

func (s *Store) List(ctx context.Context, params ReportListParams) ([]Report, int, error) {
	params.ListParams = listing.Normalize(params.ListParams)
	where, args := reportListWhere(params)
	listQuery := postgres.ListQuery{
		SelectSQL:    reportSummarySelectSQL(),
		WhereSQL:     where,
		Args:         args,
		OrderKeys:    reportOrderKeys(),
		DefaultOrder: []postgres.OrderExpr{{SQL: "r.updated_at"}, {SQL: "r.id"}},
		Params:       params.ListParams,
	}
	rows, count, err := postgres.ListWithCount[reportSummaryRow](ctx, s.pool, listQuery)
	if err != nil {
		return nil, 0, err
	}
	rpts := make([]Report, len(rows))
	rptIDs := make([]int64, len(rows))
	for i, row := range rows {
		rpts[i] = reportFromSummaryRow(row)
		rptIDs[i] = row.ID
	}
	if err := s.attachReportTargets(ctx, rpts, rptIDs); err != nil {
		return nil, 0, err
	}
	return rpts, count, nil
}

// ScheduledForHost returns reports that are scheduled and match the host's label membership.
func (s *Store) ScheduledForHost(ctx context.Context, host *hosts.Host) ([]Report, error) {
	qrows, err := s.pool.Query(ctx, reportSelectSQL()+`
		JOIN osquery_report_assignments assignment
			ON assignment.report_id = r.id
		   AND assignment.host_id = $1
		WHERE r.schedule_interval > 0
		ORDER BY r.id`, host.ID)
	if err != nil {
		return nil, err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[reportRow])
	if err != nil {
		return nil, err
	}
	rpts := make([]Report, len(rows))
	for i, row := range rows {
		rpts[i] = reportFromRow(row)
	}
	return rpts, nil
}

func reportListWhere(params ReportListParams) (string, []any) {
	var where postgres.WhereBuilder
	if params.ListParams.Q != "" {
		search := where.Arg("%" + params.ListParams.Q + "%")
		where.Add(`(r.name ILIKE ` + search + ` OR r.description ILIKE ` + search + `)`)
	}
	return where.Build()
}

func reportOrderKeys() map[string]postgres.OrderExpr {
	return map[string]postgres.OrderExpr{
		"name":                 {SQL: "lower(r.name)"},
		"collected_host_count": {SQL: "result_counts.collected_host_count"},
		"error_host_count":     {SQL: "result_counts.error_host_count"},
		"pending_host_count":   {SQL: "result_counts.pending_host_count"},
		"created_at":           {SQL: "r.created_at"},
		"updated_at":           {SQL: "r.updated_at"},
		"schedule_interval":    {SQL: "r.schedule_interval"},
	}
}

type reportRow struct {
	ID                int64     `db:"id"`
	Name              string    `db:"name"`
	Description       string    `db:"description"`
	Query             string    `db:"query"`
	MinOsqueryVersion *string   `db:"min_osquery_version"`
	ScheduleInterval  int32     `db:"schedule_interval"`
	CreatedByUserID   *int64    `db:"created_by_user_id"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}

type reportSummaryRow struct {
	reportRow

	CollectedHostCount int32 `db:"collected_host_count"`
	ErrorHostCount     int32 `db:"error_host_count"`
	PendingHostCount   int32 `db:"pending_host_count"`
}

func reportFromRow(row reportRow) Report {
	return Report{
		ID:                row.ID,
		Name:              row.Name,
		Description:       row.Description,
		Query:             row.Query,
		MinOsqueryVersion: row.MinOsqueryVersion,
		ScheduleInterval:  row.ScheduleInterval,
		Targets:           emptyReportTargets(),
		CreatedByUserID:   row.CreatedByUserID,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func reportFromSummaryRow(row reportSummaryRow) Report {
	report := reportFromRow(row.reportRow)
	report.CollectedHostCount = row.CollectedHostCount
	report.ErrorHostCount = row.ErrorHostCount
	report.PendingHostCount = row.PendingHostCount
	return report
}

type reportWrite struct {
	ID                int64   `db:"id"`
	Name              string  `db:"name"`
	Description       string  `db:"description"`
	Query             string  `db:"query"`
	MinOsqueryVersion *string `db:"min_osquery_version"`
	ScheduleInterval  int32   `db:"schedule_interval"`
	CreatedByUserID   *int64  `db:"created_by_user_id"`
}

func newReportWrite(p ReportMutation) reportWrite {
	return reportWrite{
		Name:              p.Name,
		Description:       p.Description,
		Query:             p.Query,
		MinOsqueryVersion: p.MinOsqueryVersion,
		ScheduleInterval:  p.ScheduleInterval,
	}
}

func reportSelectSQL() string {
	return `
SELECT
	r.id,
	r.name,
	r.description,
	r.query,
	r.min_osquery_version,
	r.schedule_interval,
	r.created_by_user_id,
	r.created_at,
	r.updated_at
FROM osquery_reports r`
}

func reportSummarySelectSQL() string {
	return `
SELECT
	r.id,
	r.name,
	r.description,
	r.query,
	r.min_osquery_version,
	r.schedule_interval,
	r.created_by_user_id,
	r.created_at,
	r.updated_at,
	result_counts.collected_host_count,
	result_counts.error_host_count,
	result_counts.pending_host_count
FROM osquery_reports r
LEFT JOIN LATERAL (
	SELECT
		COUNT(*) FILTER (WHERE snapshot.status = 'collected')::integer AS collected_host_count,
		COUNT(*) FILTER (WHERE snapshot.status = 'error')::integer AS error_host_count,
		COUNT(*) FILTER (WHERE snapshot.report_id IS NULL)::integer AS pending_host_count
	FROM osquery_report_assignments assignment
	LEFT JOIN osquery_report_snapshots snapshot
		ON snapshot.report_id = assignment.report_id
	   AND snapshot.host_id = assignment.host_id
	WHERE assignment.report_id = r.id
) result_counts ON true`
}
