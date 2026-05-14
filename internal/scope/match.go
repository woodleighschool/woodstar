package scope

import (
	"context"
	"fmt"
)

// MatchHostScopes reports which scopes are satisfied by one host.
func (s *Store) MatchHostScopes(
	ctx context.Context,
	hostID int64,
	scopes map[int64]LabelScope,
) (map[int64]bool, error) {
	labelIDs, err := s.hostLabelIDs(ctx, hostID)
	if err != nil {
		return nil, err
	}
	matches := make(map[int64]bool, len(scopes))
	for ownerID, lscope := range scopes {
		match, err := scopeMatchesHostLabels(lscope, labelIDs)
		if err != nil {
			return nil, err
		}
		matches[ownerID] = match
	}
	return matches, nil
}

func (s *Store) hostLabelIDs(ctx context.Context, hostID int64) (map[int64]struct{}, error) {
	rows, err := s.db.Pool().Query(ctx,
		`SELECT label_id
		 FROM label_membership
		 WHERE host_id = $1`,
		hostID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	labelIDs := make(map[int64]struct{})
	for rows.Next() {
		var labelID int64
		if err := rows.Scan(&labelID); err != nil {
			return nil, err
		}
		labelIDs[labelID] = struct{}{}
	}
	return labelIDs, rows.Err()
}

func scopeMatchesHostLabels(s LabelScope, hostLabelIDs map[int64]struct{}) (bool, error) {
	s = NormalizeLabelScope(s)
	if s.Mode == ScopeNone {
		return true, nil
	}

	var count int
	for _, labelID := range s.LabelIDs {
		if _, ok := hostLabelIDs[labelID]; ok {
			count++
		}
	}

	switch s.Mode {
	case ScopeIncludeAny:
		return count > 0, nil
	case ScopeIncludeAll:
		return count == len(s.LabelIDs), nil
	case ScopeExcludeAny:
		return count == 0, nil
	default:
		return false, fmt.Errorf("scope: unknown label scope mode: %q", s.Mode)
	}
}
