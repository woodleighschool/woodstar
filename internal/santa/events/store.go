package events

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	"github.com/woodleighschool/woodstar/internal/santa/payloadhash"
)

// Store persists Santa execution and file-access events.
type Store struct {
	db *database.DB
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db}
}

var validExecutionDecisions = valueSet(ExecutionDecisionValues)

var validFileAccessDecisions = valueSet(FileAccessDecisionValues)

func valueSet[T comparable](values []T) map[T]struct{} {
	set := make(map[T]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
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

// processEventBundle upserts and links the event's bundle when present and
// returns the bundle hash to request a binary listing for, or "" when none.
func processEventBundle(
	ctx context.Context,
	tx pgx.Tx,
	executableID int64,
	event ExecutionEventInput,
) (string, error) {
	bundleID, hasBundle, err := upsertBundle(ctx, tx, event)
	if err != nil {
		return "", err
	}
	if !hasBundle {
		return "", nil
	}
	if err := linkBundleExecutable(ctx, tx, bundleID, executableID); err != nil {
		return "", err
	}
	if err := refreshBundleUploadedAt(ctx, tx, bundleID); err != nil {
		return "", err
	}
	if event.Decision == ExecutionDecisionBundleBinary {
		return "", nil
	}
	return event.BundleHash, nil
}

func validateEventInputs(
	executionEvents []ExecutionEventInput,
	fileAccessEvents []FileAccessEventInput,
	standaloneRuleCreationEvents []StandaloneRuleCreationEventInput,
) error {
	for _, event := range executionEvents {
		if event.Decision != ExecutionDecisionBundleBinary && event.OccurredAt.IsZero() {
			return fmt.Errorf("%w: execution event occurred_at is required", dbutil.ErrInvalidInput)
		}
	}
	for _, event := range fileAccessEvents {
		if event.OccurredAt.IsZero() {
			return fmt.Errorf("%w: file access event occurred_at is required", dbutil.ErrInvalidInput)
		}
	}
	for _, event := range standaloneRuleCreationEvents {
		if strings.TrimSpace(event.Identifier) == "" {
			return fmt.Errorf("%w: standalone rule identifier is required", dbutil.ErrInvalidInput)
		}
		if event.Decision == ExecutionDecisionUnknown {
			return fmt.Errorf("%w: standalone rule decision is required", dbutil.ErrInvalidInput)
		}
		if _, ok := validExecutionDecisions[event.Decision]; !ok {
			return fmt.Errorf("%w: unknown standalone rule decision", dbutil.ErrInvalidInput)
		}
		if event.OccurredAt.IsZero() {
			return fmt.Errorf("%w: standalone rule occurred_at is required", dbutil.ErrInvalidInput)
		}
	}
	return nil
}

// ListEvents returns execution events and the total count matching params.
func (s *Store) ListEvents(ctx context.Context, params ExecutionEventListParams) ([]ExecutionEvent, int, error) {
	params.ListParams = dbutil.NormalizeListParams(params.ListParams)
	params.Decisions = normalizeListValues(params.Decisions)
	params.User = strings.TrimSpace(params.User)
	if err := validateExecutionEventListParams(params); err != nil {
		return nil, 0, err
	}
	where, args := executionEventWhere(params)
	rows, count, err := dbutil.ListWithCount[executionEventRow](
		ctx,
		s.db.Pool(),
		executionEventListQuery(params, where, args),
	)
	if err != nil {
		return nil, 0, err
	}
	return executionEventsFromRows(rows), count, nil
}

// GetExecutionEvent returns one execution event by id.
func (s *Store) GetExecutionEvent(ctx context.Context, id int64) (*ExecutionEvent, error) {
	row, err := dbutil.GetOne[executionEventRow](ctx, s.db.Pool(), executionEventSelectSQL()+"\nWHERE ee.id = $1", id)
	if err != nil {
		return nil, err
	}
	event := executionEventFromRow(row)
	return &event, nil
}

// ListFileAccessEvents returns file-access events and the total count matching params.
func (s *Store) ListFileAccessEvents(
	ctx context.Context,
	params FileAccessEventListParams,
) ([]FileAccessEvent, int, error) {
	params.ListParams = dbutil.NormalizeListParams(params.ListParams)
	params.Decisions = normalizeListValues(params.Decisions)
	if err := validateFileAccessEventListParams(params); err != nil {
		return nil, 0, err
	}
	where, args := fileAccessEventWhere(params)
	rows, count, err := dbutil.ListWithCount[fileAccessEventRow](
		ctx,
		s.db.Pool(),
		fileAccessEventListQuery(params, where, args),
	)
	if err != nil {
		return nil, 0, err
	}
	return fileAccessEventsFromRows(rows), count, nil
}

// GetFileAccessEvent returns one file-access event by id.
func (s *Store) GetFileAccessEvent(ctx context.Context, id int64) (*FileAccessEvent, error) {
	row, err := dbutil.GetOne[fileAccessEventRow](
		ctx,
		s.db.Pool(),
		fileAccessEventSelectSQL()+"\nWHERE fae.id = $1",
		id,
	)
	if err != nil {
		return nil, err
	}
	event := fileAccessEventFromRow(row)
	return &event, nil
}

// SweepEventsBefore deletes Santa events that occurred before cutoff.
func (s *Store) SweepEventsBefore(ctx context.Context, cutoff time.Time) (int, error) {
	var deleted int
	err := s.db.Pool().QueryRow(ctx, `
WITH deleted_execution AS (
	DELETE FROM santa_execution_events
	WHERE santa_execution_events.occurred_at < $1
	RETURNING 1
),
deleted_file_access AS (
	DELETE FROM santa_file_access_events
	WHERE santa_file_access_events.occurred_at < $1
	RETURNING 1
),
deleted_standalone_rules AS (
	DELETE FROM santa_standalone_rule_creation_events
	WHERE santa_standalone_rule_creation_events.occurred_at < $1
	RETURNING 1
)
SELECT
	(SELECT count(*) FROM deleted_execution)::integer
	+ (SELECT count(*) FROM deleted_file_access)::integer
	+ (SELECT count(*) FROM deleted_standalone_rules)::integer AS deleted_count`,
		cutoff,
	).Scan(&deleted)
	if err != nil {
		return 0, err
	}
	return deleted, nil
}

func upsertExecutable(ctx context.Context, tx pgx.Tx, event ExecutionEventInput) (int64, error) {
	write := executableWrite{
		SHA256:                      event.FileSHA256,
		FileName:                    event.FileName,
		FileBundleID:                event.BundleID,
		FileBundlePath:              event.BundlePath,
		FileBundleExecutableRelPath: event.BundleExecutableRelPath,
		FileBundleName:              event.BundleName,
		FileBundleVersion:           event.BundleVersion,
		FileBundleVersionString:     event.BundleVersionString,
		FileBundleHash:              event.BundleHash,
		FileBundleHashMillis:        event.BundleHashMillis,
		FileBundleBinaryCount:       event.BundleBinaryCount,
		SigningID:                   event.SigningID,
		TeamID:                      event.TeamID,
		CDHash:                      event.CDHash,
		CodesigningFlags:            int64(event.CodesigningFlags),
		SigningStatus:               string(normalizeSigningStatus(event.SigningStatus)),
		SecureSigningTime:           timeOrNil(event.SecureSigningTime),
		SigningTime:                 timeOrNil(event.SigningTime),
		Entitlements:                executableEntitlements(event),
	}
	var id int64
	if err := tx.QueryRow(ctx, `
INSERT INTO santa_executables (
	sha256,
	file_name,
	file_bundle_id,
	file_bundle_path,
	file_bundle_executable_rel_path,
	file_bundle_name,
	file_bundle_version,
	file_bundle_version_string,
	file_bundle_hash,
	file_bundle_hash_millis,
	file_bundle_binary_count,
	signing_id,
	team_id,
	cdhash,
	codesigning_flags,
	signing_status,
	secure_signing_time,
	signing_time,
	entitlements,
	updated_at
)
VALUES (
	@sha256,
	@file_name,
	@file_bundle_id,
	@file_bundle_path,
	@file_bundle_executable_rel_path,
	@file_bundle_name,
	@file_bundle_version,
	@file_bundle_version_string,
	@file_bundle_hash,
	@file_bundle_hash_millis,
	@file_bundle_binary_count,
	@signing_id,
	@team_id,
	@cdhash,
	@codesigning_flags,
	@signing_status::santa_signing_status,
	@secure_signing_time::timestamptz,
	@signing_time::timestamptz,
	@entitlements,
	now()
)
ON CONFLICT (sha256) DO UPDATE SET
	file_name = EXCLUDED.file_name,
	file_bundle_id = EXCLUDED.file_bundle_id,
	file_bundle_path = EXCLUDED.file_bundle_path,
	file_bundle_executable_rel_path = EXCLUDED.file_bundle_executable_rel_path,
	file_bundle_name = EXCLUDED.file_bundle_name,
	file_bundle_version = EXCLUDED.file_bundle_version,
	file_bundle_version_string = EXCLUDED.file_bundle_version_string,
	file_bundle_hash = EXCLUDED.file_bundle_hash,
	file_bundle_hash_millis = EXCLUDED.file_bundle_hash_millis,
	file_bundle_binary_count = EXCLUDED.file_bundle_binary_count,
	signing_id = EXCLUDED.signing_id,
	team_id = EXCLUDED.team_id,
	cdhash = EXCLUDED.cdhash,
	codesigning_flags = EXCLUDED.codesigning_flags,
	signing_status = EXCLUDED.signing_status,
	secure_signing_time = EXCLUDED.secure_signing_time,
	signing_time = EXCLUDED.signing_time,
	entitlements = EXCLUDED.entitlements,
	updated_at = now()
RETURNING id`, pgx.StructArgs(write)).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func upsertSigningChain(ctx context.Context, tx pgx.Tx, executableID int64, chain []CertificateInput) error {
	entries := signingChainEntries(chain)
	if len(entries) == 0 {
		return nil
	}
	var chainID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO santa_signing_chains (sha256)
VALUES ($1)
ON CONFLICT (sha256) DO UPDATE SET sha256 = EXCLUDED.sha256
RETURNING id`,
		signingChainHash(entries),
	).Scan(&chainID); err != nil {
		return err
	}
	for position, entry := range entries {
		certificateID, err := upsertCertificate(ctx, tx, entry)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO santa_signing_chain_entries (signing_chain_id, position, certificate_id)
VALUES (@signing_chain_id, @position, @certificate_id)
ON CONFLICT (signing_chain_id, position) DO UPDATE SET certificate_id = EXCLUDED.certificate_id`,
			pgx.NamedArgs{
				"signing_chain_id": chainID,
				"position":         int32(position),
				"certificate_id":   certificateID,
			}); err != nil {
			return err
		}
	}
	_, err := tx.Exec(ctx, `
INSERT INTO santa_executable_signing_chains (executable_id, signing_chain_id)
VALUES (@executable_id, @signing_chain_id)
ON CONFLICT DO NOTHING`,
		pgx.NamedArgs{
			"executable_id":    executableID,
			"signing_chain_id": chainID,
		})
	return err
}

func upsertCertificate(ctx context.Context, tx pgx.Tx, entry signingChainEntry) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
INSERT INTO santa_certificates (
	sha256,
	common_name,
	organization,
	organizational_unit,
	valid_from,
	valid_until,
	updated_at
)
VALUES (
	@sha256,
	@common_name,
	@organization,
	@organizational_unit,
	@valid_from::timestamptz,
	@valid_until::timestamptz,
	now()
)
ON CONFLICT (sha256) DO UPDATE SET
	common_name = EXCLUDED.common_name,
	organization = EXCLUDED.organization,
	organizational_unit = EXCLUDED.organizational_unit,
	valid_from = EXCLUDED.valid_from,
	valid_until = EXCLUDED.valid_until,
	updated_at = now()
RETURNING id`,
		pgx.NamedArgs{
			"sha256":              entry.SHA256,
			"common_name":         entry.CommonName,
			"organization":        entry.Org,
			"organizational_unit": entry.OU,
			"valid_from":          certificateTime(entry.ValidFrom),
			"valid_until":         certificateTime(entry.ValidUntil),
		}).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func upsertBundle(ctx context.Context, tx pgx.Tx, event ExecutionEventInput) (int64, bool, error) {
	if event.BundleHash == "" {
		return 0, false, nil
	}
	write := bundleWrite{
		SHA256:            event.BundleHash,
		BundleID:          event.BundleID,
		Name:              event.BundleName,
		Path:              event.BundlePath,
		ExecutableRelPath: event.BundleExecutableRelPath,
		Version:           event.BundleVersion,
		VersionString:     event.BundleVersionString,
		BinaryCount:       event.BundleBinaryCount,
		HashMillis:        event.BundleHashMillis,
	}
	var id int64
	if err := tx.QueryRow(ctx, `
INSERT INTO santa_bundles (
	sha256,
	bundle_id,
	name,
	path,
	executable_rel_path,
	version,
	version_string,
	binary_count,
	hash_millis,
	updated_at
)
VALUES (
	@sha256,
	@bundle_id,
	@name,
	@path,
	@executable_rel_path,
	@version,
	@version_string,
	@binary_count,
	@hash_millis,
	now()
)
ON CONFLICT (sha256) DO UPDATE SET
	bundle_id = COALESCE(NULLIF(EXCLUDED.bundle_id, ''), santa_bundles.bundle_id),
	name = COALESCE(NULLIF(EXCLUDED.name, ''), santa_bundles.name),
	path = COALESCE(NULLIF(EXCLUDED.path, ''), santa_bundles.path),
	executable_rel_path = COALESCE(NULLIF(EXCLUDED.executable_rel_path, ''), santa_bundles.executable_rel_path),
	version = COALESCE(NULLIF(EXCLUDED.version, ''), santa_bundles.version),
	version_string = COALESCE(NULLIF(EXCLUDED.version_string, ''), santa_bundles.version_string),
	binary_count = CASE
		WHEN EXCLUDED.binary_count > 0 THEN EXCLUDED.binary_count
		ELSE santa_bundles.binary_count
	END,
	hash_millis = CASE
		WHEN EXCLUDED.hash_millis > 0 THEN EXCLUDED.hash_millis
		ELSE santa_bundles.hash_millis
	END,
	updated_at = now()
RETURNING id`, pgx.StructArgs(write)).Scan(&id); err != nil {
		return 0, false, err
	}
	return id, true, nil
}

func linkBundleExecutable(ctx context.Context, tx pgx.Tx, bundleID int64, executableID int64) error {
	_, err := tx.Exec(ctx, `
INSERT INTO santa_bundle_executables (bundle_id, executable_id)
VALUES (@bundle_id, @executable_id)
ON CONFLICT DO NOTHING`,
		pgx.NamedArgs{
			"bundle_id":     bundleID,
			"executable_id": executableID,
		})
	return err
}

func refreshBundleUploadedAt(ctx context.Context, tx pgx.Tx, bundleID int64) error {
	_, err := tx.Exec(ctx, `
UPDATE santa_bundles b
SET uploaded_at = COALESCE(uploaded_at, now()), updated_at = now()
WHERE b.id = @bundle_id
  AND b.binary_count > 0
  AND (
	  SELECT count(*)
	  FROM santa_bundle_executables be
	  WHERE be.bundle_id = b.id
  ) >= b.binary_count`,
		pgx.NamedArgs{"bundle_id": bundleID},
	)
	return err
}

func incompleteBundleHashes(ctx context.Context, tx pgx.Tx, candidates []string) ([]string, error) {
	hashes := normalizeStringSlice(candidates)
	if len(hashes) == 0 {
		return nil, nil
	}
	rows, err := tx.Query(ctx, `
SELECT b.sha256
FROM santa_bundles b
WHERE b.sha256 = ANY($1::text[])
  AND b.uploaded_at IS NULL
ORDER BY b.sha256`,
		hashes,
	)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
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

func insertFileAccessEvent(ctx context.Context, tx pgx.Tx, hostID int64, event FileAccessEventInput) error {
	chain := processChainColumn(processEntries(event.ProcessChain))
	primary := primaryProcess(event.ProcessChain)
	write := fileAccessEventWrite{
		HostID:                  hostID,
		RuleVersion:             event.RuleVersion,
		RuleName:                event.RuleName,
		Target:                  event.Target,
		Decision:                string(event.Decision),
		PrimaryProcessSHA256:    primary.FileSHA256,
		PrimaryProcessPath:      primary.FilePath,
		PrimaryProcessSigningID: primary.SigningID,
		PrimaryProcessTeamID:    normalizeTeamID(primary.TeamID),
		PrimaryProcessCDHash:    primary.CDHash,
		PrimaryProcessPID:       primary.PID,
		ProcessChain:            chain,
		OccurredAt:              event.OccurredAt,
	}
	_, err := tx.Exec(ctx, `
INSERT INTO santa_file_access_events (
	host_id,
	rule_version,
	rule_name,
	target,
	decision,
	primary_process_sha256,
	primary_process_path,
	primary_process_signing_id,
	primary_process_team_id,
	primary_process_cdhash,
	primary_process_pid,
	process_chain,
	occurred_at
)
VALUES (
	@host_id,
	@rule_version,
	@rule_name,
	@target,
	@decision::santa_file_access_decision,
	@primary_process_sha256,
	@primary_process_path,
	@primary_process_signing_id,
	@primary_process_team_id,
	@primary_process_cdhash,
	@primary_process_pid,
	@process_chain::jsonb,
	@occurred_at
)`, pgx.StructArgs(write))
	return err
}

func timeOrNil(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func primaryProcess(chain []ProcessInput) ProcessInput {
	if len(chain) == 0 {
		return ProcessInput{}
	}
	return chain[0]
}

func normalizeStringSlice(values []string) []string {
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
	if len(event.Entitlements) == 0 {
		return nil
	}
	return event.Entitlements
}

func processEntries(chain []ProcessInput) []Process {
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
	return processes
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
		entry := signingChainEntry(cert)
		if entry.SHA256 == "" {
			continue
		}
		entries = append(entries, entry)
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
	return payloadhash.Hash(fields...)
}

func normalizeTeamID(value string) string {
	if value == "<unknown team id>" {
		return ""
	}
	return value
}

func normalizeSigningStatus(status SigningStatus) SigningStatus {
	if slices.Contains(SigningStatusValues, status) {
		return status
	}
	return SigningStatusUnspecified
}

func executionEventWhere(params ExecutionEventListParams) (string, []any) {
	var where dbutil.WhereBuilder
	if params.HostID != 0 {
		where.Add("ee.host_id = " + where.Arg(params.HostID))
	}
	if params.User != "" {
		where.Add("ee.executing_user = " + where.Arg(params.User))
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
			OR e.file_bundle_name ILIKE ` + search + `
			OR e.file_bundle_hash ILIKE ` + search + `
			OR e.signing_id ILIKE ` + search + `
			OR e.team_id ILIKE ` + search + `
			OR e.cdhash ILIKE ` + search + `
		)`)
	}
	addExecutionEventFilters(&where, params)
	return where.Build()
}

func fileAccessEventWhere(params FileAccessEventListParams) (string, []any) {
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
	if !params.Since.IsZero() {
		where.Add("fae.occurred_at >= " + where.Arg(params.Since))
	}
	if len(params.Decisions) > 0 {
		clauses := make([]string, 0, len(params.Decisions))
		for _, decision := range params.Decisions {
			clauses = append(clauses, "fae.decision = "+where.Arg(decision))
		}
		where.Add("(" + strings.Join(clauses, " OR ") + ")")
	}
	return where.Build()
}

func addExecutionEventFilters(where *dbutil.WhereBuilder, params ExecutionEventListParams) {
	if !params.Since.IsZero() {
		where.Add("ee.occurred_at >= " + where.Arg(params.Since))
	}
	if len(params.Decisions) == 0 {
		return
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
			clauses = append(clauses, "ee.decision = "+where.Arg(decision))
		}
	}
	where.Add("(" + strings.Join(clauses, " OR ") + ")")
}

func executionEventListQuery(params ExecutionEventListParams, where string, args []any) dbutil.ListQuery {
	return dbutil.ListQuery{
		SelectSQL:    executionEventSelectSQL(),
		WhereSQL:     where,
		Args:         args,
		OrderKeys:    eventOrderKeys("ee", "e"),
		Params:       params.ListParams,
		DefaultOrder: defaultEventOrder("ee"),
	}
}

func fileAccessEventListQuery(params FileAccessEventListParams, where string, args []any) dbutil.ListQuery {
	return dbutil.ListQuery{
		SelectSQL:    fileAccessEventSelectSQL(),
		WhereSQL:     where,
		Args:         args,
		OrderKeys:    fileAccessEventOrderKeys(),
		Params:       params.ListParams,
		DefaultOrder: defaultEventOrder("fae"),
	}
}

func executionEventsFromRows(rows []executionEventRow) []ExecutionEvent {
	events := make([]ExecutionEvent, len(rows))
	for i, row := range rows {
		events[i] = executionEventFromRow(row)
	}
	return events
}

func fileAccessEventsFromRows(rows []fileAccessEventRow) []FileAccessEvent {
	events := make([]FileAccessEvent, len(rows))
	for i, row := range rows {
		events[i] = fileAccessEventFromRow(row)
	}
	return events
}

func normalizeListValues[T ~string](items []T) []T {
	raw := make([]string, len(items))
	for i, item := range items {
		raw[i] = string(item)
	}
	values := dbutil.NormalizeListValues(raw)
	out := make([]T, len(values))
	for i, value := range values {
		out[i] = T(value)
	}
	return out
}

func validateExecutionEventListParams(params ExecutionEventListParams) error {
	if err := dbutil.ValidateListParams(params.ListParams); err != nil {
		return err
	}
	if params.HostID < 0 {
		return fmt.Errorf("%w: host_id must be non-negative", dbutil.ErrInvalidInput)
	}
	for _, filter := range params.Decisions {
		if filter == DecisionFilterAllowed || filter == DecisionFilterBlocked {
			continue
		}
		if _, ok := validExecutionDecisions[ExecutionDecision(filter)]; !ok {
			return fmt.Errorf("%w: unknown decision", dbutil.ErrInvalidInput)
		}
	}
	return nil
}

func validateFileAccessEventListParams(params FileAccessEventListParams) error {
	if err := dbutil.ValidateListParams(params.ListParams); err != nil {
		return err
	}
	if params.HostID < 0 {
		return fmt.Errorf("%w: host_id must be non-negative", dbutil.ErrInvalidInput)
	}
	for _, decision := range params.Decisions {
		if _, ok := validFileAccessDecisions[decision]; !ok {
			return fmt.Errorf("%w: unknown file access decision", dbutil.ErrInvalidInput)
		}
	}
	return nil
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
		{SQL: alias + ".occurred_at", Descending: true},
		{SQL: alias + ".id", Descending: true},
	}
}

// executionEventRow is the canonical scan target for the execution-event projection.
type executionEventRow struct {
	ID                          int64              `db:"id"`
	HostID                      int64              `db:"host_id"`
	DisplayName                 string             `db:"display_name"`
	Hostname                    string             `db:"hostname"`
	ComputerName                string             `db:"computer_name"`
	HardwareSerial              string             `db:"hardware_serial"`
	HardwareModelIdentifier     string             `db:"hardware_model_identifier"`
	SantaMachineID              string             `db:"santa_machine_id"`
	SantaVersion                string             `db:"santa_version"`
	SantaClientMode             string             `db:"santa_client_mode"`
	FilePath                    string             `db:"file_path"`
	ExecutingUser               string             `db:"executing_user"`
	PID                         int32              `db:"pid"`
	PPID                        int32              `db:"ppid"`
	ParentName                  string             `db:"parent_name"`
	LoggedInUsers               []string           `db:"logged_in_users"`
	CurrentSessions             []string           `db:"current_sessions"`
	Decision                    string             `db:"decision"`
	StaticRule                  bool               `db:"static_rule"`
	OccurredAt                  time.Time          `db:"occurred_at"`
	IngestedAt                  time.Time          `db:"ingested_at"`
	ExecutableID                int64              `db:"executable_id"`
	SHA256                      string             `db:"sha256"`
	FileName                    string             `db:"file_name"`
	FileBundleID                string             `db:"file_bundle_id"`
	FileBundlePath              string             `db:"file_bundle_path"`
	FileBundleExecutableRelPath string             `db:"file_bundle_executable_rel_path"`
	FileBundleName              string             `db:"file_bundle_name"`
	FileBundleVersion           string             `db:"file_bundle_version"`
	FileBundleVersionString     string             `db:"file_bundle_version_string"`
	FileBundleHash              string             `db:"file_bundle_hash"`
	FileBundleHashMillis        int32              `db:"file_bundle_hash_millis"`
	FileBundleBinaryCount       int32              `db:"file_bundle_binary_count"`
	SigningID                   string             `db:"signing_id"`
	TeamID                      string             `db:"team_id"`
	CDHash                      string             `db:"cdhash"`
	CodesigningFlags            int64              `db:"codesigning_flags"`
	SigningStatus               string             `db:"signing_status"`
	SecureSigningTime           *time.Time         `db:"secure_signing_time"`
	SigningTime                 *time.Time         `db:"signing_time"`
	Entitlements                []byte             `db:"entitlements"`
	SigningChain                signingChainColumn `db:"signing_chain"`
}

// fileAccessEventRow is the canonical scan target for the file-access-event projection.
type fileAccessEventRow struct {
	ID                      int64              `db:"id"`
	HostID                  int64              `db:"host_id"`
	DisplayName             string             `db:"display_name"`
	Hostname                string             `db:"hostname"`
	ComputerName            string             `db:"computer_name"`
	HardwareSerial          string             `db:"hardware_serial"`
	HardwareModelIdentifier string             `db:"hardware_model_identifier"`
	SantaMachineID          string             `db:"santa_machine_id"`
	SantaVersion            string             `db:"santa_version"`
	SantaClientMode         string             `db:"santa_client_mode"`
	RuleVersion             string             `db:"rule_version"`
	RuleName                string             `db:"rule_name"`
	Target                  string             `db:"target"`
	Decision                string             `db:"decision"`
	ProcessChain            processChainColumn `db:"process_chain"`
	OccurredAt              time.Time          `db:"occurred_at"`
	IngestedAt              time.Time          `db:"ingested_at"`
}

func (row executionEventRow) host() HostSummary {
	return HostSummary{
		ID:           row.HostID,
		DisplayName:  row.DisplayName,
		Hostname:     row.Hostname,
		ComputerName: row.ComputerName,
		Hardware: HostSummaryHardware{
			Serial:          row.HardwareSerial,
			ModelIdentifier: row.HardwareModelIdentifier,
		},
		SantaMachineID:  row.SantaMachineID,
		SantaVersion:    row.SantaVersion,
		SantaClientMode: configurations.ReportedClientMode(row.SantaClientMode),
	}
}

func (row fileAccessEventRow) host() HostSummary {
	return HostSummary{
		ID:           row.HostID,
		DisplayName:  row.DisplayName,
		Hostname:     row.Hostname,
		ComputerName: row.ComputerName,
		Hardware: HostSummaryHardware{
			Serial:          row.HardwareSerial,
			ModelIdentifier: row.HardwareModelIdentifier,
		},
		SantaMachineID:  row.SantaMachineID,
		SantaVersion:    row.SantaVersion,
		SantaClientMode: configurations.ReportedClientMode(row.SantaClientMode),
	}
}

// executionEventFromRow assembles an ExecutionEvent from a scanned executionEventRow.
func executionEventFromRow(row executionEventRow) ExecutionEvent {
	event := ExecutionEvent{
		ID:              row.ID,
		HostID:          row.HostID,
		Host:            row.host(),
		FilePath:        row.FilePath,
		ExecutingUser:   row.ExecutingUser,
		PID:             row.PID,
		PPID:            row.PPID,
		ParentName:      row.ParentName,
		LoggedInUsers:   row.LoggedInUsers,
		CurrentSessions: row.CurrentSessions,
		Decision:        ExecutionDecision(row.Decision),
		StaticRule:      row.StaticRule,
		OccurredAt:      row.OccurredAt,
		IngestedAt:      row.IngestedAt,
		Executable: Executable{
			ID:                      row.ExecutableID,
			SHA256:                  row.SHA256,
			FileName:                row.FileName,
			BundleID:                row.FileBundleID,
			BundlePath:              row.FileBundlePath,
			BundleExecutableRelPath: row.FileBundleExecutableRelPath,
			BundleName:              row.FileBundleName,
			BundleVersion:           row.FileBundleVersion,
			BundleVersionString:     row.FileBundleVersionString,
			BundleHash:              row.FileBundleHash,
			BundleHashMillis:        row.FileBundleHashMillis,
			BundleBinaryCount:       row.FileBundleBinaryCount,
			SigningID:               row.SigningID,
			TeamID:                  row.TeamID,
			CDHash:                  row.CDHash,
			CodesigningFlags:        uint32(row.CodesigningFlags),
			SigningStatus:           normalizeSigningStatus(SigningStatus(row.SigningStatus)),
			SecureSigningTime:       row.SecureSigningTime,
			SigningTime:             row.SigningTime,
			SigningChain:            signingChainOutputEntries(row.SigningChain),
		},
	}
	if len(row.Entitlements) > 0 {
		event.Executable.Entitlements = append(json.RawMessage(nil), row.Entitlements...)
	}
	return event
}

// fileAccessEventFromRow assembles a FileAccessEvent from a scanned fileAccessEventRow.
func fileAccessEventFromRow(row fileAccessEventRow) FileAccessEvent {
	event := FileAccessEvent{
		ID:           row.ID,
		HostID:       row.HostID,
		Host:         row.host(),
		RuleVersion:  row.RuleVersion,
		RuleName:     row.RuleName,
		Target:       row.Target,
		Decision:     FileAccessDecision(row.Decision),
		ProcessChain: row.ProcessChain,
		OccurredAt:   row.OccurredAt,
		IngestedAt:   row.IngestedAt,
	}
	if len(event.ProcessChain) > 0 {
		event.PrimaryProcess = event.ProcessChain[0]
	}
	return event
}

func hostEventSelectSQL() string {
	return `
	h.id AS host_id,
	h.display_name,
	h.hostname,
	h.computer_name,
	h.hardware_serial,
	h.hardware_model_identifier,
	COALESCE(sh.machine_id, '') AS santa_machine_id,
	COALESCE(sh.santa_version, '') AS santa_version,
	COALESCE(sh.client_mode_reported::text, '') AS santa_client_mode`
}

func executionEventSelectSQL() string {
	return `
SELECT
	ee.id,
` + hostEventSelectSQL() + `,
	ee.file_path,
	ee.executing_user,
	ee.pid,
	ee.ppid,
	ee.parent_name,
	ee.logged_in_users,
	ee.current_sessions,
	ee.decision::text AS decision,
	ee.static_rule,
	ee.occurred_at,
	ee.ingested_at,
	e.id AS executable_id,
	e.sha256,
	e.file_name,
	e.file_bundle_id,
	e.file_bundle_path,
	e.file_bundle_executable_rel_path,
	e.file_bundle_name,
	e.file_bundle_version,
	e.file_bundle_version_string,
	e.file_bundle_hash,
	e.file_bundle_hash_millis,
	e.file_bundle_binary_count,
	e.signing_id,
	e.team_id,
	e.cdhash,
	e.codesigning_flags,
	e.signing_status::text AS signing_status,
	e.secure_signing_time,
	e.signing_time,
	e.entitlements,
	COALESCE((
		SELECT jsonb_agg(
			jsonb_build_object(
				'sha256', c.sha256,
				'common_name', c.common_name,
				'org', c.organization,
				'ou', c.organizational_unit,
				'valid_from', COALESCE(extract(epoch from c.valid_from)::integer, 0),
				'valid_until', COALESCE(extract(epoch from c.valid_until)::integer, 0)
			)
			ORDER BY sce.position
		)
		FROM (
			SELECT sc.id
			FROM santa_executable_signing_chains esc
			JOIN santa_signing_chains sc ON sc.id = esc.signing_chain_id
			WHERE esc.executable_id = e.id
			ORDER BY sc.first_seen_at DESC, sc.id DESC
			LIMIT 1
		) latest_chain
		JOIN santa_signing_chain_entries sce ON sce.signing_chain_id = latest_chain.id
		JOIN santa_certificates c ON c.id = sce.certificate_id
	), '[]'::jsonb) AS signing_chain
FROM santa_execution_events ee
JOIN santa_executables e ON e.id = ee.executable_id
JOIN hosts h ON h.id = ee.host_id
LEFT JOIN santa_hosts sh ON sh.host_id = h.id`
}

func fileAccessEventSelectSQL() string {
	return `
SELECT
	fae.id,
` + hostEventSelectSQL() + `,
	fae.rule_version,
	fae.rule_name,
	fae.target,
	fae.decision::text AS decision,
	fae.process_chain,
	fae.occurred_at,
	fae.ingested_at
FROM santa_file_access_events fae
JOIN hosts h ON h.id = fae.host_id
LEFT JOIN santa_hosts sh ON sh.host_id = h.id`
}

type executableWrite struct {
	SHA256                      string     `db:"sha256"`
	FileName                    string     `db:"file_name"`
	FileBundleID                string     `db:"file_bundle_id"`
	FileBundlePath              string     `db:"file_bundle_path"`
	FileBundleExecutableRelPath string     `db:"file_bundle_executable_rel_path"`
	FileBundleName              string     `db:"file_bundle_name"`
	FileBundleVersion           string     `db:"file_bundle_version"`
	FileBundleVersionString     string     `db:"file_bundle_version_string"`
	FileBundleHash              string     `db:"file_bundle_hash"`
	FileBundleHashMillis        int32      `db:"file_bundle_hash_millis"`
	FileBundleBinaryCount       int32      `db:"file_bundle_binary_count"`
	SigningID                   string     `db:"signing_id"`
	TeamID                      string     `db:"team_id"`
	CDHash                      string     `db:"cdhash"`
	CodesigningFlags            int64      `db:"codesigning_flags"`
	SigningStatus               string     `db:"signing_status"`
	SecureSigningTime           *time.Time `db:"secure_signing_time"`
	SigningTime                 *time.Time `db:"signing_time"`
	Entitlements                []byte     `db:"entitlements"`
}

type bundleWrite struct {
	SHA256            string `db:"sha256"`
	BundleID          string `db:"bundle_id"`
	Name              string `db:"name"`
	Path              string `db:"path"`
	ExecutableRelPath string `db:"executable_rel_path"`
	Version           string `db:"version"`
	VersionString     string `db:"version_string"`
	BinaryCount       int32  `db:"binary_count"`
	HashMillis        int32  `db:"hash_millis"`
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

type fileAccessEventWrite struct {
	HostID                  int64              `db:"host_id"`
	RuleVersion             string             `db:"rule_version"`
	RuleName                string             `db:"rule_name"`
	Target                  string             `db:"target"`
	Decision                string             `db:"decision"`
	PrimaryProcessSHA256    string             `db:"primary_process_sha256"`
	PrimaryProcessPath      string             `db:"primary_process_path"`
	PrimaryProcessSigningID string             `db:"primary_process_signing_id"`
	PrimaryProcessTeamID    string             `db:"primary_process_team_id"`
	PrimaryProcessCDHash    string             `db:"primary_process_cdhash"`
	PrimaryProcessPID       int32              `db:"primary_process_pid"`
	ProcessChain            processChainColumn `db:"process_chain"`
	OccurredAt              time.Time          `db:"occurred_at"`
}
