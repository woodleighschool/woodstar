package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
)

// Store persists Santa execution and file-access events.
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

var validFileAccessDecisions = map[FileAccessDecision]struct{}{
	FileAccessDecisionUnknown:                {},
	FileAccessDecisionDenied:                 {},
	FileAccessDecisionDeniedInvalidSignature: {},
	FileAccessDecisionAuditOnly:              {},
}

type signingChainEntry struct {
	SHA256     string `json:"sha256"`
	CommonName string `json:"common_name,omitempty"`
	Org        string `json:"org,omitempty"`
	OU         string `json:"ou,omitempty"`
	ValidFrom  uint32 `json:"valid_from,omitempty"`
	ValidUntil uint32 `json:"valid_until,omitempty"`
}

// IngestEvents persists one Santa upload batch for a host.
func (s *Store) IngestEvents(
	ctx context.Context,
	hostID int64,
	executionEvents []ExecutionEventInput,
	fileAccessEvents []FileAccessEventInput,
) error {
	if len(executionEvents) == 0 && len(fileAccessEvents) == 0 {
		return nil
	}
	if err := validateEventsHaveOccurrenceTimes(executionEvents, fileAccessEvents); err != nil {
		return err
	}
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		for _, event := range executionEvents {
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
		for _, event := range fileAccessEvents {
			if err := insertFileAccessEvent(ctx, tx, hostID, event); err != nil {
				return err
			}
		}
		return nil
	})
}

func validateEventsHaveOccurrenceTimes(
	executionEvents []ExecutionEventInput,
	fileAccessEvents []FileAccessEventInput,
) error {
	for _, event := range executionEvents {
		if event.OccurredAt.IsZero() {
			return fmt.Errorf("%w: execution event occurred_at is required", dbutil.ErrInvalidInput)
		}
	}
	for _, event := range fileAccessEvents {
		if event.OccurredAt.IsZero() {
			return fmt.Errorf("%w: file access event occurred_at is required", dbutil.ErrInvalidInput)
		}
	}
	return nil
}

// ListEvents returns execution events and the total count matching params.
func (s *Store) ListEvents(ctx context.Context, params ExecutionEventListParams) ([]ExecutionEvent, int, error) {
	params.Decisions = cleanDecisionFilters(params.Decisions)
	where, args, err := executionEventWhere(params)
	if err != nil {
		return nil, 0, err
	}

	var count int
	if err := s.db.Pool().QueryRow(ctx, executionEventCountSQL+"\n"+where, args...).Scan(&count); err != nil {
		return nil, 0, err
	}

	query, args, err := executionEventListQuery(params, where, args)
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

// GetExecutionEvent returns one execution event by id.
func (s *Store) GetExecutionEvent(ctx context.Context, id int64) (*ExecutionEvent, error) {
	event, err := scanExecutionEvent(s.db.Pool().QueryRow(ctx, executionEventSelectSQL+"\nWHERE ee.id = $1", id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, dbutil.ErrNotFound
		}
		return nil, err
	}
	return &event, nil
}

// ListFileAccessEvents returns file-access events and the total count matching params.
func (s *Store) ListFileAccessEvents(
	ctx context.Context,
	params FileAccessEventListParams,
) ([]FileAccessEvent, int, error) {
	params.Decisions = cleanFileAccessDecisions(params.Decisions)
	where, args, err := fileAccessEventWhere(params)
	if err != nil {
		return nil, 0, err
	}

	var count int
	if err := s.db.Pool().QueryRow(ctx, fileAccessEventCountSQL+"\n"+where, args...).Scan(&count); err != nil {
		return nil, 0, err
	}

	query, args, err := fileAccessEventListQuery(params, where, args)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []FileAccessEvent{}
	for rows.Next() {
		event, err := scanFileAccessEvent(rows)
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

// GetFileAccessEvent returns one file-access event by id.
func (s *Store) GetFileAccessEvent(ctx context.Context, id int64) (*FileAccessEvent, error) {
	event, err := scanFileAccessEvent(s.db.Pool().QueryRow(ctx, fileAccessEventSelectSQL+"\nWHERE fae.id = $1", id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, dbutil.ErrNotFound
		}
		return nil, err
	}
	return &event, nil
}

// SweepEventsBefore deletes Santa events that occurred before cutoff.
func (s *Store) SweepEventsBefore(ctx context.Context, cutoff time.Time) (int, error) {
	var deleted int
	err := s.db.Pool().QueryRow(ctx, `
		WITH deleted_execution AS (
			DELETE FROM santa_execution_events
			WHERE occurred_at < $1
			RETURNING 1
		), deleted_file_access AS (
			DELETE FROM santa_file_access_events
			WHERE occurred_at < $1
			RETURNING 1
		)
		SELECT
			(SELECT count(*) FROM deleted_execution)::integer
			+ (SELECT count(*) FROM deleted_file_access)::integer
	`, cutoff).Scan(&deleted)
	return deleted, err
}

func upsertExecutable(ctx context.Context, tx pgx.Tx, event ExecutionEventInput) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
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
		executableEntitlements(event),
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
		cleanStringSlice(event.LoggedInUsers),
		cleanStringSlice(event.CurrentSessions),
		event.Decision,
		event.OccurredAt,
	)
	return err
}

func insertFileAccessEvent(ctx context.Context, tx pgx.Tx, hostID int64, event FileAccessEventInput) error {
	processChain, err := processChainJSON(event.ProcessChain)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO santa_file_access_events (
			host_id,
			rule_version,
			rule_name,
			target,
			decision,
			process_chain,
			occurred_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, hostID,
		event.RuleVersion,
		event.RuleName,
		event.Target,
		event.Decision,
		processChain,
		event.OccurredAt,
	)
	return err
}

func cleanStringSlice(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(strings.ReplaceAll(value, "\x00", ""))
		if value == "" || slices.Contains(out, value) {
			continue
		}
		out = append(out, value)
	}
	return out
}

func executableEntitlements(event ExecutionEventInput) []byte {
	return event.Entitlements
}

func processChainJSON(chain []ProcessInput) ([]byte, error) {
	processes := make([]Process, 0, len(chain))
	for _, process := range chain {
		processes = append(processes, Process{
			PID:          process.PID,
			FilePath:     process.FilePath,
			FileName:     fileNameFromPath(process.FilePath),
			FileSHA256:   process.FileSHA256,
			SigningID:    process.SigningID,
			TeamID:       normalizeTeamID(process.TeamID),
			CDHash:       process.CDHash,
			SigningChain: signingChainOutputEntries(signingChainEntries(process.SigningChain)),
		})
	}
	return json.Marshal(processes)
}

func fileNameFromPath(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Base(path)
}

func signingChainEntries(chain []CertificateInput) []signingChainEntry {
	entries := make([]signingChainEntry, 0, len(chain))
	for _, cert := range chain {
		entries = append(entries, signingChainEntry(cert))
	}
	return entries
}

func signingChainOutputEntries(entries []signingChainEntry) []SigningChainEntry {
	out := make([]SigningChainEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, SigningChainEntry{
			SHA256:             entry.SHA256,
			CommonName:         entry.CommonName,
			Organization:       entry.Org,
			OrganizationalUnit: entry.OU,
			ValidFrom:          certificateTime(entry.ValidFrom),
			ValidUntil:         certificateTime(entry.ValidUntil),
		})
	}
	return out
}

func certificateTime(seconds uint32) *time.Time {
	if seconds == 0 {
		return nil
	}
	t := time.Unix(int64(seconds), 0).UTC()
	return &t
}

func signingChainHash(entries []signingChainEntry) string {
	fields := make([]string, len(entries))
	for i, entry := range entries {
		fields[i] = entry.SHA256
	}
	return syncstate.PayloadHash(fields...)
}

func normalizeTeamID(value string) string {
	if value == "<unknown team id>" {
		return ""
	}
	return value
}

func executionEventWhere(params ExecutionEventListParams) (string, []any, error) {
	var where dbutil.WhereBuilder
	if params.HostID != 0 {
		where.Add("ee.host_id = " + where.Arg(params.HostID))
	}
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add(`(
			h.id::text ILIKE ` + search + `
			OR h.display_name ILIKE ` + search + `
			OR h.hostname ILIKE ` + search + `
			OR h.computer_name ILIKE ` + search + `
			OR h.hardware_serial ILIKE ` + search + `
			OR sh.machine_id ILIKE ` + search + `
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
	if err := addExecutionEventFilters(&where, params); err != nil {
		return "", nil, err
	}
	whereSQL, args := where.Build()
	return whereSQL, args, nil
}

func fileAccessEventWhere(params FileAccessEventListParams) (string, []any, error) {
	var where dbutil.WhereBuilder
	if params.HostID != 0 {
		where.Add("fae.host_id = " + where.Arg(params.HostID))
	}
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add(`(
			h.id::text ILIKE ` + search + `
			OR h.display_name ILIKE ` + search + `
			OR h.hostname ILIKE ` + search + `
			OR h.computer_name ILIKE ` + search + `
			OR h.hardware_serial ILIKE ` + search + `
			OR sh.machine_id ILIKE ` + search + `
			OR fae.rule_version ILIKE ` + search + `
			OR fae.rule_name ILIKE ` + search + `
			OR fae.target ILIKE ` + search + `
			OR fae.decision::text ILIKE ` + search + `
			OR fae.process_chain::text ILIKE ` + search + `
		)`)
	}
	if params.Since != nil {
		where.Add("fae.occurred_at >= " + where.Arg(*params.Since))
	}
	if len(params.Decisions) > 0 {
		clauses := make([]string, 0, len(params.Decisions))
		for _, decision := range params.Decisions {
			if !validFileAccessDecision(decision) {
				return "", nil, fmt.Errorf("%w: unknown file access decision", dbutil.ErrInvalidInput)
			}
			clauses = append(clauses, "fae.decision = "+where.Arg(decision))
		}
		where.Add("(" + strings.Join(clauses, " OR ") + ")")
	}
	whereSQL, args := where.Build()
	return whereSQL, args, nil
}

func addExecutionEventFilters(where *dbutil.WhereBuilder, params ExecutionEventListParams) error {
	if params.Since != nil {
		where.Add("ee.occurred_at >= " + where.Arg(*params.Since))
	}
	if len(params.Decisions) == 0 {
		return nil
	}
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
				return fmt.Errorf("%w: unknown decision", dbutil.ErrInvalidInput)
			}
			clauses = append(clauses, "ee.decision = "+where.Arg(decision))
		}
	}
	where.Add("(" + strings.Join(clauses, " OR ") + ")")
	return nil
}

func executionEventListQuery(params ExecutionEventListParams, where string, args []any) (string, []any, error) {
	params.ListParams = dbutil.CleanListParams(params.ListParams)
	if params.Sort == "" {
		params.Sort = "occurred_at.desc"
	}
	return dbutil.ListQuery{
		SelectSQL:    executionEventSelectSQL,
		WhereSQL:     where,
		Args:         args,
		OrderKeys:    eventOrderKeys("ee", "e"),
		Params:       params.ListParams,
		DefaultOrder: defaultEventOrder("ee"),
	}.Build()
}

func fileAccessEventListQuery(params FileAccessEventListParams, where string, args []any) (string, []any, error) {
	params.ListParams = dbutil.CleanListParams(params.ListParams)
	if params.Sort == "" {
		params.Sort = "occurred_at.desc"
	}
	return dbutil.ListQuery{
		SelectSQL:    fileAccessEventSelectSQL,
		WhereSQL:     where,
		Args:         args,
		OrderKeys:    fileAccessEventOrderKeys(),
		Params:       params.ListParams,
		DefaultOrder: defaultEventOrder("fae"),
	}.Build()
}

func scanExecutionEvent(row pgx.Row) (ExecutionEvent, error) {
	var event ExecutionEvent
	var entitlements []byte
	var signingChain []byte
	err := row.Scan(
		&event.ID,
		&event.Host.ID,
		&event.Host.DisplayName,
		&event.Host.Hostname,
		&event.Host.ComputerName,
		&event.Host.HardwareSerial,
		&event.Host.HardwareModel,
		&event.Host.SantaMachineID,
		&event.Host.SantaVersion,
		&event.Host.SantaClientMode,
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
		&entitlements,
		&signingChain,
	)
	if err != nil {
		return event, err
	}
	event.HostID = event.Host.ID
	if err := decodeExecutableJSON(&event.Executable, entitlements, signingChain); err != nil {
		return event, err
	}
	return event, nil
}

func scanFileAccessEvent(row pgx.Row) (FileAccessEvent, error) {
	var event FileAccessEvent
	var processChain []byte
	err := row.Scan(
		&event.ID,
		&event.Host.ID,
		&event.Host.DisplayName,
		&event.Host.Hostname,
		&event.Host.ComputerName,
		&event.Host.HardwareSerial,
		&event.Host.HardwareModel,
		&event.Host.SantaMachineID,
		&event.Host.SantaVersion,
		&event.Host.SantaClientMode,
		&event.RuleVersion,
		&event.RuleName,
		&event.Target,
		&event.Decision,
		&processChain,
		&event.OccurredAt,
		&event.IngestedAt,
	)
	if err != nil {
		return event, err
	}
	event.HostID = event.Host.ID
	if len(processChain) > 0 {
		if err := json.Unmarshal(processChain, &event.ProcessChain); err != nil {
			return event, err
		}
	}
	if len(event.ProcessChain) > 0 {
		event.PrimaryProcess = event.ProcessChain[0]
	}
	return event, nil
}

func decodeExecutableJSON(executable *Executable, entitlements []byte, signingChain []byte) error {
	if len(entitlements) > 0 {
		if err := json.Unmarshal(entitlements, &executable.Entitlements); err != nil {
			return err
		}
	}
	if len(signingChain) == 0 {
		return nil
	}
	var entries []signingChainEntry
	if err := json.Unmarshal(signingChain, &entries); err != nil {
		return err
	}
	executable.SigningChain = signingChainOutputEntries(entries)
	return nil
}

func validExecutionDecision(decision ExecutionDecision) bool {
	_, ok := validExecutionDecisions[decision]
	return ok
}

func validFileAccessDecision(decision FileAccessDecision) bool {
	_, ok := validFileAccessDecisions[decision]
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

func cleanFileAccessDecisions(decisions []FileAccessDecision) []FileAccessDecision {
	raw := make([]string, len(decisions))
	for i, decision := range decisions {
		raw[i] = string(decision)
	}
	values := dbutil.SplitListValues(raw)
	out := make([]FileAccessDecision, len(values))
	for i, value := range values {
		out[i] = FileAccessDecision(value)
	}
	return out
}

func eventOrderKeys(eventAlias string, executableAlias string) map[string]dbutil.OrderExpr {
	out := map[string]dbutil.OrderExpr{
		"occurred_at":    {SQL: eventAlias + ".occurred_at"},
		"ingested_at":    {SQL: eventAlias + ".ingested_at"},
		"decision":       {SQL: eventAlias + ".decision::text"},
		"host":           {SQL: "lower(h.display_name)"},
		"host_id":        {SQL: eventAlias + ".host_id"},
		"executing_user": {SQL: "lower(" + eventAlias + ".executing_user)"},
	}
	if executableAlias != "" {
		out["file_name"] = dbutil.OrderExpr{SQL: "lower(" + executableAlias + ".file_name)"}
	}
	return out
}

func fileAccessEventOrderKeys() map[string]dbutil.OrderExpr {
	return map[string]dbutil.OrderExpr{
		"occurred_at": {SQL: "fae.occurred_at"},
		"ingested_at": {SQL: "fae.ingested_at"},
		"decision":    {SQL: "fae.decision::text"},
		"host":        {SQL: "lower(h.display_name)"},
		"host_id":     {SQL: "fae.host_id"},
		"rule_name":   {SQL: "lower(fae.rule_name)"},
		"target":      {SQL: "lower(fae.target)"},
	}
}

func defaultEventOrder(alias string) []dbutil.OrderExpr {
	return []dbutil.OrderExpr{
		{SQL: "-extract(epoch from " + alias + ".occurred_at)"},
		{SQL: "-" + alias + ".id"},
	}
}

const hostEventSelectSQL = `
	h.id,
	h.display_name,
	h.hostname,
	h.computer_name,
	h.hardware_serial,
	h.hardware_model,
	COALESCE(sh.machine_id, ''),
	COALESCE(sh.santa_version, ''),
	COALESCE(sh.client_mode_reported::text, '')`

const executionEventSelectSQL = `
SELECT
	ee.id,
` + hostEventSelectSQL + `,
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
	e.cdhash,
	e.entitlements,
	COALESCE((
		SELECT sc.entries
		FROM santa_executable_signing_chains esc
		JOIN santa_signing_chains sc ON sc.id = esc.signing_chain_id
		WHERE esc.executable_id = e.id
		ORDER BY sc.first_seen_at DESC, sc.id DESC
		LIMIT 1
	), '[]'::jsonb)
FROM santa_execution_events ee
JOIN santa_executables e ON e.id = ee.executable_id
JOIN hosts h ON h.id = ee.host_id
LEFT JOIN santa_hosts sh ON sh.host_id = h.id`

const executionEventCountSQL = `
SELECT count(*)
FROM santa_execution_events ee
JOIN santa_executables e ON e.id = ee.executable_id
JOIN hosts h ON h.id = ee.host_id
LEFT JOIN santa_hosts sh ON sh.host_id = h.id`

const fileAccessEventSelectSQL = `
SELECT
	fae.id,
` + hostEventSelectSQL + `,
	fae.rule_version,
	fae.rule_name,
	fae.target,
	fae.decision::text,
	fae.process_chain,
	fae.occurred_at,
	fae.ingested_at
FROM santa_file_access_events fae
JOIN hosts h ON h.id = fae.host_id
LEFT JOIN santa_hosts sh ON sh.host_id = h.id`

const fileAccessEventCountSQL = `
SELECT count(*)
FROM santa_file_access_events fae
JOIN hosts h ON h.id = fae.host_id
LEFT JOIN santa_hosts sh ON sh.host_id = h.id`
