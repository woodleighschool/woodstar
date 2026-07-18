package events

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// IngestEvents persists one Santa upload batch for a host.
func (s *Store) IngestEvents(
	ctx context.Context,
	hostID int64,
	executionEvents []ExecutionEventInput,
	fileAccessEvents []FileAccessEventInput,
	standaloneRuleCreationEvents []StandaloneRuleCreationEventInput,
) ([]string, error) {
	if len(executionEvents) == 0 && len(fileAccessEvents) == 0 && len(standaloneRuleCreationEvents) == 0 {
		return nil, nil
	}
	if err := validateEventInputs(executionEvents, fileAccessEvents, standaloneRuleCreationEvents); err != nil {
		return nil, err
	}
	var bundleBinaryRequests []string
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		requests, err := ingestEventsTx(
			ctx,
			tx,
			hostID,
			executionEvents,
			fileAccessEvents,
			standaloneRuleCreationEvents,
		)
		if err != nil {
			return err
		}
		bundleBinaryRequests = requests
		return nil
	})
	return bundleBinaryRequests, err
}

func ingestEventsTx(
	ctx context.Context,
	tx pgx.Tx,
	hostID int64,
	executionEvents []ExecutionEventInput,
	fileAccessEvents []FileAccessEventInput,
	standaloneRuleCreationEvents []StandaloneRuleCreationEventInput,
) ([]string, error) {
	bundleRequestCandidates := []string{}
	for _, event := range executionEvents {
		candidate, err := processExecutionEvent(ctx, tx, hostID, event)
		if err != nil {
			return nil, err
		}
		if candidate != "" {
			bundleRequestCandidates = append(bundleRequestCandidates, candidate)
		}
	}
	for _, event := range fileAccessEvents {
		if err := insertFileAccessEvent(ctx, tx, hostID, event); err != nil {
			return nil, err
		}
	}
	for _, event := range standaloneRuleCreationEvents {
		if err := insertStandaloneRuleCreationEvent(ctx, tx, hostID, event); err != nil {
			return nil, err
		}
	}
	return incompleteBundleHashes(ctx, tx, bundleRequestCandidates)
}

// processExecutionEvent persists one execution event and returns the bundle
// hash to request a binary listing for, or "" when none is needed.
func processExecutionEvent(
	ctx context.Context,
	tx pgx.Tx,
	hostID int64,
	event ExecutionEventInput,
) (string, error) {
	executableID, err := upsertExecutable(ctx, tx, event)
	if err != nil {
		return "", err
	}
	if err := upsertSigningChain(ctx, tx, executableID, event.SigningChain); err != nil {
		return "", err
	}
	bundleRequest, err := processEventBundle(ctx, tx, executableID, event)
	if err != nil {
		return "", err
	}
	if event.Decision == ExecutionDecisionBundleBinary {
		return bundleRequest, nil
	}
	if err := insertExecutionEvent(ctx, tx, hostID, executableID, event); err != nil {
		return "", err
	}
	return bundleRequest, nil
}

func insertExecutionEvent(
	ctx context.Context,
	tx pgx.Tx,
	hostID int64,
	executableID int64,
	event ExecutionEventInput,
) error {
	write := executionEventWrite{
		HostID:          hostID,
		ExecutableID:    executableID,
		FilePath:        event.FilePath,
		ExecutingUser:   event.ExecutingUser,
		PID:             event.PID,
		PPID:            event.PPID,
		ParentName:      event.ParentName,
		LoggedInUsers:   normalizeStringSlice(event.LoggedInUsers),
		CurrentSessions: normalizeStringSlice(event.CurrentSessions),
		Decision:        string(event.Decision),
		StaticRule:      event.StaticRule,
		OccurredAt:      event.OccurredAt,
	}
	_, err := tx.Exec(ctx, `
INSERT INTO santa_execution_events (
	host_id,
	executable_id,
	file_path,
	executing_user,
	pid,
	ppid,
	parent_name,
	logged_in_users,
	current_sessions,
	decision,
	static_rule,
	occurred_at
)
VALUES (
	@host_id,
	@executable_id,
	@file_path,
	@executing_user,
	@pid,
	@ppid,
	@parent_name,
	@logged_in_users,
	@current_sessions,
	@decision::santa_execution_decision,
	@static_rule,
	@occurred_at
)`, pgx.StructArgs(write))
	return err
}

func insertStandaloneRuleCreationEvent(
	ctx context.Context,
	tx pgx.Tx,
	hostID int64,
	event StandaloneRuleCreationEventInput,
) error {
	_, err := tx.Exec(ctx, `
INSERT INTO santa_standalone_rule_creation_events (
	host_id,
	identifier,
	decision,
	occurred_at
)
VALUES ($1, $2, $3::santa_execution_decision, $4)`,
		hostID,
		event.Identifier,
		event.Decision,
		event.OccurredAt,
	)
	return err
}

type executionEventWrite struct {
	HostID          int64     `db:"host_id"`
	ExecutableID    int64     `db:"executable_id"`
	FilePath        string    `db:"file_path"`
	ExecutingUser   string    `db:"executing_user"`
	PID             int32     `db:"pid"`
	PPID            int32     `db:"ppid"`
	ParentName      string    `db:"parent_name"`
	LoggedInUsers   []string  `db:"logged_in_users"`
	CurrentSessions []string  `db:"current_sessions"`
	Decision        string    `db:"decision"`
	StaticRule      bool      `db:"static_rule"`
	OccurredAt      time.Time `db:"occurred_at"`
}
