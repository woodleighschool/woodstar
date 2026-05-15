package hosts

import (
	"context"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/labels"
)

const (
	hostOnlineWindow          = 5 * time.Minute
	hostMissingInActionWindow = 30 * 24 * time.Hour
)

// TargetSelection is the live targeting shape.
type TargetSelection struct {
	HostIDs  []int64
	LabelIDs []int64
}

// TargetMetrics counts the current status split for a resolved target set.
type TargetMetrics struct {
	Total           int
	Online          int
	Offline         int
	MissingInAction int
}

// ResolveSelectedTargets returns active host ids for a live target selection.
func (s *Store) ResolveSelectedTargets(ctx context.Context, selection TargetSelection) ([]int64, error) {
	hostIDs := cleanPositiveIDs(selection.HostIDs)
	labelIDs := cleanPositiveIDs(selection.LabelIDs)
	directHostIDs, err := activeSelectedHostIDs(ctx, s.db, hostIDs)
	if err != nil {
		return nil, err
	}
	if len(labelIDs) == 0 {
		return directHostIDs, nil
	}
	matches, err := resolveSelectedLabelTargets(ctx, s.db, labelIDs)
	if err != nil {
		return nil, err
	}
	return mergePositiveIDs(directHostIDs, matches), nil
}

// CountSelectedTargets returns Fleet-style target status totals for a selection.
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
	err = s.db.Pool().QueryRow(ctx,
		`SELECT
		     count(*)::integer AS total,
		     count(*) FILTER (WHERE last_seen_at >= $2)::integer AS online,
		     count(*) FILTER (WHERE last_seen_at IS NULL OR last_seen_at < $2)::integer AS offline,
		     count(*) FILTER (WHERE COALESCE(last_seen_at, created_at) <= $3)::integer AS missing_in_action
		 FROM hosts
		 WHERE deleted_at IS NULL AND id = ANY($1::bigint[])`,
		hostIDs,
		now.Add(-hostOnlineWindow),
		now.Add(-hostMissingInActionWindow),
	).Scan(&metrics.Total, &metrics.Online, &metrics.Offline, &metrics.MissingInAction)
	if err != nil {
		return TargetMetrics{}, err
	}
	return metrics, nil
}

func activeSelectedHostIDs(ctx context.Context, db *database.DB, hostIDs []int64) ([]int64, error) {
	if len(hostIDs) == 0 {
		return nil, nil
	}
	rows, err := db.Pool().Query(ctx,
		`SELECT id
		 FROM hosts
		 WHERE deleted_at IS NULL AND id = ANY($1::bigint[])
		 ORDER BY id`,
		hostIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHostIDs(rows)
}

func resolveSelectedLabelTargets(ctx context.Context, db *database.DB, labelIDs []int64) ([]int64, error) {
	rows, err := db.Pool().Query(ctx,
		`SELECT id, name, label_type
		 FROM labels
		 WHERE id = ANY($1::bigint[])
		 ORDER BY id`,
		labelIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	builtinIDs := make([]int64, 0, len(labelIDs))
	regularIDs := make([]int64, 0, len(labelIDs))
	for rows.Next() {
		var id int64
		var name string
		var labelType string
		if err := rows.Scan(&id, &name, &labelType); err != nil {
			return nil, err
		}
		if labelType == labels.LabelTypeBuiltin {
			if name == "All Hosts" {
				return allActiveHostIDs(ctx, db)
			}
			builtinIDs = append(builtinIDs, id)
			continue
		}
		regularIDs = append(regularIDs, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	switch {
	case len(builtinIDs) > 0 && len(regularIDs) > 0:
		return hostsMatchingBuiltinAndRegularLabels(ctx, db, builtinIDs, regularIDs)
	case len(builtinIDs) > 0:
		return hostsMatchingAnyLabel(ctx, db, builtinIDs)
	case len(regularIDs) > 0:
		return hostsMatchingAnyLabel(ctx, db, regularIDs)
	default:
		return nil, nil
	}
}

func allActiveHostIDs(ctx context.Context, db *database.DB) ([]int64, error) {
	rows, err := db.Pool().Query(ctx, `SELECT id FROM hosts WHERE deleted_at IS NULL ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHostIDs(rows)
}

func hostsMatchingAnyLabel(ctx context.Context, db *database.DB, labelIDs []int64) ([]int64, error) {
	rows, err := db.Pool().Query(ctx,
		`SELECT DISTINCT h.id
		 FROM hosts h
		 JOIN label_membership lm ON lm.host_id = h.id
		 WHERE h.deleted_at IS NULL AND lm.label_id = ANY($1::bigint[])
		 ORDER BY h.id`,
		labelIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHostIDs(rows)
}

func hostsMatchingBuiltinAndRegularLabels(
	ctx context.Context,
	db *database.DB,
	builtinIDs []int64,
	regularIDs []int64,
) ([]int64, error) {
	rows, err := db.Pool().Query(ctx,
		`SELECT DISTINCT h.id
		 FROM hosts h
		 WHERE h.deleted_at IS NULL
		   AND EXISTS (
		       SELECT 1 FROM label_membership lm
		       WHERE lm.host_id = h.id AND lm.label_id = ANY($1::bigint[])
		   )
		   AND EXISTS (
		       SELECT 1 FROM label_membership lm
		       WHERE lm.host_id = h.id AND lm.label_id = ANY($2::bigint[])
		   )
		 ORDER BY h.id`,
		builtinIDs,
		regularIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHostIDs(rows)
}

func scanHostIDs(rows pgx.Rows) ([]int64, error) {
	hostIDs := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		hostIDs = append(hostIDs, id)
	}
	return hostIDs, rows.Err()
}

func mergePositiveIDs(a, b []int64) []int64 {
	seen := make(map[int64]struct{}, len(a)+len(b))
	out := make([]int64, 0, len(a)+len(b))
	for _, id := range a {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	for _, id := range b {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func cleanPositiveIDs(ids []int64) []int64 {
	out := make([]int64, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}
