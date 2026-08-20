package configurations_test

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/fault"
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
			m.RemovableMediaPolicy = &configurations.RemovableMediaPolicy{
				Action: configurations.RemovableMediaActionRemount,
			}
		}},
		{name: "unknown remount flag", mutate: func(m *configurations.ConfigurationMutation) {
			m.RemovableMediaPolicy = &configurations.RemovableMediaPolicy{
				Action:       configurations.RemovableMediaActionRemount,
				RemountFlags: []configurations.RemountFlag{"unknown"},
			}
		}},
		{name: "duplicate remount flag", mutate: func(m *configurations.ConfigurationMutation) {
			m.RemovableMediaPolicy = &configurations.RemovableMediaPolicy{
				Action: configurations.RemovableMediaActionRemount,
				RemountFlags: []configurations.RemountFlag{
					configurations.RemountFlagNoExec,
					configurations.RemountFlagNoExec,
				},
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mutation := baseline(tt.name)
			tt.mutate(&mutation)
			if err := mutation.Validate(); !errors.Is(err, fault.ErrInvalidInput) {
				t.Fatalf("Validate error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestConfigurationMutationAcceptsEveryRemountFlag(t *testing.T) {
	t.Parallel()

	for _, flag := range configurations.RemountFlagValues {
		t.Run(string(flag), func(t *testing.T) {
			t.Parallel()
			mutation := baseline("Remount " + string(flag))
			mutation.RemovableMediaPolicy = &configurations.RemovableMediaPolicy{
				Action:       configurations.RemovableMediaActionRemount,
				RemountFlags: []configurations.RemountFlag{flag},
			}
			if err := mutation.Validate(); err != nil {
				t.Fatalf("Validate error = %v, want valid remount flag", err)
			}
		})
	}
}

func TestSyncPolicyDigestIncludesIdentityAndExcludesDisplayMetadata(t *testing.T) {
	t.Parallel()

	configuration := &configurations.Configuration{
		ID:                       1,
		Name:                     "First",
		Description:              "First description",
		ClientMode:               configurations.ClientModeLockdown,
		OverrideFileAccessAction: configurations.FileAccessActionNone,
		FullSyncIntervalSeconds:  600,
		BatchSize:                50,
		BlockedPathRegex:         new(`^/private/tmp/`),
	}
	first, err := configurations.SyncPolicyDigest(configuration)
	if err != nil {
		t.Fatalf("digest first configuration: %v", err)
	}
	configuration.Name = "Second"
	configuration.Description = "Second description"
	configuration.Position = 9
	configuration.Targets = configurationTargets(labelRefs(99), nil)
	second, err := configurations.SyncPolicyDigest(configuration)
	if err != nil {
		t.Fatalf("digest second configuration: %v", err)
	}
	if first != second {
		t.Fatalf("metadata changed digest from %q to %q", first, second)
	}

	configuration.ID = 2
	changedIdentity, err := configurations.SyncPolicyDigest(configuration)
	if err != nil {
		t.Fatalf("digest changed identity: %v", err)
	}
	if changedIdentity == first {
		t.Fatal("changing configuration identity did not change the digest")
	}

	configuration.ID = 1
	configuration.BlockedPathRegex = nil
	changedSettings, err := configurations.SyncPolicyDigest(configuration)
	if err != nil {
		t.Fatalf("digest changed settings: %v", err)
	}
	if changedSettings == first {
		t.Fatal("removing a device setting did not change the digest")
	}
}

func TestSyncPolicyDigestDistinguishesNoConfigurationAndConcreteSettings(t *testing.T) {
	t.Parallel()

	undefined, err := configurations.SyncPolicyDigest(nil)
	if err != nil {
		t.Fatalf("digest undefined settings: %v", err)
	}
	configuration := configurations.Configuration{
		ID:                       1,
		ClientMode:               configurations.ClientModeMonitor,
		OverrideFileAccessAction: configurations.FileAccessActionNone,
		FullSyncIntervalSeconds:  600,
		BatchSize:                50,
	}
	configured, err := configurations.SyncPolicyDigest(&configuration)
	if err != nil {
		t.Fatalf("digest configured defaults: %v", err)
	}
	configuration.EnableBundles = true
	enabled, err := configurations.SyncPolicyDigest(&configuration)
	if err != nil {
		t.Fatalf("digest enabled bundles: %v", err)
	}
	if undefined == configured {
		t.Fatal("no configuration and a selected configuration have the same digest")
	}
	if configured == enabled {
		t.Fatal("changing a concrete setting did not change the digest")
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
