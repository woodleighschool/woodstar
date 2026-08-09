package entra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/riverqueue/river"

	"github.com/woodleighschool/woodstar/internal/backgroundjobs"
)

const (
	SyncJobKind        = "entra_sync"
	syncLockRetryDelay = 10 * time.Second

	// SyncAdvisoryLockID identifies the session lock held across Graph fetch and apply.
	SyncAdvisoryLockID int64 = 7146808627076917100
)

var ErrSyncDisabled = errors.New("entra sync is disabled")

// SyncJobArgs identifies one scheduled or manually requested Entra sync.
type SyncJobArgs struct {
	Trigger backgroundjobs.Trigger `json:"trigger"`
}

func (SyncJobArgs) Kind() string { return SyncJobKind }

func (SyncJobArgs) InsertOpts() river.InsertOpts { return backgroundjobs.SingletonInsertOpts() }

// SyncLocker serializes a full fetch-and-apply pass between replicas.
type SyncLocker interface {
	Try(context.Context, func(context.Context) error) (bool, error)
}

// SyncWorker executes durable Entra sync jobs.
type SyncWorker struct {
	river.WorkerDefaults[SyncJobArgs]

	service *Service
	locker  SyncLocker
}

func NewSyncWorker(service *Service, locker SyncLocker) *SyncWorker {
	return &SyncWorker{service: service, locker: locker}
}

func (w *SyncWorker) Work(ctx context.Context, _ *river.Job[SyncJobArgs]) error {
	output, err := w.run(ctx)
	return errors.Join(err, river.RecordOutput(ctx, output))
}

func (w *SyncWorker) run(ctx context.Context) (SyncJobOutput, error) {
	var result SyncResult
	acquired, err := w.locker.Try(ctx, func(ctx context.Context) error {
		var syncErr error
		result, syncErr = w.service.Sync(ctx)
		return syncErr
	})
	if !acquired && err == nil {
		return SyncJobOutput{}, river.JobSnooze(syncLockRetryDelay)
	}
	if err != nil {
		return SyncJobOutput{Error: err.Error()}, err
	}
	return SyncJobOutput{SyncResult: result}, nil
}

// SyncJobOutput is retained on the River job row for API status reporting.
type SyncJobOutput struct {
	SyncResult

	Error string `json:"error,omitempty"`
}

// SyncStatus is the directory-facing state of the Entra job.
type SyncStatus struct {
	Enabled    bool                    `json:"enabled"`
	Activity   backgroundjobs.Activity `json:"activity" enum:"idle,queued,running"`
	CurrentRun *SyncRun                `json:"current_run,omitempty"`
	LastRun    *SyncRun                `json:"last_run,omitempty"`
}

// SyncRun describes one current or finalized Entra sync.
type SyncRun struct {
	ID         int64                   `json:"id"`
	Trigger    backgroundjobs.Trigger  `json:"trigger" enum:"scheduled,manual"`
	Outcome    *backgroundjobs.Outcome `json:"outcome,omitempty" enum:"succeeded,failed"`
	QueuedAt   time.Time               `json:"queued_at"`
	StartedAt  *time.Time              `json:"started_at,omitempty"`
	FinishedAt *time.Time              `json:"finished_at,omitempty"`
	Users      int                     `json:"users,omitempty"`
	Groups     int                     `json:"groups,omitempty"`
	DurationMS int                     `json:"duration_ms,omitempty"`
	Error      string                  `json:"error,omitempty"`
}

type jobRuntime interface {
	Enqueue(context.Context, river.JobArgs) error
	Status(context.Context, string) (backgroundjobs.Status, error)
}

// SyncJobs exposes Entra job state and manual enqueueing to the API.
type SyncJobs struct {
	enabled bool
	runtime jobRuntime
}

func NewSyncJobs(enabled bool, runtime jobRuntime) *SyncJobs {
	return &SyncJobs{enabled: enabled, runtime: runtime}
}

func (s *SyncJobs) Status(ctx context.Context) (SyncStatus, error) {
	if !s.enabled {
		return SyncStatus{Enabled: false, Activity: backgroundjobs.ActivityIdle}, nil
	}
	status, err := s.runtime.Status(ctx, SyncJobKind)
	if err != nil {
		return SyncStatus{}, fmt.Errorf("get Entra sync status: %w", err)
	}
	return syncStatus(status), nil
}

func (s *SyncJobs) Trigger(ctx context.Context) (SyncStatus, error) {
	if !s.enabled {
		return SyncStatus{}, ErrSyncDisabled
	}
	if err := s.runtime.Enqueue(ctx, SyncJobArgs{Trigger: backgroundjobs.TriggerManual}); err != nil {
		return SyncStatus{}, fmt.Errorf("trigger Entra sync: %w", err)
	}
	return s.Status(ctx)
}

func syncStatus(status backgroundjobs.Status) SyncStatus {
	return SyncStatus{
		Enabled:    true,
		Activity:   status.Activity,
		CurrentRun: syncRun(status.CurrentRun),
		LastRun:    syncRun(status.LastRun),
	}
}

func syncRun(run *backgroundjobs.Run) *SyncRun {
	if run == nil {
		return nil
	}
	value := &SyncRun{
		ID:         run.ID,
		Trigger:    run.Trigger,
		Outcome:    run.Outcome,
		QueuedAt:   run.QueuedAt,
		StartedAt:  run.StartedAt,
		FinishedAt: run.FinishedAt,
	}
	var output SyncJobOutput
	if json.Unmarshal(run.Output, &output) == nil {
		value.Users = output.Users
		value.Groups = output.Groups
		value.DurationMS = output.DurationMS
		value.Error = output.Error
	}
	return value
}
