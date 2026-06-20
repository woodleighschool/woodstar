package labels

import (
	"context"
	"maps"
	"slices"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/database/dbtest/crudtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func TestLabelsConformance(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := NewStore(db)

	crudtest.RunConformance(
		t,
		ctx,
		crudtest.Fixtures[Label, LabelMutation, LabelMutation, LabelListParams]{
			Store: store,
			NewValid: func(_ *testing.T, _ context.Context) LabelMutation {
				return LabelMutation{
					Name:                "Conformance Label",
					LabelMembershipType: LabelMembershipTypeManual,
				}
			},
			Mutate: func(_ Label) LabelMutation {
				return LabelMutation{
					Name:                "Conformance Label Updated",
					LabelMembershipType: LabelMembershipTypeManual,
				}
			},
			ID:         func(l Label) int64 { return l.ID },
			ListParams: labelListParamsFromKnobs,
			SortKeys:   slices.Sorted(maps.Keys(labelOrderKeys())),
			SearchMatch: func(l Label) string {
				return l.Name
			},
			NewInvalid: func() (LabelMutation, bool) {
				return LabelMutation{
					Name:                "",
					LabelMembershipType: LabelMembershipTypeManual,
				}, true
			},
		},
	)
}

func labelListParamsFromKnobs(q, sort string, pageIndex, pageSize int32) LabelListParams {
	return LabelListParams{
		ListParams: dbutil.ListParams{
			Q:         q,
			Sort:      sort,
			PageIndex: pageIndex,
			PageSize:  pageSize,
		},
	}
}
