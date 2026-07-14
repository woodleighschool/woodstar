package entra_test

import (
	"context"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/directory/entra"
)

func TestSchedulerStopWaitsForInFlightSync(t *testing.T) {
	fetcher := &blockingFetcher{started: make(chan struct{})}
	service := entra.NewService(discardingApplier{}, fetcher, slog.New(slog.DiscardHandler))
	stop := service.StartScheduler(t.Context(), time.Hour)

	select {
	case <-fetcher.started:
	case <-time.After(time.Second):
		t.Fatal("Entra sync did not start")
	}
	stop()
	if !fetcher.done.Load() {
		t.Fatal("scheduler stop returned before sync observed cancellation")
	}
}

type discardingApplier struct{}

func (discardingApplier) ApplyProviderSnapshot(
	context.Context,
	directory.Source,
	directory.ProviderSnapshot,
) error {
	return nil
}

type blockingFetcher struct {
	started chan struct{}
	done    atomic.Bool
}

func (f *blockingFetcher) Fetch(ctx context.Context) (directory.ProviderSnapshot, error) {
	close(f.started)
	<-ctx.Done()
	f.done.Store(true)
	return directory.ProviderSnapshot{}, ctx.Err()
}
