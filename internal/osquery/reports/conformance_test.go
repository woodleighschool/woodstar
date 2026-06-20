package reports

import (
	"context"
	"maps"
	"slices"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/database/dbtest/crudtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func TestReportStoreConformance(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := NewStore(db)

	crudtest.RunConformance(
		t,
		ctx,
		crudtest.Fixtures[Report, ReportCreateMutation, ReportMutation, ReportListParams]{
			Store: store,
			NewValid: func(_ *testing.T, _ context.Context) ReportCreateMutation {
				return ReportCreateMutation{
					ReportMutation: ReportMutation{
						Name:             "Conformance Report",
						Query:            "SELECT 1;",
						ScheduleInterval: 60,
					},
				}
			},
			Mutate: func(_ Report) ReportMutation {
				return ReportMutation{
					Name:             "Conformance Report Updated",
					Query:            "SELECT 2;",
					ScheduleInterval: 120,
				}
			},
			ID: func(r Report) int64 { return r.ID },
			ListParams: func(q, sort string, pageIndex, pageSize int32) ReportListParams {
				return ReportListParams{
					ListParams: dbutil.ListParams{
						Q:         q,
						Sort:      sort,
						PageIndex: pageIndex,
						PageSize:  pageSize,
					},
				}
			},
			SortKeys: slices.Sorted(maps.Keys(reportOrderKeys())),
			SearchMatch: func(r Report) string {
				return r.Name
			},
			NewInvalid: func() (ReportCreateMutation, bool) {
				return ReportCreateMutation{
					ReportMutation: ReportMutation{
						Name:  "",
						Query: "SELECT 1;",
					},
				}, true
			},
		},
	)
}
