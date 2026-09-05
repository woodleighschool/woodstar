//go:build postgres

package entra_test

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/woodleighschool/goodies/pglock"

	"github.com/woodleighschool/woodstar/internal/backgroundjobs"
	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/directory/entra"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestTwoSchedulersRunOneDueEntraSync(t *testing.T) {
	pool, _ := testdb.Open(t)
	started := make(chan struct{})
	release := make(chan struct{})
	applied := make(chan struct{})
	fetcher := &countingFetcher{started: started, release: release}
	applier := &countingApplier{applied: applied}
	service := entra.NewService(applier, fetcher, slog.New(slog.DiscardHandler))

	newRuntime := func() *backgroundjobs.Runtime {
		workers := river.NewWorkers()
		river.AddWorker(
			workers,
			entra.NewSyncWorker(
				service,
				pglock.New(pool, entra.SyncAdvisoryLockID),
			),
		)
		periodic := river.NewPeriodicJob(
			river.PeriodicInterval(time.Hour),
			func() (river.JobArgs, *river.InsertOpts) {
				return entra.SyncJobArgs{Trigger: backgroundjobs.TriggerScheduled}, nil
			},
			&river.PeriodicJobOpts{ID: entra.SyncJobKind, RunOnStart: true},
		)
		runtime, err := backgroundjobs.New(
			pool,
			workers,
			[]*river.PeriodicJob{periodic},
			slog.New(slog.DiscardHandler),
		)
		if err != nil {
			t.Fatalf("configure background jobs: %v", err)
		}
		return runtime
	}

	first := newRuntime()
	second := newRuntime()
	ctx, cancel := context.WithCancel(t.Context())
	if err := first.Start(ctx); err != nil {
		t.Fatalf("start first scheduler: %v", err)
	}
	if err := second.Start(ctx); err != nil {
		t.Fatalf("start second scheduler: %v", err)
	}

	select {
	case <-started:
		close(release)
	case <-time.After(15 * time.Second):
		cancel()
		t.Fatal("timed out waiting for scheduled Entra fetch")
	}

	select {
	case <-applied:
		cancel()
	case <-time.After(15 * time.Second):
		cancel()
		t.Fatal("timed out waiting for scheduled Entra sync")
	}

	var stopGroup sync.WaitGroup
	for _, runtime := range []*backgroundjobs.Runtime{first, second} {
		stopGroup.Go(func() {
			if err := runtime.Stop(t.Context()); err != nil {
				t.Errorf("stop scheduler: %v", err)
			}
		})
	}
	stopGroup.Wait()

	if got := fetcher.calls.Load(); got != 1 {
		t.Fatalf("fetch calls = %d, want 1", got)
	}
	if got := applier.calls.Load(); got != 1 {
		t.Fatalf("apply calls = %d, want 1", got)
	}
}

type countingFetcher struct {
	calls   atomic.Int64
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (f *countingFetcher) Fetch(context.Context) (directory.ProviderSnapshot, error) {
	f.calls.Add(1)
	f.once.Do(func() { close(f.started) })
	<-f.release
	return directory.ProviderSnapshot{}, nil
}

type countingApplier struct {
	calls   atomic.Int64
	applied chan struct{}
	once    sync.Once
}

func (a *countingApplier) ApplyProviderSnapshot(
	context.Context,
	directory.Source,
	directory.ProviderSnapshot,
) error {
	a.calls.Add(1)
	a.once.Do(func() { close(a.applied) })
	return nil
}
