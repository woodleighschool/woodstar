package hosts

import (
	"context"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/labels"
)

// Store persists hosts.
type Store struct {
	db     *database.DB
	labels hostLabelReader
}

type hostLabelReader interface {
	ListForHost(context.Context, int64) ([]labels.Label, error)
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db, labels: labels.NewStore(db)}
}

// UpsertOnOrbitEnroll creates or refreshes a host from Orbit enroll.
func (s *Store) UpsertOnOrbitEnroll(ctx context.Context, update InventoryUpdate) (*Host, error) {
	write := orbitEnrollWrite{
		HardwareUUID:            update.Hardware.UUID,
		DisplayName:             inventoryDisplayName(update.Hardware.UUID, update.Hostname, update.ComputerName),
		Hostname:                update.Hostname,
		ComputerName:            update.ComputerName,
		HardwareSerial:          update.Hardware.Serial,
		HardwareModelIdentifier: update.Hardware.ModelIdentifier,
		OrbitNodeKey:            update.OrbitNodeKey,
	}
	return s.upsertOnEnroll(ctx, upsertHostOnOrbitEnrollSQL, write)
}

// UpsertOnOsqueryEnroll creates or refreshes a host from osquery enroll.
func (s *Store) UpsertOnOsqueryEnroll(ctx context.Context, update InventoryUpdate) (*Host, error) {
	write := osqueryEnrollWrite{
		HardwareUUID:            update.Hardware.UUID,
		DisplayName:             inventoryDisplayName(update.Hardware.UUID, update.Hostname, update.ComputerName),
		Hostname:                update.Hostname,
		ComputerName:            update.ComputerName,
		HardwareSerial:          update.Hardware.Serial,
		HardwareModelIdentifier: update.Hardware.ModelIdentifier,
		OSName:                  update.OS.Name,
		OSVersion:               update.OS.Version,
		OSBuild:                 update.OS.Build,
		OSPlatform:              update.OS.Platform,
		OsqueryVersion:          update.Agents.Osquery.Version,
		OsqueryNodeKey:          update.OsqueryNodeKey,
		OrbitVersion:            update.Agents.Orbit.Version,
		CPUType:                 update.Hardware.CPU.Architecture,
		CPUSubtype:              update.Hardware.CPU.Subtype,
		CPUBrand:                update.Hardware.CPU.Brand,
		CPULogicalCores:         update.Hardware.CPU.LogicalCores,
		CPUPhysicalCores:        update.Hardware.CPU.PhysicalCores,
		MemoryBytes:             update.Hardware.MemoryBytes,
		HardwareVendor:          update.Hardware.Vendor,
		OSKernelVersion:         update.OS.KernelVersion,
	}
	return s.upsertOnEnroll(ctx, upsertHostOnOsqueryEnrollSQL, write)
}

func (s *Store) upsertOnEnroll(ctx context.Context, sql string, write any) (*Host, error) {
	now := time.Now()
	var host Host
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, sql, pgx.StructArgs(write))
		if err != nil {
			return err
		}
		row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[hostRow])
		if err != nil {
			return err
		}
		host = hostFromRow(row, now)
		_, err = tx.Exec(ctx, addHostToAllHostsLabelSQL, pgx.NamedArgs{
			"host_id":     host.ID,
			"builtin_key": string(labels.BuiltinKeyAllHosts),
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	return &host, nil
}

func (s *Store) List(ctx context.Context, params HostListParams) ([]Host, int, error) {
	where, args, err := hostListWhere(params)
	if err != nil {
		return nil, 0, err
	}
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
	if err := s.attachUserAffinity(ctx, hosts); err != nil {
		return nil, 0, err
	}
	return hosts, count, nil
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Host, error) {
	row, err := dbutil.GetOne[hostRow](ctx, s.db.Pool(), hostSelectSQL+"\nWHERE id = $1", id)
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
	rows, err := s.db.Pool().Query(ctx, hostSelectSQL+`
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

func (s *Store) Delete(ctx context.Context, id int64) error {
	tag, err := s.db.Pool().Exec(ctx, `DELETE FROM hosts WHERE id = $1`, id)
	if err != nil {
		return dbutil.GetError(err)
	}
	if tag.RowsAffected() == 0 {
		return dbutil.ErrNotFound
	}
	return nil
}

// DeleteMany removes hosts. Missing IDs are fine.
func (s *Store) DeleteMany(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	rows, err := s.db.Pool().Query(ctx, `DELETE FROM hosts WHERE id = ANY($1::bigint[]) RETURNING id`, ids)
	if err != nil {
		return 0, err
	}
	deleted, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return 0, err
	}
	return len(deleted), nil
}

func (s *Store) GetByOrbitNodeKey(ctx context.Context, nodeKey string) (*Host, error) {
	return s.touchByNodeKey(ctx, touchHostByOrbitNodeKeySQL, nodeKey)
}

func (s *Store) GetByOsqueryNodeKey(ctx context.Context, nodeKey string) (*Host, error) {
	return s.touchByNodeKey(ctx, touchHostByOsqueryNodeKeySQL, nodeKey)
}

func (s *Store) touchByNodeKey(ctx context.Context, sql, nodeKey string) (*Host, error) {
	row, err := dbutil.GetOne[hostRow](ctx, s.db.Pool(), sql, nodeKey)
	if err != nil {
		return nil, err
	}
	host := hostFromRow(row, time.Now())
	return &host, nil
}

func (s *Store) ApplyInventory(ctx context.Context, hostID int64, update InventoryUpdate) error {
	write := applyInventoryWrite{
		ID:                                hostID,
		Hostname:                          update.Hostname,
		ComputerName:                      update.ComputerName,
		HardwareSerial:                    update.Hardware.Serial,
		HardwareModelIdentifier:           update.Hardware.ModelIdentifier,
		OSName:                            update.OS.Name,
		OSVersion:                         update.OS.Version,
		OSBuild:                           update.OS.Build,
		OSPlatform:                        update.OS.Platform,
		OsqueryVersion:                    update.Agents.Osquery.Version,
		OrbitVersion:                      update.Agents.Orbit.Version,
		CPUType:                           update.Hardware.CPU.Architecture,
		CPUSubtype:                        update.Hardware.CPU.Subtype,
		CPUBrand:                          update.Hardware.CPU.Brand,
		CPULogicalCores:                   update.Hardware.CPU.LogicalCores,
		CPUPhysicalCores:                  update.Hardware.CPU.PhysicalCores,
		MemoryBytes:                       update.Hardware.MemoryBytes,
		HardwareVendor:                    update.Hardware.Vendor,
		OSKernelVersion:                   update.OS.KernelVersion,
		LastRestartedAt:                   update.Timestamps.LastRestartedAt,
		BootVolumeAvailableBytes:          update.Storage.BootVolume.AvailableBytes,
		BootVolumeTotalBytes:              update.Storage.BootVolume.TotalBytes,
		LastRemoteIP:                      update.Network.LastRemoteIP,
		PrimaryIP:                         update.Network.PrimaryIP,
		PrimaryMAC:                        update.Network.PrimaryMAC,
		OsqueryDistributedIntervalSeconds: update.Agents.Osquery.DistributedIntervalSeconds,
		OsqueryConfigRefreshSeconds:       update.Agents.Osquery.ConfigRefreshSeconds,
	}
	_, err := s.db.Pool().Exec(ctx, applyHostInventorySQL, pgx.StructArgs(write))
	return err
}

func (s *Store) ReplaceUsers(ctx context.Context, hostID int64, users []HostUser) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, deleteHostUsersSQL, hostID); err != nil {
			return err
		}
		for _, user := range users {
			if user.UID == "" || user.Username == "" {
				continue
			}
			if _, err := tx.Exec(ctx, insertHostUserSQL, pgx.StructArgs(newHostUserWrite(hostID, user))); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) ReplaceBatteries(ctx context.Context, hostID int64, batteries []HostBattery) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, deleteHostBatteriesSQL, hostID); err != nil {
			return err
		}
		for _, battery := range batteries {
			if battery.SerialNumber == "" {
				continue
			}
			if _, err := tx.Exec(
				ctx,
				insertHostBatterySQL,
				pgx.StructArgs(newHostBatteryWrite(hostID, battery)),
			); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) ReplaceCertificates(ctx context.Context, hostID int64, certificates []HostCertificate) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, deleteHostCertificatesSQL, hostID); err != nil {
			return err
		}
		for _, certificate := range certificates {
			if certificate.SHA1 == "" {
				continue
			}
			if _, err := tx.Exec(
				ctx,
				insertHostCertificateSQL,
				pgx.StructArgs(newHostCertificateWrite(hostID, certificate)),
			); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) ListUsers(ctx context.Context, hostID int64) ([]HostUser, error) {
	rows, err := s.db.Pool().Query(ctx, listHostUsersSQL, hostID)
	if err != nil {
		return nil, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[hostUserRow])
	if err != nil {
		return nil, err
	}
	users := make([]HostUser, len(records))
	for i, record := range records {
		users[i] = hostUserFromRow(record)
	}
	return users, nil
}

func (s *Store) ListBatteries(ctx context.Context, hostID int64) ([]HostBattery, error) {
	rows, err := s.db.Pool().Query(ctx, listHostBatteriesSQL, hostID)
	if err != nil {
		return nil, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[hostBatteryRow])
	if err != nil {
		return nil, err
	}
	batteries := make([]HostBattery, len(records))
	for i, record := range records {
		batteries[i] = hostBatteryFromRow(record)
	}
	return batteries, nil
}

func (s *Store) ListCertificates(ctx context.Context, hostID int64) ([]HostCertificate, error) {
	rows, err := s.db.Pool().Query(ctx, listHostCertificatesSQL, hostID)
	if err != nil {
		return nil, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[hostCertificateRow])
	if err != nil {
		return nil, err
	}
	certificates := make([]HostCertificate, len(records))
	for i, record := range records {
		certificates[i] = hostCertificateFromRow(record)
	}
	return certificates, nil
}

func (s *Store) MarkInventoryFresh(ctx context.Context, hostID int64, inventoryQueryHash string) error {
	_, err := s.db.Pool().Exec(ctx, markHostInventoryFreshSQL, pgx.NamedArgs{
		"id":                   hostID,
		"inventory_query_hash": inventoryQueryHash,
	})
	return err
}

func (s *Store) attachUserAffinity(ctx context.Context, hosts []Host) error {
	if len(hosts) == 0 {
		return nil
	}
	hostIDs := make([]int64, len(hosts))
	for i := range hosts {
		hostIDs[i] = hosts[i].ID
	}
	affinity, err := s.loadUserAffinity(ctx, hostIDs)
	if err != nil {
		return err
	}
	for i := range hosts {
		hosts[i].UserAffinity = affinity[hosts[i].ID]
	}
	return nil
}

func hostListQuery(params HostListParams, where string, args []any) dbutil.ListQuery {
	return dbutil.ListQuery{
		SelectSQL: hostSelectSQL,
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

func hostListWhere(params HostListParams) (string, []any, error) {
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
					SELECT 1 FROM host_user_affinity_mappings he
					WHERE he.host_id = hosts.id AND he.email ILIKE ` + search + `
				)
			)`)
	}
	if len(params.IDs) > 0 {
		where.Addf("id = ANY(%s::bigint[])", params.IDs)
	}
	switch params.Status {
	case "":
	case "online":
		where.Add("last_seen_at >= now() - interval '5 minutes'")
	case "offline":
		where.Add("(last_seen_at IS NULL OR last_seen_at < now() - interval '5 minutes')")
	default:
		return "", nil, fmt.Errorf("%w: unknown status %q", dbutil.ErrInvalidInput, params.Status)
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
	whereSQL, args := where.Build()
	return whereSQL, args, nil
}

// inventoryDisplayName persists the canonical host label exposed by the API.
func inventoryDisplayName(hardwareUUID, hostname, computerName string) string {
	if computerName != "" {
		return computerName
	}
	if hostname != "" {
		return hostname
	}
	return hardwareUUID
}

func statusFromLastSeen(lastSeen *time.Time, now time.Time) HostStatus {
	if lastSeen == nil || lastSeen.Before(now.Add(-hostOnlineWindow)) {
		return HostStatusOffline
	}
	return HostStatusOnline
}

func (s *Store) loadUserAffinity(ctx context.Context, hostIDs []int64) (map[int64]HostUserAffinity, error) {
	affinity := make(map[int64]HostUserAffinity, len(hostIDs))
	for _, hostID := range hostIDs {
		affinity[hostID] = HostUserAffinity{Mappings: []HostUserAffinityMapping{}}
	}
	if len(hostIDs) == 0 {
		return affinity, nil
	}

	mappingRows, err := s.db.Pool().Query(ctx, listHostUserAffinityMappingsForHostsSQL, hostIDs)
	if err != nil {
		return nil, err
	}
	mappings, err := pgx.CollectRows(mappingRows, pgx.RowToStructByName[hostUserAffinityMappingRow])
	if err != nil {
		return nil, err
	}
	grouped := groupHostUserAffinityMappings(mappings)

	primaryRows, err := s.db.Pool().Query(ctx, listHostUserAffinityPrimariesSQL, hostIDs)
	if err != nil {
		return nil, err
	}
	primaries, err := pgx.CollectRows(primaryRows, pgx.RowToStructByName[hostUserAffinityPrimaryRow])
	if err != nil {
		return nil, err
	}
	for _, row := range primaries {
		hostAffinity := affinity[row.HostID]
		hostAffinity.Primary = &HostUserAffinityPrimary{
			Email:      row.Email,
			Username:   row.Username,
			Name:       row.Name,
			Department: row.Department,
			Groups:     row.Groups,
			Source:     UserAffinitySource(row.Source),
		}
		affinity[row.HostID] = hostAffinity
	}
	for hostID, hostMappings := range grouped {
		hostAffinity := affinity[hostID]
		hostAffinity.Mappings = hostMappings
		affinity[hostID] = hostAffinity
	}
	return affinity, nil
}

// hostRow is the canonical scan target for the hosts projection.
type hostRow struct {
	ID                                int64       `db:"id"`
	HardwareUUID                      string      `db:"hardware_uuid"`
	DisplayName                       string      `db:"display_name"`
	Hostname                          string      `db:"hostname"`
	ComputerName                      string      `db:"computer_name"`
	HardwareSerial                    string      `db:"hardware_serial"`
	HardwareModelIdentifier           string      `db:"hardware_model_identifier"`
	HardwareVendor                    string      `db:"hardware_vendor"`
	OSName                            string      `db:"os_name"`
	OSVersion                         string      `db:"os_version"`
	OSBuild                           string      `db:"os_build"`
	OSPlatform                        string      `db:"os_platform"`
	OsqueryVersion                    string      `db:"osquery_version"`
	OrbitVersion                      string      `db:"orbit_version"`
	OrbitNodeKey                      string      `db:"orbit_node_key"`
	OsqueryNodeKey                    string      `db:"osquery_node_key"`
	EnrollmentAgent                   string      `db:"enrollment_agent"`
	CPUType                           string      `db:"cpu_type"`
	CPUSubtype                        string      `db:"cpu_subtype"`
	CPUBrand                          string      `db:"cpu_brand"`
	CPULogicalCores                   int32       `db:"cpu_logical_cores"`
	CPUPhysicalCores                  int32       `db:"cpu_physical_cores"`
	MemoryBytes                       int64       `db:"memory_bytes"`
	OSKernelVersion                   string      `db:"os_kernel_version"`
	LastRestartedAt                   *time.Time  `db:"last_restarted_at"`
	BootVolumeAvailableBytes          *int64      `db:"boot_volume_available_bytes"`
	BootVolumeTotalBytes              *int64      `db:"boot_volume_total_bytes"`
	LastRemoteIP                      *netip.Addr `db:"last_remote_ip"`
	PrimaryIP                         *netip.Addr `db:"primary_ip"`
	PrimaryMAC                        string      `db:"primary_mac"`
	OsqueryDistributedIntervalSeconds *int32      `db:"osquery_distributed_interval_seconds"`
	OsqueryConfigRefreshSeconds       *int32      `db:"osquery_config_refresh_seconds"`
	InventoryQueryHash                string      `db:"inventory_query_hash"`
	EnrolledAt                        *time.Time  `db:"enrolled_at"`
	LastSeenAt                        *time.Time  `db:"last_seen_at"`
	InventoryUpdatedAt                *time.Time  `db:"inventory_updated_at"`
	CreatedAt                         time.Time   `db:"created_at"`
	UpdatedAt                         time.Time   `db:"updated_at"`
}

func hostFromRow(row hostRow, now time.Time) Host {
	return Host{
		ID:           row.ID,
		DisplayName:  row.DisplayName,
		Status:       statusFromLastSeen(row.LastSeenAt, now),
		Hostname:     row.Hostname,
		ComputerName: row.ComputerName,
		Enrollment: HostEnrollment{
			Agent:      row.EnrollmentAgent,
			EnrolledAt: row.EnrolledAt,
		},
		Hardware: HostHardware{
			UUID:            row.HardwareUUID,
			Serial:          row.HardwareSerial,
			Vendor:          row.HardwareVendor,
			ModelIdentifier: row.HardwareModelIdentifier,
			MemoryBytes:     row.MemoryBytes,
			CPU: HostCPU{
				Architecture:  row.CPUType,
				Subtype:       row.CPUSubtype,
				Brand:         row.CPUBrand,
				LogicalCores:  row.CPULogicalCores,
				PhysicalCores: row.CPUPhysicalCores,
			},
		},
		OS: HostOS{
			Platform:      row.OSPlatform,
			Name:          row.OSName,
			Version:       row.OSVersion,
			Build:         row.OSBuild,
			KernelVersion: row.OSKernelVersion,
		},
		Storage: HostStorage{
			BootVolume: HostBootVolume{
				AvailableBytes: row.BootVolumeAvailableBytes,
				TotalBytes:     row.BootVolumeTotalBytes,
			},
		},
		Network: HostNetwork{
			PrimaryIP:    row.PrimaryIP,
			PrimaryMAC:   row.PrimaryMAC,
			LastRemoteIP: row.LastRemoteIP,
		},
		Agents: HostAgents{
			Osquery: HostOsqueryAgent{
				Version:                    row.OsqueryVersion,
				DistributedIntervalSeconds: row.OsqueryDistributedIntervalSeconds,
				ConfigRefreshSeconds:       row.OsqueryConfigRefreshSeconds,
			},
			Orbit: HostOrbitAgent{Version: row.OrbitVersion},
		},
		UserAffinity: HostUserAffinity{Mappings: []HostUserAffinityMapping{}},
		Timestamps: HostTimestamps{
			CreatedAt:          row.CreatedAt,
			UpdatedAt:          row.UpdatedAt,
			LastSeenAt:         row.LastSeenAt,
			InventoryUpdatedAt: row.InventoryUpdatedAt,
			LastRestartedAt:    row.LastRestartedAt,
		},
		OrbitNodeKey:       row.OrbitNodeKey,
		OsqueryNodeKey:     row.OsqueryNodeKey,
		InventoryQueryHash: row.InventoryQueryHash,
	}
}

type hostUserRow struct {
	ID          int64     `db:"id"`
	HostID      int64     `db:"host_id"`
	UID         string    `db:"uid"`
	Username    string    `db:"username"`
	Type        string    `db:"type"`
	Description string    `db:"description"`
	Directory   string    `db:"directory"`
	Shell       string    `db:"shell"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

func hostUserFromRow(row hostUserRow) HostUser {
	return HostUser(row)
}

type hostBatteryRow struct {
	ID               int64     `db:"id"`
	HostID           int64     `db:"host_id"`
	SerialNumber     string    `db:"serial_number"`
	Manufacturer     string    `db:"manufacturer"`
	Model            string    `db:"model"`
	Chemistry        string    `db:"chemistry"`
	CycleCount       *int32    `db:"cycle_count"`
	Health           string    `db:"health"`
	DesignedCapacity *int32    `db:"designed_capacity"`
	MaxCapacity      *int32    `db:"max_capacity"`
	CurrentCapacity  *int32    `db:"current_capacity"`
	PercentRemaining *float64  `db:"percent_remaining"`
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}

func hostBatteryFromRow(row hostBatteryRow) HostBattery {
	return HostBattery(row)
}

type hostCertificateRow struct {
	ID                        int64      `db:"id"`
	HostID                    int64      `db:"host_id"`
	SHA1                      string     `db:"sha1"`
	CommonName                string     `db:"common_name"`
	SubjectCountry            string     `db:"subject_country"`
	SubjectOrganization       string     `db:"subject_organization"`
	SubjectOrganizationalUnit string     `db:"subject_organizational_unit"`
	SubjectCommonName         string     `db:"subject_common_name"`
	IssuerCountry             string     `db:"issuer_country"`
	IssuerOrganization        string     `db:"issuer_organization"`
	IssuerOrganizationalUnit  string     `db:"issuer_organizational_unit"`
	IssuerCommonName          string     `db:"issuer_common_name"`
	KeyAlgorithm              string     `db:"key_algorithm"`
	KeyStrength               *int32     `db:"key_strength"`
	KeyUsage                  string     `db:"key_usage"`
	SigningAlgorithm          string     `db:"signing_algorithm"`
	NotValidAfter             *time.Time `db:"not_valid_after"`
	NotValidBefore            *time.Time `db:"not_valid_before"`
	Serial                    string     `db:"serial"`
	CertificateAuthority      bool       `db:"certificate_authority"`
	Source                    string     `db:"source"`
	Username                  string     `db:"username"`
	Path                      string     `db:"path"`
	CreatedAt                 time.Time  `db:"created_at"`
	UpdatedAt                 time.Time  `db:"updated_at"`
}

func hostCertificateFromRow(row hostCertificateRow) HostCertificate {
	return HostCertificate{
		ID:         row.ID,
		HostID:     row.HostID,
		SHA1:       row.SHA1,
		CommonName: row.CommonName,
		Subject: CertificateName{
			Country:            row.SubjectCountry,
			Organization:       row.SubjectOrganization,
			OrganizationalUnit: row.SubjectOrganizationalUnit,
			CommonName:         row.SubjectCommonName,
		},
		Issuer: CertificateName{
			Country:            row.IssuerCountry,
			Organization:       row.IssuerOrganization,
			OrganizationalUnit: row.IssuerOrganizationalUnit,
			CommonName:         row.IssuerCommonName,
		},
		KeyAlgorithm:         row.KeyAlgorithm,
		KeyStrength:          row.KeyStrength,
		KeyUsage:             row.KeyUsage,
		SigningAlgorithm:     row.SigningAlgorithm,
		NotValidAfter:        row.NotValidAfter,
		NotValidBefore:       row.NotValidBefore,
		Serial:               row.Serial,
		CertificateAuthority: row.CertificateAuthority,
		Source:               row.Source,
		Username:             row.Username,
		Path:                 row.Path,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}
