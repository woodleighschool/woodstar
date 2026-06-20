package configurations

import (
	"context"
	"maps"
	"slices"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/database/dbtest/crudtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func TestConfigurationsConformance(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := NewStore(db)

	crudtest.RunConformance(
		t,
		ctx,
		crudtest.Fixtures[Configuration, ConfigurationMutation, ConfigurationMutation, ConfigurationListParams]{
			Store: store,
			NewValid: func(_ *testing.T, _ context.Context) ConfigurationMutation {
				return ConfigurationMutation{
					Name:                    "Conformance Config",
					ClientMode:              ClientModeMonitor,
					FullSyncIntervalSeconds: 600,
					BatchSize:               50,
				}
			},
			Mutate: func(_ Configuration) ConfigurationMutation {
				return ConfigurationMutation{
					Name:                    "Conformance Config Updated",
					ClientMode:              ClientModeLockdown,
					FullSyncIntervalSeconds: 120,
					BatchSize:               10,
				}
			},
			ID:         func(c Configuration) int64 { return c.ID },
			ListParams: configurationListParamsFromKnobs,
			SortKeys:   slices.Sorted(maps.Keys(configurationOrderKeys())),
			SearchMatch: func(c Configuration) string {
				return c.Name
			},
			NewInvalid: func() (ConfigurationMutation, bool) {
				return ConfigurationMutation{
					Name:                    "Invalid",
					ClientMode:              ClientMode("bogus"),
					FullSyncIntervalSeconds: 600,
					BatchSize:               50,
				}, true
			},
		},
	)
}

func configurationListParamsFromKnobs(q, sort string, pageIndex, pageSize int32) ConfigurationListParams {
	return ConfigurationListParams{
		ListParams: dbutil.ListParams{
			Q:         q,
			Sort:      sort,
			PageIndex: pageIndex,
			PageSize:  pageSize,
		},
	}
}
