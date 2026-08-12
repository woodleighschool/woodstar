package policies

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/postgres"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

// PolicyTargets is the include/exclude label targeting contract for a policy.
type PolicyTargets struct {
	Include []targeting.LabelRef `json:"include" nullable:"false"`
	Exclude []targeting.LabelRef `json:"exclude" nullable:"false"`
}

func (s *Store) loadPolicyTarget(ctx context.Context, policyID int64) (PolicyTargets, error) {
	targets, err := s.loadPolicyTargets(ctx, []int64{policyID})
	if err != nil {
		return PolicyTargets{}, err
	}
	if rows, ok := targets[policyID]; ok {
		return rows, nil
	}
	return emptyPolicyTargets(), nil
}

func (s *Store) loadPolicyTargets(
	ctx context.Context,
	policyIDs []int64,
) (map[int64]PolicyTargets, error) {
	if len(policyIDs) == 0 {
		return map[int64]PolicyTargets{}, nil
	}
	qrows, err := s.pool.Query(ctx, listPolicyTargetsSQL, policyIDs)
	if err != nil {
		return nil, err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[policyTargetRow])
	if err != nil {
		return nil, err
	}
	targetSets, err := policyTargetSets(policyIDs, rows)
	if err != nil {
		return nil, err
	}
	targets := make(map[int64]PolicyTargets, len(targetSets))
	for policyID, targetSet := range targetSets {
		targets[policyID] = PolicyTargets(targetSet)
	}
	return targets, nil
}

const insertPolicyTargetSQL = `
INSERT INTO osquery_policy_targets (policy_id, label_id, direction, position)
VALUES (@owner_id, @label_id, @direction::target_direction, @position)`

func replacePolicyTargets(ctx context.Context, tx pgx.Tx, policyID int64, targets PolicyTargets) error {
	targets = normalizePolicyTargets(targets)
	if err := targets.validate(); err != nil {
		return err
	}
	rows := policyTargetWrites(policyID, targeting.LabelSet(targets))
	if err := postgres.ReplaceChildren(
		ctx, tx,
		deletePolicyTargetsSQL, []any{policyID},
		insertPolicyTargetSQL, rows,
	); err != nil {
		return postgres.MutationError(err)
	}
	return nil
}

func (targets PolicyTargets) validate() error {
	if err := targeting.ValidateLabelSets(targets.Include, targets.Exclude); err != nil {
		return fmt.Errorf("%w: %w", fault.ErrInvalidInput, err)
	}
	return nil
}

func normalizePolicyTargets(targets PolicyTargets) PolicyTargets {
	return PolicyTargets(targeting.NormalizeLabelSet(targeting.LabelSet(targets)))
}

func emptyPolicyTargets() PolicyTargets {
	return PolicyTargets(targeting.EmptyLabelSet())
}

type policyTargetRow struct {
	OwnerID   int64  `db:"owner_id"`
	LabelID   int64  `db:"label_id"`
	Direction string `db:"direction"`
}

type policyTargetWrite struct {
	OwnerID   int64  `db:"owner_id"`
	LabelID   int64  `db:"label_id"`
	Direction string `db:"direction"`
	Position  int32  `db:"position"`
}

func policyTargetSets(ownerIDs []int64, rows []policyTargetRow) (map[int64]targeting.LabelSet, error) {
	targets := make(map[int64]targeting.LabelSet, len(ownerIDs))
	for _, ownerID := range ownerIDs {
		targets[ownerID] = targeting.EmptyLabelSet()
	}
	for _, row := range rows {
		targetSet := targets[row.OwnerID]
		ref := targeting.LabelRef{LabelID: row.LabelID}
		switch targeting.Direction(row.Direction) {
		case targeting.Include:
			targetSet.Include = append(targetSet.Include, ref)
		case targeting.Exclude:
			targetSet.Exclude = append(targetSet.Exclude, ref)
		default:
			return nil, fmt.Errorf("policies: unsupported target direction %q", row.Direction)
		}
		targets[row.OwnerID] = targetSet
	}
	return targets, nil
}

func policyTargetWrites(ownerID int64, targets targeting.LabelSet) []policyTargetWrite {
	targets = targeting.NormalizeLabelSet(targets)
	rows := make([]policyTargetWrite, 0, len(targets.Include)+len(targets.Exclude))
	for i, ref := range targets.Include {
		rows = append(rows, policyTargetWrite{
			OwnerID:   ownerID,
			LabelID:   ref.LabelID,
			Direction: string(targeting.Include),
			Position:  int32(i),
		})
	}
	for i, ref := range targets.Exclude {
		rows = append(rows, policyTargetWrite{
			OwnerID:   ownerID,
			LabelID:   ref.LabelID,
			Direction: string(targeting.Exclude),
			Position:  int32(i),
		})
	}
	return rows
}

const listPolicyTargetsSQL = `
SELECT policy_id AS owner_id, label_id, direction::text AS direction
FROM osquery_policy_targets
WHERE policy_id = ANY($1::bigint[])
ORDER BY
    policy_id,
    direction,
    position`

const deletePolicyTargetsSQL = `DELETE FROM osquery_policy_targets WHERE policy_id = $1`
