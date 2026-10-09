//go:build postgres

package packages_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/woodleighschool/goodies/bloby"
	blobydb "github.com/woodleighschool/goodies/bloby/pgxstore"
	"github.com/woodleighschool/woodstar/internal/backgroundjobs"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/testutil/testbloby"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestInstallerFinalizationLifecycle(t *testing.T) { //nolint:funlen // Lifecycle assertions share one retained upload and job.
	db, ctx := testdb.Open(t)
	objects := testbloby.New(t, db)
	workers := river.NewWorkers()
	river.AddWorker(workers, packages.NewFinalizeInstallerWorker(objects))
	jobs, err := backgroundjobs.New(db, workers, nil, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	finalizations := packages.NewFinalizations(db, objects, jobs)
	registry := blobydb.New(db)
	reserve := func() int64 {
		t.Helper()
		object, action, err := objects.Begin(ctx, packages.ObjectPrefix, "example.pkg", 7)
		if err != nil {
			t.Fatal(err)
		}
		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(ctx, http.MethodPut, action.Target.URL, strings.NewReader("payload"))
		objects.TransferHandler().ServeHTTP(recorder, request)
		if recorder.Code != 204 {
			t.Fatalf("upload: %d %s", recorder.Code, recorder.Body.String())
		}
		return object.ID
	}
	id := reserve()
	var group sync.WaitGroup
	for range 12 {
		group.Go(func() {
			object, err := finalizations.Ensure(ctx, id)
			if err != nil || object != nil {
				t.Errorf("ensure: %v %v", object, err)
			}
		})
	}
	group.Wait()
	var count int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM river_job WHERE kind = 'munki_finalize_installer'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("jobs: %d %v", count, err)
	}
	otherID := reserve()
	if _, err := finalizations.Ensure(ctx, otherID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(ctx, `SELECT count(*) FROM river_job WHERE kind = 'munki_finalize_installer'`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("independent installer jobs: %d %v", count, err)
	}
	if err := objects.Delete(ctx, id, packages.ObjectPrefix); !errors.Is(err, bloby.ErrConflict) {
		t.Fatalf("queued deletion: %v", err)
	}
	// Even uploads older than the orphan cutoff belong to queued verification.
	if _, err := db.Exec(ctx, `UPDATE storage_objects SET updated_at = now() - interval '48 hours' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	expired, err := registry.ClaimExpiredPending(ctx, time.Now().Add(-24*time.Hour), time.Now(), 100)
	if err != nil || len(expired) != 0 {
		t.Fatalf("claimed retained upload: %v %v", expired, err)
	}
	// A disconnected polling request cannot cancel the durable job.
	disconnected, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := finalizations.Ensure(disconnected, id); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled request: %v", err)
	}
	// Starting a fresh service/runtime after enqueue exercises durable recovery.
	finalizations = packages.NewFinalizations(db, objects, jobs)
	if err := jobs.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := jobs.Stop(context.Background()); err != nil {
			t.Error(err)
		}
	})
	awaitFinalization(t, func() bool {
		object, err := finalizations.Ensure(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		return object != nil && object.SizeBytesValue() == 7 && object.SHA256Value() == "239f59ed55e737c77147cf55ad0c1b030b6d7ee748a7426952f9b852d5a935e5"
	})
	awaitFinalization(t, func() bool {
		var state string
		err := db.QueryRow(ctx, `SELECT state FROM munki_installer_finalizations WHERE object_id=$1`, id).Scan(&state)
		return err == nil && state == "completed"
	})
	if _, err := db.Exec(ctx, `DELETE FROM river_job WHERE id=(SELECT job_id FROM munki_installer_finalizations WHERE object_id=$1)`, id); err != nil {
		t.Fatal(err)
	}
	if object, err := finalizations.Ensure(ctx, id); err != nil || object == nil {
		t.Fatalf("completed after job pruning: %v %v", object, err)
	}
	if err := objects.Delete(ctx, id, packages.ObjectPrefix); err != nil {
		t.Fatalf("delete completed: %v", err)
	}
}

func TestInstallerFinalizationTerminalStateAndDeletion(t *testing.T) { //nolint:gocognit // Exercise each River state against polling and deletion.
	db, ctx := testdb.Open(t)
	objects := testbloby.New(t, db)
	workers := river.NewWorkers()
	river.AddWorker(workers, packages.NewFinalizeInstallerWorker(objects))
	jobs, err := backgroundjobs.New(db, workers, nil, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	service := packages.NewFinalizations(db, objects, jobs)
	for _, terminal := range []string{"discarded", "cancelled", "deleted"} {
		t.Run(terminal, func(t *testing.T) {
			object, _, err := objects.Begin(ctx, packages.ObjectPrefix, "missing.pkg", 1)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := service.Ensure(ctx, object.ID); err != nil {
				t.Fatal(err)
			}
			var jobID int64
			if err := db.QueryRow(ctx, `SELECT job_id FROM munki_installer_finalizations WHERE object_id=$1`, object.ID).Scan(&jobID); err != nil {
				t.Fatal(err)
			}
			for _, state := range []string{"running", "retryable", "available"} {
				if _, err := db.Exec(ctx, `UPDATE river_job SET state=$2::river_job_state WHERE id=$1`, jobID, state); err != nil {
					t.Fatal(err)
				}
				if got, err := service.Ensure(ctx, object.ID); err != nil || got != nil {
					t.Fatalf("%s polling: %v %v", state, got, err)
				}
				if err := objects.Delete(ctx, object.ID, packages.ObjectPrefix); !errors.Is(err, bloby.ErrConflict) {
					t.Fatalf("%s deletion: %v", state, err)
				}
			}
			if terminal == "deleted" {
				_, err = db.Exec(ctx, `DELETE FROM river_job WHERE id=$1`, jobID)
			} else {
				_, err = db.Exec(ctx, `UPDATE river_job SET state=$2::river_job_state, finalized_at=now() WHERE id=$1`, jobID, terminal)
			}
			if err != nil {
				t.Fatal(err)
			}
			for range 3 {
				if _, err := service.Ensure(ctx, object.ID); !errors.Is(err, packages.ErrFinalizationFailed) {
					t.Fatalf("terminal polling: %v", err)
				}
			}
			if _, err := db.Exec(ctx, `DELETE FROM river_job WHERE id=$1`, jobID); err != nil {
				t.Fatal(err)
			}
			if _, err := service.Ensure(ctx, object.ID); !errors.Is(err, packages.ErrFinalizationFailed) {
				t.Fatalf("failure after pruning: %v", err)
			}
			if err := objects.Delete(ctx, object.ID, packages.ObjectPrefix); err != nil {
				t.Fatal(err)
			}
		})
	}
	t.Run("delete races enqueue", func(t *testing.T) {
		for range 12 {
			object, _, err := objects.Begin(ctx, packages.ObjectPrefix, "race.pkg", 1)
			if err != nil {
				t.Fatal(err)
			}
			var ensureErr, deleteErr error
			var group sync.WaitGroup
			group.Go(func() { _, ensureErr = service.Ensure(ctx, object.ID) })
			group.Go(func() { deleteErr = objects.Delete(ctx, object.ID, packages.ObjectPrefix) })
			group.Wait()
			enqueued := ensureErr == nil && errors.Is(deleteErr, bloby.ErrConflict)
			deleted := deleteErr == nil && errors.Is(ensureErr, bloby.ErrNotFound)
			if !enqueued && !deleted {
				t.Fatalf("ensure=%v delete=%v", ensureErr, deleteErr)
			}
		}
	})
}

func awaitFinalization(t *testing.T, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if ready() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for verification")
}

func TestInstallerFinalizationExhaustsRetries(t *testing.T) {
	db, ctx := testdb.Open(t)
	objects := testbloby.New(t, db)
	workers := river.NewWorkers()
	river.AddWorker(workers, packages.NewFinalizeInstallerWorker(objects))
	jobs, err := backgroundjobs.New(db, workers, nil, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	service := packages.NewFinalizations(db, objects, jobs)
	object, _, err := objects.Begin(ctx, packages.ObjectPrefix, "missing.pkg", 7)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Ensure(ctx, object.ID); err != nil {
		t.Fatal(err)
	}
	// One attempt exercises River's real exhaustion transition without waiting
	// through production backoff. The bytes were never uploaded.
	if _, err := db.Exec(ctx, `UPDATE river_job SET max_attempts=1 WHERE id=(SELECT job_id FROM munki_installer_finalizations WHERE object_id=$1)`, object.ID); err != nil {
		t.Fatal(err)
	}
	if err := jobs.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := jobs.Stop(context.Background()); err != nil {
			t.Error(err)
		}
	})
	awaitFinalization(t, func() bool {
		_, err := service.Ensure(ctx, object.ID)
		return errors.Is(err, packages.ErrFinalizationFailed)
	})
	var attempt int
	if err := db.QueryRow(ctx, `SELECT attempt FROM river_job WHERE id=(SELECT job_id FROM munki_installer_finalizations WHERE object_id=$1)`, object.ID).Scan(&attempt); err != nil || attempt != 1 {
		t.Fatalf("attempt=%d err=%v", attempt, err)
	}
	if err := objects.Delete(ctx, object.ID, packages.ObjectPrefix); err != nil {
		t.Fatal(err)
	}
}

type interruptedRegistry struct {
	*blobydb.Store

	started chan struct{}
}

func (r *interruptedRegistry) RefreshPending(ctx context.Context, _ int64) (*bloby.Object, error) {
	close(r.started)
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestInstallerFinalizationSurvivesWorkerRestart(t *testing.T) {
	db, ctx := testdb.Open(t)
	registry := &interruptedRegistry{Store: blobydb.New(db), started: make(chan struct{})}
	cfg := bloby.Config{Kind: bloby.KindFile, TransferTTL: time.Minute, File: bloby.FileConfig{Root: t.TempDir(), BaseURL: "https://storage.invalid", CapabilityKeyHex: strings.Repeat("42", 32)}}
	blocked, err := bloby.New(ctx, registry, cfg, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	objects, err := bloby.New(ctx, blobydb.New(db), cfg, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	newJobs := func(storage *bloby.Service) *backgroundjobs.Runtime {
		workers := river.NewWorkers()
		river.AddWorker(workers, packages.NewFinalizeInstallerWorker(storage))
		jobs, err := backgroundjobs.New(db, workers, nil, slog.New(slog.DiscardHandler))
		if err != nil {
			t.Fatal(err)
		}
		return jobs
	}
	first := newJobs(blocked)
	service := packages.NewFinalizations(db, objects, first)
	object, action, err := objects.Begin(ctx, packages.ObjectPrefix, "restart.pkg", 7)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	objects.TransferHandler().ServeHTTP(rec, httptest.NewRequestWithContext(ctx, http.MethodPut, action.Target.URL, strings.NewReader("payload")))
	if rec.Code != 204 {
		t.Fatalf("upload: %d", rec.Code)
	}
	if _, err := service.Ensure(ctx, object.ID); err != nil {
		t.Fatal(err)
	}
	runCtx, cancel := context.WithCancel(ctx)
	if err := first.Start(runCtx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-registry.started:
	case <-time.After(10 * time.Second):
		cancel()
		t.Fatal("worker did not start")
	}
	if err := objects.Delete(ctx, object.ID, packages.ObjectPrefix); !errors.Is(err, bloby.ErrConflict) {
		t.Fatalf("running delete: %v", err)
	}
	cancel()
	if err := first.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	if got, err := service.Ensure(ctx, object.ID); err != nil || got != nil {
		t.Fatalf("interrupted work: %v %v", got, err)
	}
	if _, err := db.Exec(ctx, `UPDATE river_job SET scheduled_at=now() WHERE id=(SELECT job_id FROM munki_installer_finalizations WHERE object_id=$1)`, object.ID); err != nil {
		t.Fatal(err)
	}
	second := newJobs(objects)
	if err := second.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := second.Stop(context.Background()); err != nil {
			t.Error(err)
		}
	})
	awaitFinalization(t, func() bool {
		got, err := service.Ensure(ctx, object.ID)
		if err != nil {
			t.Fatal(err)
		}
		return got != nil && got.SizeBytesValue() == 7
	})
}
