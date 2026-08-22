package backgroundjobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
)

const (
	QueueName       = "background"
	maxAttempts     = 3
	softStopTimeout = 10 * time.Second

	TriggerScheduled Trigger = "scheduled"
	TriggerManual    Trigger = "manual"

	ActivityIdle    Activity = "idle"
	ActivityQueued  Activity = "queued"
	ActivityRunning Activity = "running"

	OutcomeSucceeded Outcome = "succeeded"
	OutcomeFailed    Outcome = "failed"
)

type Trigger string
type Activity string
type Outcome string

type Run struct {
	ID         int64
	Trigger    Trigger
	Outcome    *Outcome
	QueuedAt   time.Time
	StartedAt  *time.Time
	FinishedAt *time.Time
	Output     json.RawMessage
}

type Status struct {
	Activity   Activity
	CurrentRun *Run
	LastRun    *Run
}

type Runtime struct {
	client *river.Client[pgx.Tx]
}

type minimumLevelHandler struct {
	next  slog.Handler
	level slog.Level
}

func (h minimumLevelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level && h.next.Enabled(ctx, level)
}

func (h minimumLevelHandler) Handle(ctx context.Context, record slog.Record) error {
	return h.next.Handle(ctx, record)
}

func (h minimumLevelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return minimumLevelHandler{next: h.next.WithAttrs(attrs), level: h.level}
}

func (h minimumLevelHandler) WithGroup(name string) slog.Handler {
	return minimumLevelHandler{next: h.next.WithGroup(name), level: h.level}
}

func New(
	pool *pgxpool.Pool,
	workers *river.Workers,
	periodicJobs []*river.PeriodicJob,
	logger *slog.Logger,
) (*Runtime, error) {
	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		// Keep River's five-second housekeeping chatter out of debug logs.
		Logger:          riverLogger(logger),
		JobTimeout:      30 * time.Minute,
		SoftStopTimeout: softStopTimeout,
		Queues: map[string]river.QueueConfig{
			QueueName: {MaxWorkers: 3},
		},
		Workers:      workers,
		PeriodicJobs: periodicJobs,
	})
	if err != nil {
		return nil, fmt.Errorf("configure River: %w", err)
	}
	return &Runtime{client: client}, nil
}

func riverLogger(logger *slog.Logger) *slog.Logger {
	return slog.New(minimumLevelHandler{
		next:  logger.Handler(),
		level: slog.LevelInfo,
	})
}

func (r *Runtime) Start(ctx context.Context) error {
	if err := r.client.Start(ctx); err != nil {
		return fmt.Errorf("start River: %w", err)
	}
	return nil
}

func (r *Runtime) Stop(ctx context.Context) error {
	if err := r.client.Stop(ctx); err != nil {
		return fmt.Errorf("stop River: %w", err)
	}
	return nil
}

func (r *Runtime) Enqueue(ctx context.Context, args river.JobArgs) error {
	_, err := r.client.Insert(ctx, args, nil)
	if err != nil {
		return fmt.Errorf("enqueue %s: %w", args.Kind(), err)
	}
	return nil
}

func (r *Runtime) Status(ctx context.Context, kind string) (Status, error) {
	active, err := r.client.JobList(ctx, river.NewJobListParams().
		Kinds(kind).
		States(
			rivertype.JobStateAvailable,
			rivertype.JobStatePending,
			rivertype.JobStateRetryable,
			rivertype.JobStateRunning,
			rivertype.JobStateScheduled,
		).
		OrderBy(river.JobListOrderByID, river.SortOrderDesc).
		First(1))
	if err != nil {
		return Status{}, fmt.Errorf("list active %s jobs: %w", kind, err)
	}

	finalized, err := r.client.JobList(ctx, river.NewJobListParams().
		Kinds(kind).
		States(
			rivertype.JobStateCancelled,
			rivertype.JobStateCompleted,
			rivertype.JobStateDiscarded,
		).
		OrderBy(river.JobListOrderByFinalizedAt, river.SortOrderDesc).
		First(1))
	if err != nil {
		return Status{}, fmt.Errorf("list finalized %s jobs: %w", kind, err)
	}

	status := Status{Activity: ActivityIdle}
	if len(active.Jobs) > 0 {
		status.CurrentRun = runFromJob(active.Jobs[0])
		if active.Jobs[0].State == rivertype.JobStateRunning {
			status.Activity = ActivityRunning
		} else {
			status.Activity = ActivityQueued
		}
	}
	if len(finalized.Jobs) > 0 {
		status.LastRun = runFromJob(finalized.Jobs[0])
	}
	return status, nil
}

func SingletonInsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: maxAttempts,
		Queue:       QueueName,
		UniqueOpts: river.UniqueOpts{ByState: []rivertype.JobState{
			rivertype.JobStateAvailable,
			rivertype.JobStatePending,
			rivertype.JobStateRetryable,
			rivertype.JobStateRunning,
			rivertype.JobStateScheduled,
		}},
	}
}

func runFromJob(job *rivertype.JobRow) *Run {
	run := &Run{
		ID:         job.ID,
		Trigger:    triggerFromArgs(job.EncodedArgs),
		QueuedAt:   job.CreatedAt,
		StartedAt:  job.AttemptedAt,
		FinishedAt: job.FinalizedAt,
		Output:     append(json.RawMessage(nil), job.Output()...),
	}
	if job.FinalizedAt != nil {
		outcome := OutcomeFailed
		if job.State == rivertype.JobStateCompleted {
			outcome = OutcomeSucceeded
		}
		run.Outcome = &outcome
	}
	return run
}

func triggerFromArgs(args []byte) Trigger {
	var value struct {
		Trigger Trigger `json:"trigger"`
	}
	if json.Unmarshal(args, &value) == nil && value.Trigger != "" {
		return value.Trigger
	}
	return TriggerScheduled
}
