// Package targeting evaluates label inclusion and exclusion for hosts.
package targeting

import (
	"fmt"

	"github.com/jackc/pgx/v5"
)

// LabelSet is a plain include/exclude target set.
type LabelSet struct {
	Include []LabelRef
	Exclude []LabelRef
}

// LabelTargetRow is the shared projection for label-target child tables.
type LabelTargetRow struct {
	OwnerID   int64  `db:"owner_id"`
	LabelID   int64  `db:"label_id"`
	Direction string `db:"direction"`
}

// LabelTargetWrite is the shared write shape for label-target child tables.
type LabelTargetWrite struct {
	OwnerID   int64  `db:"owner_id"`
	LabelID   int64  `db:"label_id"`
	Direction string `db:"direction"`
	Position  int32  `db:"position"`
}

func EmptyLabelSet() LabelSet {
	return LabelSet{
		Include: []LabelRef{},
		Exclude: []LabelRef{},
	}
}

func NormalizeLabelSet(targets LabelSet) LabelSet {
	if targets.Include == nil {
		targets.Include = []LabelRef{}
	}
	if targets.Exclude == nil {
		targets.Exclude = []LabelRef{}
	}
	return targets
}

func LabelSetsFromRows(ownerIDs []int64, rows []LabelTargetRow) (map[int64]LabelSet, error) {
	targets := make(map[int64]LabelSet, len(ownerIDs))
	for _, ownerID := range ownerIDs {
		targets[ownerID] = EmptyLabelSet()
	}
	for _, row := range rows {
		targetSet := targets[row.OwnerID]
		ref := LabelRef{LabelID: row.LabelID}
		switch Direction(row.Direction) {
		case Include:
			targetSet.Include = append(targetSet.Include, ref)
		case Exclude:
			targetSet.Exclude = append(targetSet.Exclude, ref)
		default:
			return nil, fmt.Errorf("targeting: unsupported target direction %q", row.Direction)
		}
		targets[row.OwnerID] = targetSet
	}
	return targets, nil
}

func CollectLabelTargetRows(rows pgx.Rows) ([]LabelTargetRow, error) {
	return pgx.CollectRows(rows, pgx.RowToStructByName[LabelTargetRow])
}

func LabelTargetWrites(ownerID int64, targets LabelSet) []LabelTargetWrite {
	targets = NormalizeLabelSet(targets)
	rows := make([]LabelTargetWrite, 0, len(targets.Include)+len(targets.Exclude))
	for i, ref := range targets.Include {
		rows = append(rows, LabelTargetWrite{
			OwnerID:   ownerID,
			LabelID:   ref.LabelID,
			Direction: string(Include),
			Position:  int32(i),
		})
	}
	for i, ref := range targets.Exclude {
		rows = append(rows, LabelTargetWrite{
			OwnerID:   ownerID,
			LabelID:   ref.LabelID,
			Direction: string(Exclude),
			Position:  int32(i),
		})
	}
	return rows
}
