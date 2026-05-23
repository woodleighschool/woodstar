package reports

import (
	"context"
	"log/slog"
	"time"
)

const defaultMaxReportRows = 1000

// CleanupOptions controls periodic report-result maintenance.
type CleanupOptions struct {
	MaxReportRows int
}

type Cleanup struct {
	stop context.CancelFunc
	done <-chan struct{}
}

// Stop cancels the cleanup loop and waits for it to exit.
func (c *Cleanup) Stop() {
	if c == nil {
		return
	}
	c.stop()
	<-c.done
}

// StartCleanup starts the periodic report-row trimmer. The goroutine exits
// when ctx is cancelled or the returned cleanup handle is stopped.
func StartCleanup(
	ctx context.Context,
	store *Store,
	options CleanupOptions,
	logger *slog.Logger,
) *Cleanup {
	if store == nil {
		return nil
	}
	if options.MaxReportRows <= 0 {
		options.MaxReportRows = defaultMaxReportRows
	}
	ctx, stop := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		reportTrimLoop(ctx, store, options, logger)
	}()
	return &Cleanup{stop: stop, done: done}
}

func reportTrimLoop(
	ctx context.Context,
	store *Store,
	options CleanupOptions,
	logger *slog.Logger,
) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := store.TrimResults(ctx, options.MaxReportRows); err != nil && logger != nil {
				logger.WarnContext(ctx, "report row cleanup failed", "err", err)
			}
		}
	}
}
