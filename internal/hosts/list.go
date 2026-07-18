package hosts

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func (s *Store) List(ctx context.Context, params HostListParams) ([]Host, int, error) {
	params.normalize()
	if err := params.validate(); err != nil {
		return nil, 0, err
	}
	where, args := hostListWhere(params)
	listQuery := hostListQuery(params, where, args)
	rows, count, err := dbutil.ListWithCount[hostRow](ctx, s.db.Pool(), listQuery)
	if err != nil {
		return nil, 0, err
	}
	now := time.Now()
	hosts := make([]Host, len(rows))
	for i, row := range rows {
		hosts[i] = hostFromRow(row, now)
	}
	if err := s.attachPrimaryUser(ctx, hosts); err != nil {
		return nil, 0, err
	}
	return hosts, count, nil
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Host, error) {
	row, err := dbutil.GetOne[hostRow](ctx, s.db.Pool(), hostSelectSQL()+"\nWHERE id = $1", id)
	if err != nil {
		return nil, err
	}
	host := hostFromRow(row, time.Now())
	return &host, nil
}

// GetByHardwareSerial returns the existing host with serial.
func (s *Store) GetByHardwareSerial(ctx context.Context, serial string) (*Host, error) {
	serial = strings.TrimSpace(serial)
	if serial == "" {
		return nil, dbutil.ErrNotFound
	}
	rows, err := s.db.Pool().Query(ctx, hostSelectSQL()+`
WHERE hardware_serial = $1 AND hardware_serial <> ''
ORDER BY updated_at DESC, id DESC
LIMIT 2`, serial)
	if err != nil {
		return nil, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[hostRow])
	if err != nil {
		return nil, err
	}
	switch len(records) {
	case 0:
		return nil, dbutil.ErrNotFound
	case 1:
		host := hostFromRow(records[0], time.Now())
		return &host, nil
	default:
		return nil, fmt.Errorf("multiple hosts have hardware serial %q", serial)
	}
}

func hostListQuery(params HostListParams, where string, args []any) dbutil.ListQuery {
	return dbutil.ListQuery{
		SelectSQL: hostSelectSQL(),
		WhereSQL:  where,
		Args:      args,
		OrderKeys: map[string]dbutil.OrderExpr{
			"display_name":                        {SQL: "lower(display_name)"},
			"hardware.serial":                     {SQL: "lower(hardware_serial)"},
			"hardware.model_identifier":           {SQL: "lower(hardware_model_identifier)"},
			"hardware.uuid":                       {SQL: "hardware_uuid"},
			"os.version":                          {SQL: "lower(os_version)"},
			"agents.osquery.version":              {SQL: "lower(osquery_version)"},
			"timestamps.last_seen_at":             {SQL: "last_seen_at", NullOrder: dbutil.NullsLast},
			"timestamps.last_restarted_at":        {SQL: "last_restarted_at", NullOrder: dbutil.NullsLast},
			"storage.boot_volume.available_bytes": {SQL: "boot_volume_available_bytes", NullOrder: dbutil.NullsLast},
			"hardware.memory_bytes":               {SQL: "memory_bytes"},
			"network.primary_ip":                  {SQL: "primary_ip", NullOrder: dbutil.NullsLast},
			"network.last_remote_ip":              {SQL: "last_remote_ip", NullOrder: dbutil.NullsLast},
		},
		DefaultOrder: []dbutil.OrderExpr{{SQL: "lower(display_name)"}, {SQL: "id"}},
		Params:       params.ListParams,
	}
}

func hostListWhere(params HostListParams) (string, []any) {
	var where dbutil.WhereBuilder
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add(`(
			display_name ILIKE ` + search + `
			OR hostname ILIKE ` + search + `
				OR computer_name ILIKE ` + search + `
				OR hardware_serial ILIKE ` + search + `
				OR hardware_uuid ILIKE ` + search + `
				OR hardware_model_identifier ILIKE ` + search + `
				OR os_version ILIKE ` + search + `
				OR EXISTS (
					SELECT 1 FROM host_primary_user_sources s
					WHERE s.host_id = hosts.id AND s.email ILIKE ` + search + `
				)
			)`)
	}
	if len(params.IDs) > 0 {
		where.Addf("id = ANY(%s::bigint[])", params.IDs)
	}
	switch params.Status {
	case "":
	case HostStatusOnline:
		where.Add("last_seen_at >= now() - interval '5 minutes'")
	case HostStatusOffline:
		where.Add("(last_seen_at IS NULL OR last_seen_at < now() - interval '5 minutes')")
	}
	if params.LabelID != 0 {
		labelID := where.Arg(params.LabelID)
		where.Add(`EXISTS (
			SELECT 1 FROM label_membership lm
			WHERE lm.host_id = hosts.id AND lm.label_id = ` + labelID + `::bigint
		)`)
	}
	if params.SoftwareID != 0 {
		softwareID := where.Arg(params.SoftwareID)
		where.Add(`EXISTS (
			SELECT 1 FROM host_software hs
			WHERE hs.host_id = hosts.id AND hs.software_id = ` + softwareID + `::bigint
		)`)
	}
	if params.SoftwareTitleID != 0 {
		softwareTitleID := where.Arg(params.SoftwareTitleID)
		where.Add(`EXISTS (
			SELECT 1
			FROM host_software hs
			JOIN software s ON s.id = hs.software_id
			WHERE hs.host_id = hosts.id AND s.title_id = ` + softwareTitleID + `::bigint
		)`)
	}
	return where.Build()
}
