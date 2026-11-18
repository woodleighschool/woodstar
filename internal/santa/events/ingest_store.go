package events

import (
	"context"
	"maps"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
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
	if err := reconcileBundleCompletion(ctx, tx, slices.Collect(maps.Values(ids.bundles))); err != nil {
		return nil, err
	}
	if err := insertPlanOccurrences(ctx, tx, hostID, plan, ids); err != nil {
		return nil, err
	}
	return incompleteBundleHashes(ctx, tx, plan.bundleRequestHashes)
}

type ingestReferenceIDs struct {
	newSigningChains []signingChainWrite
	executables      map[string]int64
	signingChains    map[string]int64
	certificates     map[string]int64
	bundles          map[string]int64
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
	var batch pgx.Batch
	for _, write := range plan.executables {
		batch.Queue(upsertExecutableSQL, pgx.StructArgs(write))
		ids.executables[write.SHA256] = 0
	}
	for _, chain := range plan.signingChains {
		batch.Queue(`
INSERT INTO santa_signing_chains (sha256) VALUES ($1) ON CONFLICT DO NOTHING`, chain.SHA256).Exec(
			func(tag pgconn.CommandTag) error {
				if tag.RowsAffected() == 1 {
					ids.newSigningChains = append(ids.newSigningChains, chain)
				}
				return nil
			})
		ids.signingChains[chain.SHA256] = 0
	}
	for _, entry := range plan.certificates {
		batch.Queue(upsertCertificateSQL, entry.certificateArgs())
		ids.certificates[entry.SHA256] = 0
	}
	for _, write := range plan.bundles {
		batch.Queue(upsertBundleSQL, pgx.StructArgs(write))
		ids.bundles[write.SHA256] = 0
	}

	// A separate SELECT sees rows inserted by concurrent uploads after an upsert
	// waited on their unique constraint. RETURNING omits unchanged conflicts.
	queueReferenceIDs(&batch, "santa_executables", ids.executables)
	queueReferenceIDs(&batch, "santa_signing_chains", ids.signingChains)
	queueReferenceIDs(&batch, "santa_certificates", ids.certificates)
	queueReferenceIDs(&batch, "santa_bundles", ids.bundles)
	err := tx.SendBatch(ctx, &batch).Close()
	return ids, err
}

func queueReferenceIDs(batch *pgx.Batch, table string, ids map[string]int64) {
	if len(ids) == 0 {
		return
	}
	query := "SELECT sha256, id FROM " + pgx.Identifier{table}.Sanitize() + " WHERE sha256 = ANY($1::text[])"
	batch.Queue(query, slices.Collect(maps.Keys(ids))).Query(func(rows pgx.Rows) error {
		var hash string
		var id int64
		_, err := pgx.ForEachRow(rows, []any{&hash, &id}, func() error {
			ids[hash] = id
			return nil
		})
		return err
	})
}

func insertPlanRelationships(
	ctx context.Context,
	tx pgx.Tx,
	plan ingestPlan,
	ids ingestReferenceIDs,
) error {
	// The plan sorts each relationship class by its natural composite key.
	// Keep the class order stable for the same reason as reference writes.
	for _, chain := range ids.newSigningChains {
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
