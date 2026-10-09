package packages

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/woodleighschool/goodies/bloby"
	"github.com/woodleighschool/woodstar/internal/backgroundjobs"
)

// ErrFinalizationFailed is a terminal verification failure. A new upload is required.
var ErrFinalizationFailed = errors.New("installer verification failed; upload the installer again")

type finalizationQueue interface {
	EnqueueTx(context.Context, pgx.Tx, river.JobArgs) (int64, error)
}

// Finalizations owns durable installer verification and its object reference.
type Finalizations struct {
	pool    *pgxpool.Pool
	objects *bloby.Service
	jobs    finalizationQueue
}

func NewFinalizations(pool *pgxpool.Pool, objects *bloby.Service, jobs finalizationQueue) *Finalizations {
	return &Finalizations{pool: pool, objects: objects, jobs: jobs}
}

// Ensure returns verified content, or nil while verification is pending.
// Failed work remains failed; polling never schedules a replacement job.
func (s *Finalizations) Ensure(ctx context.Context, id int64) (*bloby.Object, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	var available bool
	var multipart *string
	err = tx.QueryRow(ctx, `SELECT available_at IS NOT NULL, multipart_upload_id
 FROM storage_objects WHERE id = $1 AND prefix = $2 AND expired_at IS NULL FOR UPDATE`, id, ObjectPrefix).Scan(&available, &multipart)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, bloby.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if available {
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		return s.objects.GetByID(ctx, id)
	}
	if multipart != nil {
		return nil, fmt.Errorf("%w: multipart upload must be completed before verification", bloby.ErrInvalidInput)
	}
	var state string
	err = tx.QueryRow(ctx, `SELECT state FROM munki_installer_finalizations WHERE object_id = $1`, id).Scan(&state)
	switch {
	case err == nil:
		if state != "pending" {
			return nil, ErrFinalizationFailed
		}
	case errors.Is(err, pgx.ErrNoRows):
		// Refresh under the row lock so an expiry claim racing this transaction
		// rechecks the new timestamp before it can claim the pending object.
		if _, err := tx.Exec(ctx, `UPDATE storage_objects SET updated_at = now() WHERE id = $1`, id); err != nil {
			return nil, err
		}
		jobID, err := s.jobs.EnqueueTx(ctx, tx, FinalizeInstallerArgs{ObjectID: id})
		if err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO munki_installer_finalizations (object_id, job_id) VALUES ($1, $2)`, id, jobID); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO munki_installer_finalization_claims (object_id, job_id) VALUES ($1, $2)`, id, jobID); err != nil {
			return nil, err
		}
	default:
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return nil, nil
}

// FinalizeInstallerArgs identifies the object retained by one verification job.
type FinalizeInstallerArgs struct {
	ObjectID int64 `json:"object_id"`
}

func (FinalizeInstallerArgs) Kind() string { return "munki_finalize_installer" }
func (FinalizeInstallerArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: backgroundjobs.InstallerQueueName, MaxAttempts: 3, UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

// FinalizeInstallerWorker verifies immutable bytes outside an HTTP request.
type FinalizeInstallerWorker struct {
	river.WorkerDefaults[FinalizeInstallerArgs]

	objects *bloby.Service
}

func NewFinalizeInstallerWorker(objects *bloby.Service) *FinalizeInstallerWorker {
	return &FinalizeInstallerWorker{objects: objects}
}

func (*FinalizeInstallerWorker) Timeout(*river.Job[FinalizeInstallerArgs]) time.Duration {
	return time.Hour
}
func (w *FinalizeInstallerWorker) Work(ctx context.Context, job *river.Job[FinalizeInstallerArgs]) error {
	_, err := w.objects.Finalize(ctx, job.Args.ObjectID, ObjectPrefix)
	return err
}
