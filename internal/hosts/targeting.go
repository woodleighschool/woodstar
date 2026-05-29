package hosts

import (
	"context"
	"time"

	"github.com/woodleighschool/woodstar/internal/database/sqlc"
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
	Total   int
	Online  int
	Offline int
}

// ResolveSelectedTargets returns active host ids for a live target selection.
func (s *Store) ResolveSelectedTargets(ctx context.Context, selection TargetSelection) ([]int64, error) {
	directHostIDs, err := activeSelectedHostIDs(ctx, s.q, selection.HostIDs)
	if err != nil {
		return nil, err
	}
	if len(selection.LabelIDs) == 0 {
		return directHostIDs, nil
	}
	matches, err := resolveSelectedLabelTargets(ctx, s.q, selection.LabelIDs)
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
	return s.q.ListOnlineSelectedHostIDs(ctx, sqlc.ListOnlineSelectedHostIDsParams{
		HostIds:     hostIDs,
		OnlineSince: &onlineSince,
	})
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
	counts, err := s.q.CountSelectedHostStatus(ctx, sqlc.CountSelectedHostStatusParams{
		HostIds:     hostIDs,
		OnlineSince: &onlineSince,
	})
	if err != nil {
		return TargetMetrics{}, err
	}
	metrics.Total = int(counts.Total)
	metrics.Online = int(counts.Online)
	metrics.Offline = int(counts.Offline)
	return metrics, nil
}

func activeSelectedHostIDs(ctx context.Context, q *sqlc.Queries, hostIDs []int64) ([]int64, error) {
	if len(hostIDs) == 0 {
		return nil, nil
	}
	return q.ListSelectedHostIDs(ctx, sqlc.ListSelectedHostIDsParams{HostIds: hostIDs})
}

func resolveSelectedLabelTargets(ctx context.Context, q *sqlc.Queries, labelIDs []int64) ([]int64, error) {
	rows, err := q.ListSelectedLabels(ctx, sqlc.ListSelectedLabelsParams{LabelIds: labelIDs})
	if err != nil {
		return nil, err
	}

	builtinIDs := make([]int64, 0, len(labelIDs))
	regularIDs := make([]int64, 0, len(labelIDs))
	for _, row := range rows {
		if labels.LabelType(row.LabelType) == labels.LabelTypeBuiltin {
			if row.Name == "All Hosts" {
				return allActiveHostIDs(ctx, q)
			}
			builtinIDs = append(builtinIDs, row.ID)
			continue
		}
		regularIDs = append(regularIDs, row.ID)
	}

	switch {
	case len(builtinIDs) > 0 && len(regularIDs) > 0:
		return hostsMatchingBuiltinAndRegularLabels(ctx, q, builtinIDs, regularIDs)
	case len(builtinIDs) > 0:
		return hostsMatchingAnyLabel(ctx, q, builtinIDs)
	case len(regularIDs) > 0:
		return hostsMatchingAnyLabel(ctx, q, regularIDs)
	default:
		return nil, nil
	}
}

func allActiveHostIDs(ctx context.Context, q *sqlc.Queries) ([]int64, error) {
	return q.ListAllHostIDs(ctx)
}

func hostsMatchingAnyLabel(ctx context.Context, q *sqlc.Queries, labelIDs []int64) ([]int64, error) {
	return q.ListHostIDsByAnyLabel(ctx, sqlc.ListHostIDsByAnyLabelParams{LabelIds: labelIDs})
}

func hostsMatchingBuiltinAndRegularLabels(
	ctx context.Context,
	q *sqlc.Queries,
	builtinIDs []int64,
	regularIDs []int64,
) ([]int64, error) {
	return q.ListHostIDsByBuiltinAndRegularLabels(ctx, sqlc.ListHostIDsByBuiltinAndRegularLabelsParams{
		BuiltinLabelIds: builtinIDs,
		RegularLabelIds: regularIDs,
	})
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
