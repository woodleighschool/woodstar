package events

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
)

// Store persists Santa execution events and executable metadata.
type Store struct {
	db *database.DB
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db}
}

var validExecutionDecisions = map[ExecutionDecision]struct{}{
	ExecutionDecisionUnknown:          {},
	ExecutionDecisionAllowUnknown:     {},
	ExecutionDecisionAllowBinary:      {},
	ExecutionDecisionAllowCertificate: {},
	ExecutionDecisionAllowScope:       {},
	ExecutionDecisionAllowTeamID:      {},
	ExecutionDecisionAllowSigningID:   {},
	ExecutionDecisionAllowCDHash:      {},
	ExecutionDecisionBlockUnknown:     {},
	ExecutionDecisionBlockBinary:      {},
	ExecutionDecisionBlockCertificate: {},
	ExecutionDecisionBlockScope:       {},
	ExecutionDecisionBlockTeamID:      {},
	ExecutionDecisionBlockSigningID:   {},
	ExecutionDecisionBlockCDHash:      {},
	ExecutionDecisionBundleBinary:     {},
}

type signingChainEntry struct {
	SHA256     string `json:"sha256"`
	CommonName string `json:"common_name,omitempty"`
	Org        string `json:"org,omitempty"`
	OU         string `json:"ou,omitempty"`
	ValidFrom  uint32 `json:"valid_from,omitempty"`
	ValidUntil uint32 `json:"valid_until,omitempty"`
}

func (s *Store) IngestExecutionEvents(ctx context.Context, hostID int64, events []ExecutionEventInput) error {
	if len(events) == 0 {
		return nil
	}
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		for _, event := range events {
			executableID, err := upsertExecutable(ctx, tx, event)
			if err != nil {
				return err
			}
			if err := upsertSigningChain(ctx, tx, executableID, event.SigningChain); err != nil {
				return err
			}
			if err := insertExecutionEvent(ctx, tx, hostID, executableID, event); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) ListEvents(ctx context.Context, params EventListParams) ([]ExecutionEvent, int, error) {
	params.Decisions = cleanDecisionFilters(params.Decisions)
	where, args, err := eventListWhere(params)
	if err != nil {
		return nil, 0, err
	}

	var count int
	if err := s.db.Pool().QueryRow(ctx, eventListCountSQL+"\n"+where, args...).Scan(&count); err != nil {
		return nil, 0, err
	}

	query, args, err := eventListQuery(params, where, args)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []ExecutionEvent{}
	for rows.Next() {
		event, err := scanExecutionEvent(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, event)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, count, nil
}

func (s *Store) SweepEventsBefore(ctx context.Context, cutoff time.Time) (int, error) {
	tag, err := s.db.Pool().Exec(ctx, `
		DELETE FROM santa_execution_events
		WHERE COALESCE(occurred_at, ingested_at) < $1
	`, cutoff)
	return int(tag.RowsAffected()), err
}

func upsertExecutable(ctx context.Context, tx pgx.Tx, event ExecutionEventInput) (int64, error) {
	entitlements, err := entitlementJSON(event)
	if err != nil {
		return 0, err
	}
	var id int64
	err = tx.QueryRow(ctx, `
		INSERT INTO santa_executables (
			sha256,
			file_name,
			file_bundle_id,
			file_bundle_path,
			signing_id,
			team_id,
			cdhash,
			entitlements,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
		ON CONFLICT (sha256) DO UPDATE SET
			file_name = EXCLUDED.file_name,
			file_bundle_id = EXCLUDED.file_bundle_id,
			file_bundle_path = EXCLUDED.file_bundle_path,
			signing_id = EXCLUDED.signing_id,
			team_id = EXCLUDED.team_id,
			cdhash = EXCLUDED.cdhash,
			entitlements = EXCLUDED.entitlements,
			updated_at = now()
		RETURNING id
	`, event.FileSHA256,
		event.FileName,
		event.BundleID,
		event.BundlePath,
		event.SigningID,
		event.TeamID,
		event.CDHash,
		entitlements,
	).Scan(&id)
	return id, err
}

func upsertSigningChain(ctx context.Context, tx pgx.Tx, executableID int64, chain []CertificateInput) error {
	entries := signingChainEntries(chain)
	if len(entries) == 0 {
		return nil
	}
	payload, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	var chainID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO santa_signing_chains (sha256, entries)
		VALUES ($1, $2)
		ON CONFLICT (sha256) DO UPDATE SET entries = EXCLUDED.entries
		RETURNING id
	`, signingChainHash(entries), payload).Scan(&chainID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO santa_executable_signing_chains (executable_id, signing_chain_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, executableID, chainID)
	return err
}

func insertExecutionEvent(
	ctx context.Context,
	tx pgx.Tx,
	hostID int64,
	executableID int64,
	event ExecutionEventInput,
) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO santa_execution_events (
			host_id,
			executable_id,
			file_path,
			executing_user,
			logged_in_users,
			current_sessions,
			decision,
			occurred_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, hostID,
		executableID,
		event.FilePath,
		event.ExecutingUser,
		event.LoggedInUsers,
		event.CurrentSessions,
		event.Decision,
		eventOccurredAt(event.ExecutionTimeSeconds),
	)
	return err
}

func entitlementJSON(event ExecutionEventInput) ([]byte, error) {
	if len(event.Entitlements) == 0 {
		return nil, nil
	}
	return event.Entitlements, nil
}

func signingChainEntries(chain []CertificateInput) []signingChainEntry {
	entries := make([]signingChainEntry, 0, len(chain))
	for _, cert := range chain {
		entries = append(entries, signingChainEntry(cert))
	}
	return entries
}

func signingChainHash(entries []signingChainEntry) string {
	fields := make([]string, len(entries))
	for i, entry := range entries {
		fields[i] = entry.SHA256
	}
	return syncstate.PayloadHash(fields...)
}

func eventOccurredAt(seconds float64) *time.Time {
	if seconds <= 0 {
		return nil
	}
	whole, fraction := math.Modf(seconds)
	t := time.Unix(int64(whole), int64(fraction*1e9)).UTC()
	return &t
}

func eventListWhere(params EventListParams) (string, []any, error) {
	var where dbutil.WhereBuilder
	if params.HostID > 0 {
		where.Add("ee.host_id = " + where.Arg(params.HostID))
	}
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add(`(
			ee.host_id::text ILIKE ` + search + `
			OR ee.file_path ILIKE ` + search + `
			OR ee.executing_user ILIKE ` + search + `
			OR ee.decision::text ILIKE ` + search + `
			OR ee.logged_in_users::text ILIKE ` + search + `
			OR ee.current_sessions::text ILIKE ` + search + `
			OR e.sha256 ILIKE ` + search + `
			OR e.file_name ILIKE ` + search + `
			OR e.file_bundle_id ILIKE ` + search + `
			OR e.file_bundle_path ILIKE ` + search + `
			OR e.signing_id ILIKE ` + search + `
			OR e.team_id ILIKE ` + search + `
			OR e.cdhash ILIKE ` + search + `
		)`)
	}
	if params.Since != nil {
		where.Add("COALESCE(ee.occurred_at, ee.ingested_at) >= " + where.Arg(*params.Since))
	}
	if len(params.Decisions) > 0 {
		clauses := make([]string, 0, len(params.Decisions))
		for _, filter := range params.Decisions {
			switch filter {
			case DecisionFilterAllowed:
				clauses = append(clauses, "ee.decision::text LIKE 'allow_%'")
			case DecisionFilterBlocked:
				clauses = append(clauses, "ee.decision::text LIKE 'block_%'")
			default:
				decision := ExecutionDecision(filter)
				if !validExecutionDecision(decision) {
					return "", nil, fmt.Errorf("%w: unknown decision", dbutil.ErrInvalidInput)
				}
				clauses = append(clauses, "ee.decision = "+where.Arg(decision))
			}
		}
		where.Add("(" + strings.Join(clauses, " OR ") + ")")
	}
	whereSQL, args := where.Build()
	return whereSQL, args, nil
}

func eventListQuery(params EventListParams, where string, args []any) (string, []any, error) {
	params.ListParams = dbutil.CleanListParams(params.ListParams)
	if params.Sort == "" {
		params.Sort = "occurred_at.desc"
	}
	return dbutil.ListQuery{
		SelectSQL: eventListSelectSQL,
		WhereSQL:  where,
		Args:      args,
		OrderKeys: eventOrderKeys(),
		Params:    params.ListParams,
		DefaultOrder: []dbutil.OrderExpr{
			{SQL: "-extract(epoch from COALESCE(ee.occurred_at, ee.ingested_at))"},
			{SQL: "-ee.id"},
		},
	}.Build()
}

func scanExecutionEvent(row pgx.Row) (ExecutionEvent, error) {
	var event ExecutionEvent
	err := row.Scan(
		&event.ID,
		&event.HostID,
		&event.FilePath,
		&event.ExecutingUser,
		&event.LoggedInUsers,
		&event.CurrentSessions,
		&event.Decision,
		&event.OccurredAt,
		&event.IngestedAt,
		&event.Executable.ID,
		&event.Executable.SHA256,
		&event.Executable.FileName,
		&event.Executable.BundleID,
		&event.Executable.BundlePath,
		&event.Executable.SigningID,
		&event.Executable.TeamID,
		&event.Executable.CDHash,
	)
	return event, err
}

func validExecutionDecision(decision ExecutionDecision) bool {
	_, ok := validExecutionDecisions[decision]
	return ok
}

func cleanDecisionFilters(filters []DecisionFilter) []DecisionFilter {
	raw := make([]string, len(filters))
	for i, filter := range filters {
		raw[i] = string(filter)
	}
	values := dbutil.SplitListValues(raw)
	out := make([]DecisionFilter, len(values))
	for i, value := range values {
		out[i] = DecisionFilter(value)
	}
	return out
}

func eventOrderKeys() map[string]dbutil.OrderExpr {
	return map[string]dbutil.OrderExpr{
		"occurred_at":    {SQL: "COALESCE(ee.occurred_at, ee.ingested_at)"},
		"ingested_at":    {SQL: "ee.ingested_at"},
		"decision":       {SQL: "ee.decision::text"},
		"host_id":        {SQL: "ee.host_id"},
		"file_name":      {SQL: "lower(e.file_name)"},
		"executing_user": {SQL: "lower(ee.executing_user)"},
	}
}

const eventListSelectSQL = `
SELECT
	ee.id,
	ee.host_id,
	ee.file_path,
	ee.executing_user,
	ee.logged_in_users,
	ee.current_sessions,
	ee.decision::text,
	ee.occurred_at,
	ee.ingested_at,
	e.id,
	e.sha256,
	e.file_name,
	e.file_bundle_id,
	e.file_bundle_path,
	e.signing_id,
	e.team_id,
	e.cdhash
FROM santa_execution_events ee
JOIN santa_executables e ON e.id = ee.executable_id`

const eventListCountSQL = `
SELECT count(*)
FROM santa_execution_events ee
JOIN santa_executables e ON e.id = ee.executable_id`
