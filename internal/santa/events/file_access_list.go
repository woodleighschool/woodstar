package events

import (
	"context"
	"strings"
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
)

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

func fileAccessEventsFromRows(rows []fileAccessEventRow) []FileAccessEvent {
	events := make([]FileAccessEvent, len(rows))
	for i, row := range rows {
		events[i] = fileAccessEventFromRow(row)
	}
	return events
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
