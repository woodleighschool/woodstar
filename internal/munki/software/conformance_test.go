package software

import (
	"context"
	"log/slog"
	"maps"
	"slices"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/database/dbtest/crudtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/storage"
)

func TestSoftwareStoreConformance(t *testing.T) {
	db, ctx := dbtest.Open(t)
	objectStore := storage.NewObjectStore(db, nil)
	packageStore := packages.NewStore(db, objectStore, slog.New(slog.DiscardHandler))
	store := NewStore(db, objectStore, packageStore)

	crudtest.RunConformance(
		t,
		ctx,
		crudtest.Fixtures[Software, CreateMutation, UpdateMutation, dbutil.ListParams]{
			Store: store,
			NewValid: func(_ *testing.T, _ context.Context) CreateMutation {
				return CreateMutation{
					Name:     "Conformance App",
					Category: "Utilities",
				}
			},
			Mutate: func(_ Software) UpdateMutation {
				return UpdateMutation{Category: "Productivity"}
			},
			ID:         func(sw Software) int64 { return sw.ID },
			ListParams: softwareListParams,
			SortKeys:   slices.Sorted(maps.Keys(softwareOrderKeys())),
			SearchMatch: func(sw Software) string {
				return sw.Name
			},
			NewInvalid: func() (CreateMutation, bool) {
				return CreateMutation{Name: ""}, true
			},
		},
	)
}

func softwareListParams(q, sort string, pageIndex, pageSize int32) dbutil.ListParams {
	return dbutil.ListParams{
		Q:         q,
		Sort:      sort,
		PageIndex: pageIndex,
		PageSize:  pageSize,
	}
}
