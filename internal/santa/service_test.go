package santa

import (
	"context"
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/heartbeats"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	santaevents "github.com/woodleighschool/woodstar/internal/santa/events"
	santarules "github.com/woodleighschool/woodstar/internal/santa/rules"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
)

func TestSyncServiceRecordsContactForEveryStageAfterMachineResolution(t *testing.T) {
	contact := heartbeats.Contact{RemoteIP: "203.0.113.15", UserAgent: "santad/2026.1"}

	for _, tc := range []struct {
		name string
		call func(*SyncService) error
	}{
		{
			name: "preflight",
			call: func(service *SyncService) error {
				_, err := service.Preflight(t.Context(), "machine-1", contact, PreflightRequest{})
				return err
			},
		},
		{
			name: "event upload",
			call: func(service *SyncService) error {
				_, err := service.EventUpload(t.Context(), "machine-1", contact, EventUploadRequest{})
				return err
			},
		},
		{
			name: "rule download",
			call: func(service *SyncService) error {
				_, err := service.RuleDownload(t.Context(), "machine-1", contact, RuleDownloadRequest{})
				return err
			},
		},
		{
			name: "postflight",
			call: func(service *SyncService) error {
				_, err := service.Postflight(t.Context(), "machine-1", contact, PostflightRequest{})
				return err
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := &heartbeatRecorder{}
			service, _ := newTestSyncService(1, nil, recorder)
			if err := tc.call(service); err != nil {
				t.Fatalf("sync stage: %v", err)
			}
			if got, want := recorder.records, []heartbeatRecord{{
				hostID:  1,
				source:  heartbeats.SourceSanta,
				contact: contact,
			}}; len(got) != len(want) || got[0] != want[0] {
				t.Fatalf("records = %+v, want %+v", got, want)
			}
		})
	}
}

func TestSyncServiceDoesNotRecordUnknownMachine(t *testing.T) {
	for _, tc := range []struct {
		name string
		call func(*SyncService) error
	}{
		{
			name: "preflight",
			call: func(service *SyncService) error {
				_, err := service.Preflight(t.Context(), "unknown", heartbeats.Contact{}, PreflightRequest{})
				return err
			},
		},
		{
			name: "event upload",
			call: func(service *SyncService) error {
				_, err := service.EventUpload(t.Context(), "unknown", heartbeats.Contact{}, EventUploadRequest{})
				return err
			},
		},
		{
			name: "rule download",
			call: func(service *SyncService) error {
				_, err := service.RuleDownload(t.Context(), "unknown", heartbeats.Contact{}, RuleDownloadRequest{})
				return err
			},
		},
		{
			name: "postflight",
			call: func(service *SyncService) error {
				_, err := service.Postflight(t.Context(), "unknown", heartbeats.Contact{}, PostflightRequest{})
				return err
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := &heartbeatRecorder{}
			service, _ := newTestSyncService(0, dbutil.ErrNotFound, recorder)
			err := tc.call(service)

			if !errors.Is(err, dbutil.ErrNotFound) {
				t.Fatalf("sync stage error = %v, want not found", err)
			}
			if len(recorder.records) != 0 {
				t.Fatalf("records = %+v, want none", recorder.records)
			}
		})
	}
}

func TestSyncServicePropagatesRecorderErrorsBeforeStageWork(t *testing.T) {
	recorder := &heartbeatRecorder{err: errors.New("record heartbeat")}
	service, eventStore := newTestSyncService(1, nil, recorder)

	_, err := service.EventUpload(t.Context(), "machine-1", heartbeats.Contact{}, EventUploadRequest{})

	if !errors.Is(err, recorder.err) {
		t.Fatalf("EventUpload error = %v, want recorder error", err)
	}
	if eventStore.calls != 0 {
		t.Fatalf("event store calls = %d, want 0", eventStore.calls)
	}
}

func TestSyncServiceResolvesRulesInsideSelectedConfiguration(t *testing.T) {
	ruleStore := &recordingRuleStore{rules: []santarules.HostRule{{
		RuleID:     1,
		RuleType:   santarules.RuleTypeSigningID,
		Identifier: "ABCDE12345:com.example.app",
		Policy:     santarules.PolicyBlocklist,
	}}}
	syncStore := &recordingSyncStore{}
	service := NewSyncService(Dependencies{
		HostStore: &testHostStore{hostID: 7},
		Configurations: staticConfigurationResolver{match: &configurations.ConfigurationMatch{
			Configuration: configurations.Configuration{ID: 42},
		}},
		Events:     &testEventStore{},
		Rules:      ruleStore,
		Sync:       syncStore,
		Heartbeats: &heartbeatRecorder{},
	})

	if _, err := service.Preflight(t.Context(), "machine-1", heartbeats.Contact{}, PreflightRequest{}); err != nil {
		t.Fatalf("Preflight: %v", err)
	}
	if ruleStore.hostID != 7 || ruleStore.configurationID != 42 {
		t.Fatalf(
			"rule resolution = host %d configuration %d, want host 7 configuration 42",
			ruleStore.hostID,
			ruleStore.configurationID,
		)
	}
	if syncStore.calls != 1 || len(syncStore.targets) != 1 ||
		syncStore.targets[0].Identifier != "ABCDE12345:com.example.app" {
		t.Fatalf("prepared targets = %+v after %d calls, want selected configuration rule", syncStore.targets, syncStore.calls)
	}
}

func TestSyncServiceRejectsConflictingExpandedRulesBeforePreparingSync(t *testing.T) {
	ruleStore := &recordingRuleStore{rules: []santarules.HostRule{
		{
			RuleID:     1,
			RuleType:   santarules.RuleTypeBinary,
			Identifier: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Policy:     santarules.PolicyAllowlist,
		},
		{
			RuleID:     2,
			RuleType:   santarules.RuleTypeBinary,
			Identifier: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Policy:     santarules.PolicyBlocklist,
		},
	}}
	syncStore := &recordingSyncStore{}
	service := NewSyncService(Dependencies{
		HostStore: &testHostStore{hostID: 7},
		Configurations: staticConfigurationResolver{match: &configurations.ConfigurationMatch{
			Configuration: configurations.Configuration{ID: 42},
		}},
		Events:     &testEventStore{},
		Rules:      ruleStore,
		Sync:       syncStore,
		Heartbeats: &heartbeatRecorder{},
	})

	_, err := service.Preflight(t.Context(), "machine-1", heartbeats.Contact{}, PreflightRequest{})
	if !errors.Is(err, dbutil.ErrConflict) {
		t.Fatalf("Preflight conflict error = %v, want ErrConflict", err)
	}
	if syncStore.calls != 0 {
		t.Fatalf("PreparePending calls = %d, want 0", syncStore.calls)
	}
}

func newTestSyncService(hostID int64, hostErr error, recorder contactRecorder) (*SyncService, *testEventStore) {
	eventStore := &testEventStore{}
	service := NewSyncService(Dependencies{
		HostStore:      &testHostStore{hostID: hostID, err: hostErr},
		Configurations: testConfigurationResolver{},
		Events:         eventStore,
		Rules:          testRuleStore{},
		Sync:           testSyncStore{},
		Heartbeats:     recorder,
	})
	return service, eventStore
}

type testHostStore struct {
	hostID int64
	err    error
}

func (s *testHostStore) hostIDByMachineID(context.Context, string) (int64, error) {
	return s.hostID, s.err
}

func (*testHostStore) UpsertHostObservation(context.Context, HostObservation) error { return nil }

type testConfigurationResolver struct{}

func (testConfigurationResolver) ResolveConfigurationForHost(context.Context, int64) (*configurations.ConfigurationMatch, error) {
	return nil, nil
}

type staticConfigurationResolver struct {
	match *configurations.ConfigurationMatch
}

func (r staticConfigurationResolver) ResolveConfigurationForHost(
	context.Context,
	int64,
) (*configurations.ConfigurationMatch, error) {
	return r.match, nil
}

type testEventStore struct{ calls int }

func (s *testEventStore) IngestEvents(
	context.Context,
	int64,
	[]santaevents.ExecutionEventInput,
	[]santaevents.FileAccessEventInput,
	[]santaevents.StandaloneRuleCreationEventInput,
) ([]string, error) {
	s.calls++
	return nil, nil
}

type testRuleStore struct{}

func (testRuleStore) ResolveRulesForHost(context.Context, int64, int64) ([]santarules.HostRule, error) {
	return nil, nil
}

type recordingRuleStore struct {
	hostID          int64
	configurationID int64
	rules           []santarules.HostRule
}

func (s *recordingRuleStore) ResolveRulesForHost(
	_ context.Context,
	hostID int64,
	configurationID int64,
) ([]santarules.HostRule, error) {
	s.hostID = hostID
	s.configurationID = configurationID
	return s.rules, nil
}

type testSyncStore struct{}

func (testSyncStore) PreparePending(
	context.Context,
	int64,
	[]syncstate.Target,
	syncstate.RuleCounts,
	bool,
	string,
) (syncstate.SyncType, error) {
	return syncstate.SyncTypeNormal, nil
}

type recordingSyncStore struct {
	testSyncStore

	calls   int
	targets []syncstate.Target
}

func (s *recordingSyncStore) PreparePending(
	_ context.Context,
	_ int64,
	targets []syncstate.Target,
	_ syncstate.RuleCounts,
	_ bool,
	_ string,
) (syncstate.SyncType, error) {
	s.calls++
	s.targets = targets
	return syncstate.SyncTypeNormal, nil
}

func (testSyncStore) LoadPendingPayloadPage(
	context.Context,
	int64,
	string,
	int32,
) (syncstate.PayloadRulePage, error) {
	return syncstate.PayloadRulePage{}, nil
}

func (testSyncStore) PromotePending(
	context.Context,
	int64,
	uint32,
	uint32,
	syncstate.SyncType,
	string,
) error {
	return nil
}

type heartbeatRecord struct {
	hostID  int64
	source  heartbeats.Source
	contact heartbeats.Contact
}

type heartbeatRecorder struct {
	records []heartbeatRecord
	err     error
}

func (r *heartbeatRecorder) Record(
	_ context.Context,
	hostID int64,
	source heartbeats.Source,
	contact heartbeats.Contact,
) error {
	r.records = append(r.records, heartbeatRecord{hostID: hostID, source: source, contact: contact})
	return r.err
}
