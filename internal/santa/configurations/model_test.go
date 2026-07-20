package configurations_test

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

func baseline(name string) configurations.ConfigurationMutation {
	return configurations.ConfigurationMutation{
		Name:                     name,
		ClientMode:               configurations.ClientModeMonitor,
		OverrideFileAccessAction: configurations.FileAccessActionNone,
		FullSyncIntervalSeconds:  600,
		BatchSize:                50,
	}
}

func TestConfigurationMutationValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*configurations.ConfigurationMutation)
	}{
		{name: "short sync", mutate: func(m *configurations.ConfigurationMutation) {
			m.FullSyncIntervalSeconds = 59
		}},
		{name: "tiny batch", mutate: func(m *configurations.ConfigurationMutation) {
			m.BatchSize = 1
		}},
		{name: "missing name", mutate: func(m *configurations.ConfigurationMutation) {
			m.Name = ""
		}},
		{name: "empty client mode", mutate: func(m *configurations.ConfigurationMutation) {
			m.ClientMode = ""
		}},
		{name: "invalid file access action", mutate: func(m *configurations.ConfigurationMutation) {
			m.OverrideFileAccessAction = ""
		}},
		{name: "invalid label", mutate: func(m *configurations.ConfigurationMutation) {
			m.Targets = configurationTargets(labelRefs(0), nil)
		}},
		{name: "duplicate targets", mutate: func(m *configurations.ConfigurationMutation) {
			m.Targets = configurationTargets(labelRefs(1, 1), nil)
		}},
		{name: "overlapping targets", mutate: func(m *configurations.ConfigurationMutation) {
			m.Targets = configurationTargets(labelRefs(1), labelRefs(1))
		}},
		{name: "remount without flags", mutate: func(m *configurations.ConfigurationMutation) {
			m.RemovableMediaPolicy = configurations.RemovableMediaPolicy{
				Action: configurations.RemovableMediaActionRemount,
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mutation := baseline(tt.name)
			tt.mutate(&mutation)
			if err := mutation.Validate(); !errors.Is(err, dbutil.ErrInvalidInput) {
				t.Fatalf("Validate error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func configurationTargets(include, exclude []targeting.LabelRef) configurations.ConfigurationTargets {
	return configurations.ConfigurationTargets{Include: include, Exclude: exclude}
}

func labelRefs(labelIDs ...int64) []targeting.LabelRef {
	refs := make([]targeting.LabelRef, len(labelIDs))
	for i, labelID := range labelIDs {
		refs[i] = targeting.LabelRef{LabelID: labelID}
	}
	return refs
}
