package reports

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

// Store persists saved reports and their per-host result snapshots.
type Store struct {
	db *database.DB
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Create(ctx context.Context, in ReportCreateMutation) (*Report, error) {
	if err := in.ReportMutation.Validate(); err != nil {
		return nil, err
	}
	write := newReportWrite(in.ReportMutation)
	write.CreatedByUserID = in.CreatedByUserID
	var id int64
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, insertReportSQL, pgx.StructArgs(write)).Scan(&id); err != nil {
			return dbutil.MutationError(err)
		}
		return replaceReportTargets(ctx, tx, id, in.Targets)
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *Store) Update(ctx context.Context, id int64, params ReportMutation) (*Report, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	write := newReportWrite(params)
	write.ID = id
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		var updatedID int64
		if err := tx.QueryRow(ctx, updateReportSQL, pgx.StructArgs(write)).Scan(&updatedID); err != nil {
			return dbutil.MutationError(err)
		}
		return replaceReportTargets(ctx, tx, id, params.Targets)
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Report, error) {
	row, err := dbutil.GetOne[reportRow](ctx, s.db.Pool(), reportSelectSQL+"\nWHERE r.id = $1", id)
	if err != nil {
		return nil, err
	}
	report := reportFromRow(row)
	reports := []Report{report}
	if err := s.attachReportTargets(ctx, reports, []int64{report.ID}); err != nil {
		return nil, err
	}
	return &reports[0], nil
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	tag, err := s.db.Pool().Exec(ctx, `DELETE FROM reports WHERE id = $1`, id)
	if err != nil {
		return dbutil.DeleteConflict(err, "Report is still referenced")
	}
	if tag.RowsAffected() == 0 {
		return dbutil.ErrNotFound
	}
	return nil
}

// DeleteMany removes multiple saved reports. Missing IDs are ignored for bulk idempotency.
func (s *Store) DeleteMany(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	rows, err := s.db.Pool().Query(
		ctx,
		`DELETE FROM reports WHERE id = ANY($1::bigint[]) RETURNING id`,
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
	where, args := reportListWhere(params)
	listQuery := dbutil.ListQuery{
		SelectSQL:    reportSelectSQL,
		WhereSQL:     where,
		Args:         args,
		OrderKeys:    reportOrderKeys(),
		DefaultOrder: []dbutil.OrderExpr{{SQL: "r.updated_at"}, {SQL: "r.id"}},
		Params:       params.ListParams,
	}
	rows, count, err := dbutil.ListWithCount[reportRow](ctx, s.db.Pool(), listQuery)
	if err != nil {
		return nil, 0, err
	}
	rpts := make([]Report, len(rows))
	rptIDs := make([]int64, len(rows))
	for i, row := range rows {
		rpts[i] = reportFromRow(row)
		rptIDs[i] = row.ID
	}
	if err := s.attachReportTargets(ctx, rpts, rptIDs); err != nil {
		return nil, 0, err
	}
	return rpts, count, nil
}

// ScheduledForHost returns reports that are scheduled and match the host's label membership.
func (s *Store) ScheduledForHost(ctx context.Context, host *hosts.Host) ([]Report, error) {
	qrows, err := s.db.Pool().Query(ctx, scheduledReportsForHostSQL, host.ID)
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
	var where dbutil.WhereBuilder
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add(`(r.name ILIKE ` + search + ` OR r.description ILIKE ` + search + `)`)
	}
	return where.Build()
}

func reportOrderKeys() map[string]dbutil.OrderExpr {
	return map[string]dbutil.OrderExpr{
		"name":              {SQL: "lower(r.name)"},
		"created_at":        {SQL: "r.created_at"},
		"updated_at":        {SQL: "r.updated_at"},
		"schedule_interval": {SQL: "r.schedule_interval"},
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

const reportSelectSQL = `
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
FROM reports r`

const insertReportSQL = `
INSERT INTO reports (
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
RETURNING id`

const updateReportSQL = `
UPDATE reports
SET
	name = @name,
	description = @description,
	query = @query,
	min_osquery_version = @min_osquery_version,
	schedule_interval = @schedule_interval,
	updated_at = now()
WHERE id = @id
RETURNING id`

const scheduledReportsForHostSQL = `
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
FROM reports r
WHERE r.schedule_interval > 0
  AND EXISTS (
      SELECT 1
      FROM osquery_report_targets rt
      JOIN label_membership lm ON lm.label_id = rt.label_id AND lm.host_id = $1
      WHERE rt.report_id = r.id
        AND rt.direction = 'include'
  )
  AND NOT EXISTS (
      SELECT 1
      FROM osquery_report_targets rt
      JOIN label_membership lm ON lm.label_id = rt.label_id AND lm.host_id = $1
      WHERE rt.report_id = r.id
        AND rt.direction = 'exclude'
  )
ORDER BY r.id`
