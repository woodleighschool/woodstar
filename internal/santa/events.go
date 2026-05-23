package santa

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"
	"github.com/jackc/pgx/v5"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

type ExecutionDecision string

const (
	ExecutionDecisionUnknown          ExecutionDecision = "unknown"
	ExecutionDecisionAllowUnknown     ExecutionDecision = "allow_unknown"
	ExecutionDecisionAllowBinary      ExecutionDecision = "allow_binary"
	ExecutionDecisionAllowCertificate ExecutionDecision = "allow_certificate"
	ExecutionDecisionAllowScope       ExecutionDecision = "allow_scope"
	ExecutionDecisionAllowTeamID      ExecutionDecision = "allow_teamid"
	ExecutionDecisionAllowSigningID   ExecutionDecision = "allow_signingid"
	ExecutionDecisionAllowCDHash      ExecutionDecision = "allow_cdhash"
	ExecutionDecisionBlockUnknown     ExecutionDecision = "block_unknown"
	ExecutionDecisionBlockBinary      ExecutionDecision = "block_binary"
	ExecutionDecisionBlockCertificate ExecutionDecision = "block_certificate"
	ExecutionDecisionBlockScope       ExecutionDecision = "block_scope"
	ExecutionDecisionBlockTeamID      ExecutionDecision = "block_teamid"
	ExecutionDecisionBlockSigningID   ExecutionDecision = "block_signingid"
	ExecutionDecisionBlockCDHash      ExecutionDecision = "block_cdhash"
	ExecutionDecisionBundleBinary     ExecutionDecision = "bundle_binary"

	EventDecisionClassAllowed ExecutionDecision = "allowed"
	EventDecisionClassBlocked ExecutionDecision = "blocked"
)

type EventListParams struct {
	HostID   int64
	Decision ExecutionDecision
	Since    *time.Time
	Limit    int
	After    string
}

type EventPage struct {
	Items      []ExecutionEvent `json:"items"`
	NextCursor string           `json:"next_cursor,omitempty"`
}

type ExecutionEvent struct {
	ID              int64             `json:"id"`
	HostID          int64             `json:"host_id"`
	Executable      Executable        `json:"executable"`
	FilePath        string            `json:"file_path"`
	ExecutingUser   string            `json:"executing_user"`
	LoggedInUsers   []string          `json:"logged_in_users"`
	CurrentSessions []string          `json:"current_sessions"`
	Decision        ExecutionDecision `json:"decision"`
	OccurredAt      *time.Time        `json:"occurred_at,omitempty"`
	IngestedAt      time.Time         `json:"ingested_at"`
}

type Executable struct {
	ID         int64  `json:"id"`
	SHA256     string `json:"sha256"`
	FileName   string `json:"file_name"`
	BundleID   string `json:"file_bundle_id"`
	BundlePath string `json:"file_bundle_path"`
	SigningID  string `json:"signing_id"`
	TeamID     string `json:"team_id"`
	CDHash     string `json:"cdhash"`
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

func (s *Store) IngestExecutionEvents(ctx context.Context, hostID int64, events []*syncv1.Event) error {
	if len(events) == 0 {
		return nil
	}
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		for _, event := range events {
			if event == nil {
				continue
			}
			executableID, err := upsertExecutable(ctx, tx, event)
			if err != nil {
				return err
			}
			if err := upsertSigningChain(ctx, tx, executableID, event.GetSigningChain()); err != nil {
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
	params = cleanEventListParams(params)
	after, hasAfter, err := decodeEventCursor(params.After)
	if err != nil {
		return EventPage{}, err
	}
	where, args, err := eventListWhere(params, after, hasAfter)
	if err != nil {
		return EventPage{}, err
	}
	args = append(args, params.Limit+1)
	query := eventListSelectSQL + "\n" + where + `
ORDER BY COALESCE(ee.occurred_at, ee.ingested_at) DESC, ee.id DESC
LIMIT $` + strconv.Itoa(len(args))

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
		page.NextCursor = encodeEventCursor(eventCursor{Time: cursorTime, ID: last.ID})
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

func upsertExecutable(ctx context.Context, tx pgx.Tx, event *syncv1.Event) (int64, error) {
	sha := strings.TrimSpace(event.GetFileSha256())
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
		strings.TrimSpace(event.GetFileName()),
		strings.TrimSpace(event.GetFileBundleId()),
		strings.TrimSpace(event.GetFileBundlePath()),
		strings.TrimSpace(event.GetSigningId()),
		strings.TrimSpace(event.GetTeamId()),
		strings.TrimSpace(event.GetCdhash()),
		entitlements,
	).Scan(&id)
	return id, err
}

func upsertSigningChain(ctx context.Context, tx pgx.Tx, executableID int64, chain []*syncv1.Certificate) error {
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

func insertExecutionEvent(ctx context.Context, tx pgx.Tx, hostID int64, executableID int64, event *syncv1.Event) error {
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
		strings.TrimSpace(event.GetFilePath()),
		strings.TrimSpace(event.GetExecutingUser()),
		cleanEventStringList(event.GetLoggedInUsers()),
		cleanEventStringList(event.GetCurrentSessions()),
		decisionFromProto(event.GetDecision()),
		eventOccurredAt(event.GetExecutionTime()),
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

func entitlementJSON(event *syncv1.Event) ([]byte, error) {
	entitlements := event.GetEntitlementInfo()
	if entitlements == nil {
		return []byte(`{}`), nil
	}
	payload, err := protojson.Marshal(entitlements)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func signingChainEntries(chain []*syncv1.Certificate) []signingChainEntry {
	entries := make([]signingChainEntry, 0, len(chain))
	for _, cert := range chain {
		if cert == nil || strings.TrimSpace(cert.GetSha256()) == "" {
			continue
		}
		entries = append(entries, signingChainEntry{
			SHA256:     strings.TrimSpace(cert.GetSha256()),
			CommonName: strings.TrimSpace(cert.GetCn()),
			Org:        strings.TrimSpace(cert.GetOrg()),
			OU:         strings.TrimSpace(cert.GetOu()),
			ValidFrom:  cert.GetValidFrom(),
			ValidUntil: cert.GetValidUntil(),
		})
	}
	return entries
}

func signingChainHash(entries []signingChainEntry) string {
	parts := make([]string, 0, len(entries))
	for _, entry := range entries {
		parts = append(parts, entry.SHA256)
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(sum[:])
}

func eventOccurredAt(seconds float64) *time.Time {
	if seconds <= 0 {
		return nil
	}
	whole, fraction := math.Modf(seconds)
	t := time.Unix(int64(whole), int64(fraction*1e9)).UTC()
	return &t
}

func cleanEventListParams(params EventListParams) EventListParams {
	if params.Limit <= 0 {
		params.Limit = 100
	}
	if params.Limit > 500 {
		params.Limit = 500
	}
	params.Decision = ExecutionDecision(strings.TrimSpace(string(params.Decision)))
	params.After = strings.TrimSpace(params.After)
	return params
}

func eventListWhere(params EventListParams, after eventCursor, hasAfter bool) (string, []any, error) {
	var where dbutil.WhereBuilder
	if params.HostID > 0 {
		where.Add("ee.host_id = " + where.Arg(params.HostID))
	}
	if params.Since != nil {
		where.Add("COALESCE(ee.occurred_at, ee.ingested_at) >= " + where.Arg(*params.Since))
	}
	switch params.Decision {
	case "":
	case EventDecisionClassAllowed:
		where.Add("ee.decision::text LIKE 'allow_%'")
	case EventDecisionClassBlocked:
		where.Add("ee.decision::text LIKE 'block_%'")
	default:
		if !validExecutionDecision(params.Decision) {
			return "", nil, fmt.Errorf("%w: unknown decision", dbutil.ErrInvalidInput)
		}
		where.Add("ee.decision = " + where.Arg(params.Decision))
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
	sql, args := where.Build()
	return sql, args, nil
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

func encodeEventCursor(cursor eventCursor) string {
	payload, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(payload)
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
	return slices.Contains([]ExecutionDecision{
		ExecutionDecisionUnknown,
		ExecutionDecisionAllowUnknown,
		ExecutionDecisionAllowBinary,
		ExecutionDecisionAllowCertificate,
		ExecutionDecisionAllowScope,
		ExecutionDecisionAllowTeamID,
		ExecutionDecisionAllowSigningID,
		ExecutionDecisionAllowCDHash,
		ExecutionDecisionBlockUnknown,
		ExecutionDecisionBlockBinary,
		ExecutionDecisionBlockCertificate,
		ExecutionDecisionBlockScope,
		ExecutionDecisionBlockTeamID,
		ExecutionDecisionBlockSigningID,
		ExecutionDecisionBlockCDHash,
		ExecutionDecisionBundleBinary,
	}, decision)
}

func decisionFromProto(decision syncv1.Decision) ExecutionDecision {
	switch decision {
	case syncv1.Decision_ALLOW_UNKNOWN:
		return ExecutionDecisionAllowUnknown
	case syncv1.Decision_ALLOW_BINARY:
		return ExecutionDecisionAllowBinary
	case syncv1.Decision_ALLOW_CERTIFICATE:
		return ExecutionDecisionAllowCertificate
	case syncv1.Decision_ALLOW_SCOPE:
		return ExecutionDecisionAllowScope
	case syncv1.Decision_ALLOW_TEAMID:
		return ExecutionDecisionAllowTeamID
	case syncv1.Decision_ALLOW_SIGNINGID:
		return ExecutionDecisionAllowSigningID
	case syncv1.Decision_ALLOW_CDHASH:
		return ExecutionDecisionAllowCDHash
	case syncv1.Decision_BLOCK_UNKNOWN:
		return ExecutionDecisionBlockUnknown
	case syncv1.Decision_BLOCK_BINARY:
		return ExecutionDecisionBlockBinary
	case syncv1.Decision_BLOCK_CERTIFICATE:
		return ExecutionDecisionBlockCertificate
	case syncv1.Decision_BLOCK_SCOPE:
		return ExecutionDecisionBlockScope
	case syncv1.Decision_BLOCK_TEAMID:
		return ExecutionDecisionBlockTeamID
	case syncv1.Decision_BLOCK_SIGNINGID:
		return ExecutionDecisionBlockSigningID
	case syncv1.Decision_BLOCK_CDHASH:
		return ExecutionDecisionBlockCDHash
	case syncv1.Decision_BUNDLE_BINARY:
		return ExecutionDecisionBundleBinary
	default:
		return ExecutionDecisionUnknown
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
