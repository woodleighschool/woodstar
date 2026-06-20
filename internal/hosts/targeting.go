package hosts

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/labels"
)

const hostOnlineWindow = 5 * time.Minute

// TargetSelection is the live targeting shape.
type TargetSelection struct {
	HostIDs  []int64
	LabelIDs []int64
}

// TargetMetrics counts the current status split for a resolved target set.
type TargetMetrics struct {
	Total   int32
	Online  int32
	Offline int32
}

// ResolveSelectedTargets returns active host ids for a live target selection.
func (s *Store) ResolveSelectedTargets(ctx context.Context, selection TargetSelection) ([]int64, error) {
	directHostIDs, err := s.activeSelectedHostIDs(ctx, selection.HostIDs)
	if err != nil {
		return nil, err
	}
	if len(selection.LabelIDs) == 0 {
		return directHostIDs, nil
	}
	matches, err := s.resolveSelectedLabelTargets(ctx, selection.LabelIDs)
	if err != nil {
		return nil, err
	}
	return mergeHostIDs(directHostIDs, matches), nil
}

// ResolveOnlineSelectedTargets returns active selected host IDs that are online
// at the moment the live run starts.
func (s *Store) ResolveOnlineSelectedTargets(
	ctx context.Context,
	selection TargetSelection,
	now time.Time,
) ([]int64, error) {
	hostIDs, err := s.ResolveSelectedTargets(ctx, selection)
	if err != nil {
		return nil, err
	}
	if len(hostIDs) == 0 {
		return nil, nil
	}
	onlineSince := now.Add(-hostOnlineWindow)
	rows, err := s.db.Pool().Query(ctx, `
		SELECT id
		FROM hosts
		WHERE id = ANY($1::bigint[])
		  AND last_seen_at >= $2
		ORDER BY id`, hostIDs, &onlineSince)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[int64])
}

// CountSelectedTargets returns online/offline target status totals for a selection.
func (s *Store) CountSelectedTargets(
	ctx context.Context,
	selection TargetSelection,
	now time.Time,
) (TargetMetrics, error) {
	hostIDs, err := s.ResolveSelectedTargets(ctx, selection)
	if err != nil {
		return TargetMetrics{}, err
	}
	if len(hostIDs) == 0 {
		return TargetMetrics{}, nil
	}

	var metrics TargetMetrics
	onlineSince := now.Add(-hostOnlineWindow)
	err = s.db.Pool().QueryRow(ctx, `
		SELECT
			count(*)::integer AS total,
			count(*) FILTER (WHERE last_seen_at >= $1)::integer AS online,
			count(*) FILTER (WHERE last_seen_at IS NULL OR last_seen_at < $1)::integer AS offline
		FROM hosts
		WHERE id = ANY($2::bigint[])`, &onlineSince, hostIDs).
		Scan(&metrics.Total, &metrics.Online, &metrics.Offline)
	if err != nil {
		return TargetMetrics{}, err
	}
	return metrics, nil
}

func (s *Store) activeSelectedHostIDs(ctx context.Context, hostIDs []int64) ([]int64, error) {
	if len(hostIDs) == 0 {
		return nil, nil
	}
	rows, err := s.db.Pool().Query(ctx, `
		SELECT id
		FROM hosts
		WHERE id = ANY($1::bigint[])
		ORDER BY id`, hostIDs)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[int64])
}

type selectedLabelRow struct {
	ID         int64   `db:"id"`
	Name       string  `db:"name"`
	LabelType  string  `db:"label_type"`
	BuiltinKey *string `db:"builtin_key"`
}

func (s *Store) resolveSelectedLabelTargets(ctx context.Context, labelIDs []int64) ([]int64, error) {
	queryRows, err := s.db.Pool().Query(ctx, `
		SELECT id, name, label_type, builtin_key
		FROM labels
		WHERE id = ANY($1::bigint[])
		ORDER BY id`, labelIDs)
	if err != nil {
		return nil, err
	}
	rows, err := pgx.CollectRows(queryRows, pgx.RowToStructByName[selectedLabelRow])
	if err != nil {
		return nil, err
	}

	builtinIDs := make([]int64, 0, len(labelIDs))
	regularIDs := make([]int64, 0, len(labelIDs))
	for _, row := range rows {
		if labels.LabelType(row.LabelType) == labels.LabelTypeBuiltin {
			if row.BuiltinKey != nil && labels.BuiltinKey(*row.BuiltinKey) == labels.BuiltinKeyAllHosts {
				return s.allActiveHostIDs(ctx)
			}
			builtinIDs = append(builtinIDs, row.ID)
			continue
		}
		regularIDs = append(regularIDs, row.ID)
	}

	switch {
	case len(builtinIDs) > 0 && len(regularIDs) > 0:
		return s.hostsMatchingBuiltinAndRegularLabels(ctx, builtinIDs, regularIDs)
	case len(builtinIDs) > 0:
		return s.hostsMatchingAnyLabel(ctx, builtinIDs)
	case len(regularIDs) > 0:
		return s.hostsMatchingAnyLabel(ctx, regularIDs)
	default:
		return nil, nil
	}
}

func (s *Store) allActiveHostIDs(ctx context.Context) ([]int64, error) {
	rows, err := s.db.Pool().Query(ctx, `SELECT id FROM hosts ORDER BY id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[int64])
}

func (s *Store) hostsMatchingAnyLabel(ctx context.Context, labelIDs []int64) ([]int64, error) {
	rows, err := s.db.Pool().Query(ctx, `
		SELECT DISTINCT h.id
		FROM hosts h
		JOIN label_membership lm ON lm.host_id = h.id
		WHERE lm.label_id = ANY($1::bigint[])
		ORDER BY h.id`, labelIDs)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[int64])
}

func (s *Store) hostsMatchingBuiltinAndRegularLabels(
	ctx context.Context,
	builtinIDs []int64,
	regularIDs []int64,
) ([]int64, error) {
	rows, err := s.db.Pool().Query(ctx, `
		SELECT DISTINCT h.id
		FROM hosts h
		WHERE EXISTS (
				SELECT 1
				FROM label_membership lm
				WHERE lm.host_id = h.id AND lm.label_id = ANY($1::bigint[])
			)
		  AND EXISTS (
				SELECT 1
				FROM label_membership lm
				WHERE lm.host_id = h.id AND lm.label_id = ANY($2::bigint[])
			)
		ORDER BY h.id`, builtinIDs, regularIDs)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[int64])
}

func mergeHostIDs(a, b []int64) []int64 {
	seen := make(map[int64]struct{}, len(a)+len(b))
	out := make([]int64, 0, len(a)+len(b))
	for _, ids := range [][]int64{a, b} {
		for _, id := range ids {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}
