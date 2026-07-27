package events

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// IngestEvents persists one Santa upload atomically. It plans shared writes
// before opening the transaction so overlapping uploads use one lock order.
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
	plan := newIngestPlan(executionEvents, fileAccessEvents, standaloneRuleCreationEvents)
	var bundleBinaryRequests []string
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		requests, err := ingestEventsTx(ctx, tx, hostID, plan)
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
	plan ingestPlan,
) ([]string, error) {
	// Every transaction crosses contended resource classes in the same
	// direction: references, relationships, then bundle refreshes.
	ids, err := upsertPlanReferences(ctx, tx, plan)
	if err != nil {
		return nil, err
	}
	if err := insertPlanRelationships(ctx, tx, plan, ids); err != nil {
		return nil, err
	}
	// Bundle completion is derived from the folded metadata and complete link
	// set, so client order cannot mark a partially collected bundle complete.
	if err := reconcilePlanBundleCompletion(ctx, tx, plan, ids); err != nil {
		return nil, err
	}
	if err := insertPlanOccurrences(ctx, tx, hostID, plan, ids); err != nil {
		return nil, err
	}
	return incompleteBundleHashes(ctx, tx, plan.bundleRequestHashes)
}

type ingestReferenceIDs struct {
	executables   map[string]int64
	signingChains map[string]int64
	certificates  map[string]int64
	bundles       map[string]int64
}

func upsertPlanReferences(ctx context.Context, tx pgx.Tx, plan ingestPlan) (ingestReferenceIDs, error) {
	// Keep reference classes in this order. A transaction can hold rows from an
	// earlier class while waiting on a later one, so class order is part of the
	// deadlock-prevention contract.
	ids := ingestReferenceIDs{
		executables:   make(map[string]int64, len(plan.executables)),
		signingChains: make(map[string]int64, len(plan.signingChains)),
		certificates:  make(map[string]int64, len(plan.certificates)),
		bundles:       make(map[string]int64, len(plan.bundles)),
	}
	for _, write := range plan.executables {
		id, err := upsertExecutable(ctx, tx, write)
		if err != nil {
			return ingestReferenceIDs{}, err
		}
		ids.executables[write.SHA256] = id
	}
	for _, write := range plan.signingChains {
		id, err := upsertSigningChain(ctx, tx, write)
		if err != nil {
			return ingestReferenceIDs{}, err
		}
		ids.signingChains[write.SHA256] = id
	}
	for _, write := range plan.certificates {
		id, err := upsertCertificate(ctx, tx, write)
		if err != nil {
			return ingestReferenceIDs{}, err
		}
		ids.certificates[write.SHA256] = id
	}
	for _, write := range plan.bundles {
		id, err := upsertBundle(ctx, tx, write)
		if err != nil {
			return ingestReferenceIDs{}, err
		}
		ids.bundles[write.SHA256] = id
	}
	return ids, nil
}

func insertPlanRelationships(
	ctx context.Context,
	tx pgx.Tx,
	plan ingestPlan,
	ids ingestReferenceIDs,
) error {
	// The plan sorts each relationship class by its natural composite key.
	// Keep the class order stable for the same reason as reference writes.
	for _, chain := range plan.signingChains {
		for position, entry := range chain.Entries {
			if err := linkSigningChainEntry(
				ctx,
				tx,
				ids.signingChains[chain.SHA256],
				position,
				ids.certificates[entry.SHA256],
			); err != nil {
				return err
			}
		}
	}
	for _, link := range plan.executableSigningLinks {
		if err := linkExecutableSigningChain(
			ctx,
			tx,
			ids.executables[link.executableSHA256],
			ids.signingChains[link.signingChainSHA256],
		); err != nil {
			return err
		}
	}
	for _, link := range plan.bundleExecutableLinks {
		if err := linkBundleExecutable(
			ctx,
			tx,
			ids.bundles[link.bundleSHA256],
			ids.executables[link.executableSHA256],
		); err != nil {
			return err
		}
	}
	return nil
}

func reconcilePlanBundleCompletion(
	ctx context.Context,
	tx pgx.Tx,
	plan ingestPlan,
	ids ingestReferenceIDs,
) error {
	for _, bundle := range plan.bundles {
		if err := reconcileBundleCompletion(ctx, tx, ids.bundles[bundle.SHA256]); err != nil {
			return err
		}
	}
	return nil
}

func insertPlanOccurrences(
	ctx context.Context,
	tx pgx.Tx,
	hostID int64,
	plan ingestPlan,
	ids ingestReferenceIDs,
) error {
	for _, event := range plan.executionEvents {
		if err := insertExecutionEvent(ctx, tx, hostID, ids.executables[event.FileSHA256], event); err != nil {
			return err
		}
	}
	for _, event := range plan.fileAccessEvents {
		if err := insertFileAccessEvent(ctx, tx, hostID, event); err != nil {
			return err
		}
	}
	for _, event := range plan.standaloneEvents {
		if err := insertStandaloneRuleCreationEvent(ctx, tx, hostID, event); err != nil {
			return err
		}
	}
	return nil
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
