package events

import (
	"context"
	"encoding/base64"
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

type eventCursor struct {
	Time time.Time `json:"time"`
	ID   int64     `json:"id"`
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

func (s *Store) ListEvents(ctx context.Context, params EventListParams) (EventPage, error) {
	params, err := cleanEventListParams(params)
	if err != nil {
		return EventPage{}, err
	}
	after, hasAfter, err := decodeEventCursor(params.After)
	if err != nil {
		return EventPage{}, err
	}
	query, args, err := eventListQuery(params, after, hasAfter)
	if err != nil {
		return EventPage{}, err
	}

	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return EventPage{}, err
	}
	defer rows.Close()

	items := []ExecutionEvent{}
	for rows.Next() {
		event, err := scanExecutionEvent(rows)
		if err != nil {
			return EventPage{}, err
		}
		items = append(items, event)
	}
	if err := rows.Err(); err != nil {
		return EventPage{}, err
	}

	page := EventPage{Items: items}
	if len(page.Items) > params.Limit {
		last := page.Items[params.Limit-1]
		cursorTime := last.IngestedAt
		if last.OccurredAt != nil {
			cursorTime = *last.OccurredAt
		}
		cursor, err := encodeEventCursor(eventCursor{Time: cursorTime, ID: last.ID})
		if err != nil {
			return EventPage{}, err
		}
		page.NextCursor = cursor
		page.Items = page.Items[:params.Limit]
	}
	return page, nil
}

func (s *Store) SweepEventsBefore(ctx context.Context, cutoff time.Time) (int, error) {
	tag, err := s.db.Pool().Exec(ctx, `
		DELETE FROM santa_execution_events
		WHERE COALESCE(occurred_at, ingested_at) < $1
	`, cutoff)
	return int(tag.RowsAffected()), err
}

func upsertExecutable(ctx context.Context, tx pgx.Tx, event ExecutionEventInput) (int64, error) {
	sha := strings.TrimSpace(event.FileSHA256)
	if sha == "" {
		return 0, fmt.Errorf("%w: file_sha256 is required", dbutil.ErrInvalidInput)
	}
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
	`, sha,
		strings.TrimSpace(event.FileName),
		strings.TrimSpace(event.BundleID),
		strings.TrimSpace(event.BundlePath),
		strings.TrimSpace(event.SigningID),
		strings.TrimSpace(event.TeamID),
		strings.TrimSpace(event.CDHash),
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
		strings.TrimSpace(event.FilePath),
		strings.TrimSpace(event.ExecutingUser),
		cleanEventStringList(event.LoggedInUsers),
		cleanEventStringList(event.CurrentSessions),
		event.Decision,
		eventOccurredAt(event.ExecutionTimeSeconds),
	)
	return err
}

func cleanEventStringList(values []string) []string {
	cleaned := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		cleaned = append(cleaned, value)
		seen[value] = struct{}{}
	}
	return cleaned
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
		if strings.TrimSpace(cert.SHA256) == "" {
			continue
		}
		entries = append(entries, signingChainEntry{
			SHA256:     strings.TrimSpace(cert.SHA256),
			CommonName: strings.TrimSpace(cert.CommonName),
			Org:        strings.TrimSpace(cert.Org),
			OU:         strings.TrimSpace(cert.OU),
			ValidFrom:  cert.ValidFrom,
			ValidUntil: cert.ValidUntil,
		})
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

func cleanEventListParams(params EventListParams) (EventListParams, error) {
	if params.Limit <= 0 {
		params.Limit = 100
	}
	if params.Limit > 500 {
		return EventListParams{}, fmt.Errorf("%w: limit must be at most 500", dbutil.ErrInvalidInput)
	}
	params.Decision = DecisionFilter(strings.TrimSpace(string(params.Decision)))
	params.After = strings.TrimSpace(params.After)
	return params, nil
}

func eventListQuery(params EventListParams, after eventCursor, hasAfter bool) (string, []any, error) {
	var where dbutil.WhereBuilder
	if params.HostID > 0 {
		where.Add("ee.host_id = " + where.Arg(params.HostID))
	}
	if params.Since != nil {
		where.Add("COALESCE(ee.occurred_at, ee.ingested_at) >= " + where.Arg(*params.Since))
	}
	switch params.Decision {
	case "":
	case DecisionFilterAllowed:
		where.Add("ee.decision::text LIKE 'allow_%'")
	case DecisionFilterBlocked:
		where.Add("ee.decision::text LIKE 'block_%'")
	default:
		decision := ExecutionDecision(params.Decision)
		if !validExecutionDecision(decision) {
			return "", nil, fmt.Errorf("%w: unknown decision", dbutil.ErrInvalidInput)
		}
		where.Add("ee.decision = " + where.Arg(decision))
	}
	if hasAfter {
		where.Add(
			"(COALESCE(ee.occurred_at, ee.ingested_at), ee.id) < (" + where.Arg(
				after.Time,
			) + ", " + where.Arg(
				after.ID,
			) + ")",
		)
	}
	limit := where.Arg(params.Limit + 1)
	whereSQL, args := where.Build()
	query := eventListSelectSQL + "\n" + whereSQL + `
ORDER BY COALESCE(ee.occurred_at, ee.ingested_at) DESC, ee.id DESC
LIMIT ` + limit
	return query, args, nil
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

func encodeEventCursor(cursor eventCursor) (string, error) {
	payload, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeEventCursor(value string) (eventCursor, bool, error) {
	if value == "" {
		return eventCursor{}, false, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return eventCursor{}, false, fmt.Errorf("%w: invalid after cursor", dbutil.ErrInvalidInput)
	}
	var cursor eventCursor
	if err := json.Unmarshal(payload, &cursor); err != nil {
		return eventCursor{}, false, fmt.Errorf("%w: invalid after cursor", dbutil.ErrInvalidInput)
	}
	if cursor.ID <= 0 || cursor.Time.IsZero() {
		return eventCursor{}, false, fmt.Errorf("%w: invalid after cursor", dbutil.ErrInvalidInput)
	}
	return cursor, true, nil
}

func validExecutionDecision(decision ExecutionDecision) bool {
	_, ok := validExecutionDecisions[decision]
	return ok
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
