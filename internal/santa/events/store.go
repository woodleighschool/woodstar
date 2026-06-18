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
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
)

// Store persists Santa execution and file-access events.
type Store struct {
	db *database.DB
	q  *sqlc.Queries
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db, q: db.Queries()}
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
) ([]string, error) {
	if len(executionEvents) == 0 && len(fileAccessEvents) == 0 {
		return nil, nil
	}
	if err := validateEventsHaveOccurrenceTimes(executionEvents, fileAccessEvents); err != nil {
		return nil, err
	}
	var bundleBinaryRequests []string
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		requests, err := ingestEventsTx(ctx, tx, hostID, executionEvents, fileAccessEvents)
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

func validateEventsHaveOccurrenceTimes(
	executionEvents []ExecutionEventInput,
	fileAccessEvents []FileAccessEventInput,
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
	return nil
}

// ListEvents returns execution events and the total count matching params.
func (s *Store) ListEvents(ctx context.Context, params ExecutionEventListParams) ([]ExecutionEvent, int, error) {
	params.Decisions = cleanDecisionFilters(params.Decisions)
	where, args, err := executionEventWhere(params)
	if err != nil {
		return nil, 0, err
	}
	return dbutil.ScanListWithCount(
		ctx,
		s.db.Pool(),
		executionEventListQuery(params, where, args),
		scanExecutionEvent,
	)
}

// GetExecutionEvent returns one execution event by id.
func (s *Store) GetExecutionEvent(ctx context.Context, id int64) (*ExecutionEvent, error) {
	row, err := s.q.GetSantaExecutionEvent(ctx, sqlc.GetSantaExecutionEventParams{ID: id})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, dbutil.ErrNotFound
		}
		return nil, err
	}
	event, err := executionEventFromSQLC(row)
	if err != nil {
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
	return dbutil.ScanListWithCount(
		ctx,
		s.db.Pool(),
		fileAccessEventListQuery(params, where, args),
		scanFileAccessEvent,
	)
}

// GetFileAccessEvent returns one file-access event by id.
func (s *Store) GetFileAccessEvent(ctx context.Context, id int64) (*FileAccessEvent, error) {
	row, err := s.q.GetSantaFileAccessEvent(ctx, sqlc.GetSantaFileAccessEventParams{ID: id})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, dbutil.ErrNotFound
		}
		return nil, err
	}
	event, err := fileAccessEventFromSQLC(row)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

// SweepEventsBefore deletes Santa events that occurred before cutoff.
func (s *Store) SweepEventsBefore(ctx context.Context, cutoff time.Time) (int, error) {
	deleted, err := s.q.SweepSantaEventsBefore(ctx, sqlc.SweepSantaEventsBeforeParams{CutoffTime: cutoff})
	return int(deleted), err
}

func upsertExecutable(ctx context.Context, tx pgx.Tx, event ExecutionEventInput) (int64, error) {
	return sqlc.New(tx).UpsertSantaExecutable(ctx, sqlc.UpsertSantaExecutableParams{
		Sha256:                      event.FileSHA256,
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
		Cdhash:                      event.CDHash,
		CodesigningFlags:            int64(event.CodesigningFlags),
		SigningStatus:               sqlc.SantaSigningStatus(normalizeSigningStatus(event.SigningStatus)),
		SecureSigningTime:           timeOrNil(event.SecureSigningTime),
		SigningTime:                 timeOrNil(event.SigningTime),
		Entitlements:                executableEntitlements(event),
	})
}

func upsertSigningChain(ctx context.Context, tx pgx.Tx, executableID int64, chain []CertificateInput) error {
	entries := signingChainEntries(chain)
	if len(entries) == 0 {
		return nil
	}
	q := sqlc.New(tx)
	chainID, err := q.UpsertSantaSigningChain(
		ctx,
		sqlc.UpsertSantaSigningChainParams{Sha256: signingChainHash(entries)},
	)
	if err != nil {
		return err
	}
	for position, entry := range entries {
		certificateID, err := upsertCertificate(ctx, tx, entry)
		if err != nil {
			return err
		}
		if err := q.UpsertSantaSigningChainEntry(ctx, sqlc.UpsertSantaSigningChainEntryParams{
			SigningChainID: chainID,
			Position:       int32(position),
			CertificateID:  certificateID,
		}); err != nil {
			return err
		}
	}
	return q.LinkSantaExecutableSigningChain(ctx, sqlc.LinkSantaExecutableSigningChainParams{
		ExecutableID:   executableID,
		SigningChainID: chainID,
	})
}

func upsertCertificate(ctx context.Context, tx pgx.Tx, entry signingChainEntry) (int64, error) {
	return sqlc.New(tx).UpsertSantaCertificate(ctx, sqlc.UpsertSantaCertificateParams{
		Sha256:             entry.SHA256,
		CommonName:         entry.CommonName,
		Organization:       entry.Org,
		OrganizationalUnit: entry.OU,
		ValidFrom:          certificateTime(entry.ValidFrom),
		ValidUntil:         certificateTime(entry.ValidUntil),
	})
}

func upsertBundle(ctx context.Context, tx pgx.Tx, event ExecutionEventInput) (int64, bool, error) {
	if event.BundleHash == "" {
		return 0, false, nil
	}
	id, err := sqlc.New(tx).UpsertSantaBundle(ctx, sqlc.UpsertSantaBundleParams{
		Sha256:            event.BundleHash,
		BundleID:          event.BundleID,
		Name:              event.BundleName,
		Path:              event.BundlePath,
		ExecutableRelPath: event.BundleExecutableRelPath,
		Version:           event.BundleVersion,
		VersionString:     event.BundleVersionString,
		BinaryCount:       event.BundleBinaryCount,
		HashMillis:        event.BundleHashMillis,
	})
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

func linkBundleExecutable(ctx context.Context, tx pgx.Tx, bundleID int64, executableID int64) error {
	return sqlc.New(tx).LinkSantaBundleExecutable(ctx, sqlc.LinkSantaBundleExecutableParams{
		BundleID:     bundleID,
		ExecutableID: executableID,
	})
}

func refreshBundleUploadedAt(ctx context.Context, tx pgx.Tx, bundleID int64) error {
	return sqlc.New(tx).RefreshSantaBundleUploadedAt(
		ctx,
		sqlc.RefreshSantaBundleUploadedAtParams{BundleID: bundleID},
	)
}

func incompleteBundleHashes(ctx context.Context, tx pgx.Tx, candidates []string) ([]string, error) {
	hashes := cleanStringSlice(candidates)
	if len(hashes) == 0 {
		return nil, nil
	}
	return sqlc.New(tx).ListIncompleteSantaBundleHashes(
		ctx,
		sqlc.ListIncompleteSantaBundleHashesParams{Hashes: hashes},
	)
}

func insertExecutionEvent(
	ctx context.Context,
	tx pgx.Tx,
	hostID int64,
	executableID int64,
	event ExecutionEventInput,
) error {
	return sqlc.New(tx).InsertSantaExecutionEvent(ctx, sqlc.InsertSantaExecutionEventParams{
		HostID:          hostID,
		ExecutableID:    executableID,
		FilePath:        event.FilePath,
		ExecutingUser:   event.ExecutingUser,
		Pid:             event.PID,
		Ppid:            event.PPID,
		ParentName:      event.ParentName,
		LoggedInUsers:   cleanStringSlice(event.LoggedInUsers),
		CurrentSessions: cleanStringSlice(event.CurrentSessions),
		Decision:        sqlc.SantaExecutionDecision(event.Decision),
		OccurredAt:      event.OccurredAt,
	})
}

func insertFileAccessEvent(ctx context.Context, tx pgx.Tx, hostID int64, event FileAccessEventInput) error {
	processChain, err := processChainJSON(event.ProcessChain)
	if err != nil {
		return err
	}
	primary := primaryProcess(event.ProcessChain)
	return sqlc.New(tx).InsertSantaFileAccessEvent(ctx, sqlc.InsertSantaFileAccessEventParams{
		HostID:                  hostID,
		RuleVersion:             event.RuleVersion,
		RuleName:                event.RuleName,
		Target:                  event.Target,
		Decision:                sqlc.SantaFileAccessDecision(event.Decision),
		PrimaryProcessSha256:    primary.FileSHA256,
		PrimaryProcessPath:      primary.FilePath,
		PrimaryProcessSigningID: primary.SigningID,
		PrimaryProcessTeamID:    normalizeTeamID(primary.TeamID),
		PrimaryProcessCdhash:    primary.CDHash,
		PrimaryProcessPid:       primary.PID,
		ProcessChain:            processChain,
		OccurredAt:              event.OccurredAt,
	})
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
	if len(event.Entitlements) == 0 {
		return nil
	}
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
	return syncstate.PayloadHash(fields...)
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

func executionEventWhere(params ExecutionEventListParams) (string, []any, error) {
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
	if !params.Since.IsZero() {
		where.Add("fae.occurred_at >= " + where.Arg(params.Since))
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
	if !params.Since.IsZero() {
		where.Add("ee.occurred_at >= " + where.Arg(params.Since))
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

func executionEventListQuery(params ExecutionEventListParams, where string, args []any) dbutil.ListQuery {
	params.ListParams = dbutil.CleanListParams(params.ListParams)
	return dbutil.ListQuery{
		SelectSQL:    executionEventSelectSQL,
		WhereSQL:     where,
		Args:         args,
		OrderKeys:    eventOrderKeys("ee", "e"),
		Params:       params.ListParams,
		DefaultOrder: defaultEventOrder("ee"),
	}
}

func fileAccessEventListQuery(params FileAccessEventListParams, where string, args []any) dbutil.ListQuery {
	params.ListParams = dbutil.CleanListParams(params.ListParams)
	return dbutil.ListQuery{
		SelectSQL:    fileAccessEventSelectSQL,
		WhereSQL:     where,
		Args:         args,
		OrderKeys:    fileAccessEventOrderKeys(),
		Params:       params.ListParams,
		DefaultOrder: defaultEventOrder("fae"),
	}
}

func scanExecutionEvent(row pgx.Row) (ExecutionEvent, error) {
	var event ExecutionEvent
	var entitlements []byte
	var signingChain []byte
	var codesigningFlags int64
	var signingStatus string
	err := row.Scan(
		&event.ID,
		&event.Host.ID,
		&event.Host.DisplayName,
		&event.Host.Hostname,
		&event.Host.ComputerName,
		&event.Host.Hardware.Serial,
		&event.Host.Hardware.ModelIdentifier,
		&event.Host.SantaMachineID,
		&event.Host.SantaVersion,
		&event.Host.SantaClientMode,
		&event.FilePath,
		&event.ExecutingUser,
		&event.PID,
		&event.PPID,
		&event.ParentName,
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
		&event.Executable.BundleExecutableRelPath,
		&event.Executable.BundleName,
		&event.Executable.BundleVersion,
		&event.Executable.BundleVersionString,
		&event.Executable.BundleHash,
		&event.Executable.BundleHashMillis,
		&event.Executable.BundleBinaryCount,
		&event.Executable.SigningID,
		&event.Executable.TeamID,
		&event.Executable.CDHash,
		&codesigningFlags,
		&signingStatus,
		&event.Executable.SecureSigningTime,
		&event.Executable.SigningTime,
		&entitlements,
		&signingChain,
	)
	if err != nil {
		return event, err
	}
	event.HostID = event.Host.ID
	if codesigningFlags > 0 {
		event.Executable.CodesigningFlags = uint32(codesigningFlags)
	}
	event.Executable.SigningStatus = normalizeSigningStatus(SigningStatus(signingStatus))
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
		&event.Host.Hardware.Serial,
		&event.Host.Hardware.ModelIdentifier,
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

func executionEventFromSQLC(row sqlc.GetSantaExecutionEventRow) (ExecutionEvent, error) {
	event := ExecutionEvent{
		ID: row.ID,
		Host: HostSummary{
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
		},
		FilePath:        row.FilePath,
		ExecutingUser:   row.ExecutingUser,
		PID:             row.Pid,
		PPID:            row.Ppid,
		ParentName:      row.ParentName,
		LoggedInUsers:   row.LoggedInUsers,
		CurrentSessions: row.CurrentSessions,
		Decision:        ExecutionDecision(row.Decision),
		OccurredAt:      row.OccurredAt,
		IngestedAt:      row.IngestedAt,
		Executable: Executable{
			ID:                      row.ExecutableID,
			SHA256:                  row.Sha256,
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
			CDHash:                  row.Cdhash,
			SecureSigningTime:       row.SecureSigningTime,
			SigningTime:             row.SigningTime,
		},
	}
	event.HostID = event.Host.ID
	if row.CodesigningFlags > 0 {
		event.Executable.CodesigningFlags = uint32(row.CodesigningFlags)
	}
	event.Executable.SigningStatus = normalizeSigningStatus(SigningStatus(row.SigningStatus))
	if err := decodeExecutableJSON(&event.Executable, row.Entitlements, []byte(row.SigningChain)); err != nil {
		return event, err
	}
	return event, nil
}

func fileAccessEventFromSQLC(row sqlc.GetSantaFileAccessEventRow) (FileAccessEvent, error) {
	event := FileAccessEvent{
		ID: row.ID,
		Host: HostSummary{
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
		},
		RuleVersion: row.RuleVersion,
		RuleName:    row.RuleName,
		Target:      row.Target,
		Decision:    FileAccessDecision(row.Decision),
		OccurredAt:  row.OccurredAt,
		IngestedAt:  row.IngestedAt,
	}
	event.HostID = event.Host.ID
	if len(row.ProcessChain) > 0 {
		if err := json.Unmarshal(row.ProcessChain, &event.ProcessChain); err != nil {
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
	h.hardware_model_identifier,
	COALESCE(sh.machine_id, ''),
	COALESCE(sh.santa_version, ''),
	COALESCE(sh.client_mode_reported::text, '')`

const executionEventSelectSQL = `
SELECT
	ee.id,
` + hostEventSelectSQL + `,
	ee.file_path,
	ee.executing_user,
	ee.pid,
	ee.ppid,
	ee.parent_name,
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
	e.signing_status::text,
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
	), '[]'::jsonb)
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
