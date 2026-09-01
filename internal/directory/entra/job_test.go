package entra

import (
	"context"
	"errors"
	"testing"

	"github.com/riverqueue/river"
)

func TestSyncWorkerSnoozesWhileAnotherReplicaIsSyncing(t *testing.T) {
	worker := NewSyncWorker(nil, busySyncLocker{})

	_, err := worker.run(t.Context())
	snoozeErr, ok := errors.AsType[*river.JobSnoozeError](err)
	if !ok {
		t.Fatalf("run error = %v, want River snooze", err)
	}
	if snoozeErr.Duration <= 0 {
		t.Fatalf("snooze duration = %v, want a positive duration", snoozeErr.Duration)
	}
}

func TestDisabledSyncReportsIdleAndRejectsTrigger(t *testing.T) {
	jobs := NewSyncJobs(false, nil)

	status, err := jobs.Status(t.Context())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.Enabled || status.Activity != "idle" {
		t.Fatalf("Status() = %+v, want disabled and idle", status)
	}

	if _, err := jobs.Trigger(t.Context()); !errors.Is(err, ErrSyncDisabled) {
		t.Fatalf("Trigger() error = %v, want ErrSyncDisabled", err)
	}
}

type busySyncLocker struct{}

func (busySyncLocker) Try(context.Context, func(context.Context) error) (bool, error) {
	return false, nil
}
