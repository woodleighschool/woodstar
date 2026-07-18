package events

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
)

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

func executionEventsFromRows(rows []executionEventRow) []ExecutionEvent {
	events := make([]ExecutionEvent, len(rows))
	for i, row := range rows {
		events[i] = executionEventFromRow(row)
	}
	return events
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
