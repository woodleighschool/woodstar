package activity

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woodleighschool/woodstar/internal/postgres"
)

// Store persists and lists activity events.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore returns an activity store backed by pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Record appends one activity event.
func (s *Store) Record(ctx context.Context, event NewEvent) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO activity_events (
			area,
			action,
			actor_kind,
			actor_user_id,
			actor_name,
			actor_email,
			subject_type,
			subject_id,
			subject_name
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		event.Area,
		event.Action,
		event.Actor.Kind,
		event.Actor.UserID,
		event.Actor.Name,
		event.Actor.Email,
		event.Subject.Type,
		event.Subject.ID,
		event.Subject.Name,
	)
	return err
}

// List returns newest-first activities and the total matching count.
func (s *Store) List(ctx context.Context, params ListParams) ([]ActivityEvent, int, error) {
	params.normalize()
	if err := params.validate(); err != nil {
		return nil, 0, err
	}
	var where postgres.WhereBuilder
	if params.Area != "" {
		where.Add("area = " + where.Arg(params.Area))
	}
	if params.ActorKind != "" {
		where.Add("actor_kind = " + where.Arg(params.ActorKind))
	}
	if params.Action != "" {
		where.Add("action = " + where.Arg(params.Action))
	}
	if !params.Since.IsZero() {
		where.Add("occurred_at >= " + where.Arg(params.Since))
	}
	if !params.Before.IsZero() {
		where.Add("occurred_at < " + where.Arg(params.Before))
	}
	if params.SubjectType != "" {
		where.Add("subject_type = " + where.Arg(params.SubjectType))
		where.Add("subject_id = " + where.Arg(params.SubjectID))
	}
	if params.ListParams.Q != "" {
		search := where.Arg("%" + params.ListParams.Q + "%")
		where.Add(`(
			action ILIKE ` + search + `
			OR actor_name ILIKE ` + search + `
			OR actor_email ILIKE ` + search + `
			OR subject_name ILIKE ` + search + `
		)`)
	}
	whereSQL, args := where.Build()
	rows, count, err := postgres.ListWithCount[eventRow](ctx, s.pool, postgres.ListQuery{
		SelectSQL: `
			SELECT
				id,
				area,
				action,
				actor_kind,
				actor_user_id,
				actor_name,
				actor_email,
				subject_type,
				subject_id,
				subject_name,
				occurred_at
			FROM activity_events`,
		WhereSQL: whereSQL,
		Args:     args,
		OrderKeys: map[string]postgres.OrderExpr{
			"occurred_at": {SQL: "occurred_at"},
		},
		DefaultOrder: []postgres.OrderExpr{
			{SQL: "occurred_at", Descending: true},
			{SQL: "id", Descending: true},
		},
		Params: params.ListParams,
	})
	if err != nil {
		return nil, 0, err
	}
	events := make([]ActivityEvent, len(rows))
	for i, row := range rows {
		events[i] = row.event()
	}
	return events, count, nil
}

// SweepBefore removes activities older than cutoff.
func (s *Store) SweepBefore(ctx context.Context, cutoff time.Time) (int, error) {
	command, err := s.pool.Exec(ctx, `DELETE FROM activity_events WHERE occurred_at < $1`, cutoff)
	if err != nil {
		return 0, err
	}
	return int(command.RowsAffected()), nil
}

type eventRow struct {
	ID          int64     `db:"id"`
	Area        Area      `db:"area"`
	Action      Action    `db:"action"`
	ActorKind   ActorKind `db:"actor_kind"`
	ActorUserID *int64    `db:"actor_user_id"`
	ActorName   string    `db:"actor_name"`
	ActorEmail  string    `db:"actor_email"`
	SubjectType string    `db:"subject_type"`
	SubjectID   *int64    `db:"subject_id"`
	SubjectName string    `db:"subject_name"`
	OccurredAt  time.Time `db:"occurred_at"`
}

func (row eventRow) event() ActivityEvent {
	return ActivityEvent{
		ID:     row.ID,
		Area:   row.Area,
		Action: row.Action,
		Actor: Actor{
			Kind:   row.ActorKind,
			UserID: row.ActorUserID,
			Name:   row.ActorName,
			Email:  row.ActorEmail,
		},
		Subject: Subject{
			Type: row.SubjectType,
			ID:   row.SubjectID,
			Name: row.SubjectName,
		},
		OccurredAt: row.OccurredAt,
	}
}
