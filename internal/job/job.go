// Package job runs context-cancellable background goroutines with a Stop
// that waits for the goroutine to finish before returning.
package job

import "context"

// Handle owns one background goroutine. The zero value is a no-op.
type Handle struct {
	stop context.CancelFunc
	done <-chan struct{}
}

// Stop cancels the goroutine's context and blocks until the goroutine exits.
// Safe to call on a nil handle.
func (h *Handle) Stop() {
	if h == nil {
		return
	}
	h.stop()
	<-h.done
}

// Start launches fn as a background goroutine that observes ctx for
// cancellation. The returned handle stops the goroutine and waits for it to
// finish.
func Start(parent context.Context, fn func(ctx context.Context)) *Handle {
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn(ctx)
	}()
	return &Handle{stop: cancel, done: done}
}
