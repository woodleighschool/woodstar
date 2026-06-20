package checks

import (
	"context"
	"maps"
	"slices"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/database/dbtest/crudtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func TestCheckStoreConformance(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := NewStore(db)

	crudtest.RunConformance(
		t,
		ctx,
		crudtest.Fixtures[Check, CheckCreateMutation, CheckMutation, CheckListParams]{
			Store: store,
			NewValid: func(_ *testing.T, _ context.Context) CheckCreateMutation {
				return CheckCreateMutation{
					CheckMutation: CheckMutation{
						Name:  "ConformanceCheck",
						Query: "SELECT 1",
					},
				}
			},
			Mutate: func(_ Check) CheckMutation {
				return CheckMutation{
					Name:  "ConformanceCheckUpdated",
					Query: "SELECT 2",
				}
			},
			ID:         func(c Check) int64 { return c.ID },
			ListParams: checkListParams,
			SortKeys:   slices.Sorted(maps.Keys(checkOrderKeys())),
			SearchMatch: func(c Check) string {
				return c.Name
			},
			NewInvalid: func() (CheckCreateMutation, bool) {
				return CheckCreateMutation{
					CheckMutation: CheckMutation{
						Name:  "",
						Query: "SELECT 1",
					},
				}, true
			},
		},
	)
}

func checkListParams(q, sort string, pageIndex, pageSize int32) CheckListParams {
	return CheckListParams{
		ListParams: dbutil.ListParams{
			Q:         q,
			Sort:      sort,
			PageIndex: pageIndex,
			PageSize:  pageSize,
		},
	}
}
