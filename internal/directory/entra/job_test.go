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
	var snoozeErr *river.JobSnoozeError
	if !errors.As(err, &snoozeErr) {
		t.Fatalf("run error = %v, want River snooze", err)
	}
	if snoozeErr.Duration <= 0 {
		t.Fatalf("snooze duration = %v, want a positive duration", snoozeErr.Duration)
	}
}

type busySyncLocker struct{}

func (busySyncLocker) Try(context.Context, func(context.Context) error) (bool, error) {
	return false, nil
}
