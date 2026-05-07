package models

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
)

// Host is an enrolled Mac.
type Host struct {
	ID                      int64
	HardwareUUID            string
	DisplayName             string
	Hostname                string
	ComputerName            string
	HardwareSerial          string
	HardwareModel           string
	HardwareVersion         string
	OSName                  string
	Platform                string
	PlatformLike            string
	OSVersion               string
	OSBuild                 string
	OsqueryVersion          string
	OrbitVersion            string
	CPUType                 string
	CPUSubtype              string
	OrbitNodeKey            string
	OsqueryNodeKey          string
	CPUBrand                string
	CPULogicalCores         int
	CPUPhysicalCores        int
	PhysicalMemory          int64
	HardwareVendor          string
	KernelVersion           string
	UptimeSeconds           *int64
	LastRestartedAt         *time.Time
	DiskSpaceAvailableBytes *int64
	DiskSpaceTotalBytes     *int64
	PublicIP                *netip.Addr
	PrimaryIP               *netip.Addr
	PrimaryMAC              string
	DistributedInterval     *int32
	ConfigTLSRefresh        *int32
	EnrolledAt              *time.Time
	LastSeenAt              *time.Time
	DetailUpdatedAt         *time.Time
	DetailQueryHash         string
	LabelUpdatedAt          *time.Time
	SoftwareUpdatedAt       *time.Time
	Labels                  []Label
	Users                   []HostUser
	Batteries               []HostBattery
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// HostUser is one local account reported by osquery for a host.
type HostUser struct {
	ID          int64
	HostID      int64
	UID         string
	Username    string
	Type        string
	Description string
	Directory   string
	Shell       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// HostBattery is one battery reported by osquery for a host.
type HostBattery struct {
	ID               int64
	HostID           int64
	SerialNumber     string
	Manufacturer     string
	Model            string
	Chemistry        string
	CycleCount       *int32
	Health           string
	DesignedCapacity *int32
	MaxCapacity      *int32
	CurrentCapacity  *int32
	PercentRemaining *float64
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// HostStore persists Orbit-managed Macs.
type HostStore struct {
	db *database.DB
	q  *sqlc.Queries
}

// HostListParams filters host list results.
type HostListParams struct {
	ListParams

	Status          string
	Platform        string
	LabelID         int64
	SoftwareTitleID int64
	SoftwareID      int64
}

// NewHostStore returns a host store backed by db.
func NewHostStore(db *database.DB) *HostStore {
	return &HostStore{db: db, q: db.Queries()}
}

// EnrollParams holds the fields supplied by an Orbit enrollment request.
// Only HardwareUUID and OrbitNodeKey are required; the rest are best-effort hints.
type EnrollParams struct {
	HardwareUUID   string
	HardwareSerial string
	Hostname       string
	ComputerName   string
	HardwareModel  string
	Platform       string
	PlatformLike   string
	OrbitNodeKey   string
}

// HostDetailUpdate is inventory reported by osquery detail queries.
type HostDetailUpdate struct {
	HardwareUUID            string
	Hostname                string
	ComputerName            string
	HardwareSerial          string
	HardwareModel           string
	HardwareVersion         string
	OSName                  string
	OSVersion               string
	OSBuild                 string
	Platform                string
	PlatformLike            string
	KernelVersion           string
	HardwareVendor          string
	OrbitVersion            string
	CPUType                 string
	CPUSubtype              string
	CPUBrand                string
	CPULogicalCores         int
	CPUPhysicalCores        int
	PhysicalMemory          int64
	OsqueryVersion          string
	OsqueryNodeKey          string
	UptimeSeconds           *int64
	LastRestartedAt         *time.Time
	DiskSpaceAvailableBytes *int64
	DiskSpaceTotalBytes     *int64
	PublicIP                string
	PrimaryIP               string
	PrimaryMAC              string
	DistributedInterval     *int32
	ConfigTLSRefresh        *int32
}

// UpsertOnOrbitEnroll inserts a new host or refreshes an existing one keyed by hardware UUID.
// Re-enrollment overwrites the node key so prior keys stop authenticating.
func (s *HostStore) UpsertOnOrbitEnroll(ctx context.Context, params EnrollParams) (*Host, error) {
	params, err := cleanOrbitEnrollParams(params)
	if err != nil {
		return nil, err
	}

	row, err := s.q.UpsertHostOnOrbitEnroll(ctx, upsertOrbitEnrollParams(params))
	if err != nil {
		return nil, err
	}
	return hostFromRecord(hostRecord(row)), nil
}

// UpsertOnOsqueryEnroll refreshes the osquery node key and rich host inventory.
func (s *HostStore) UpsertOnOsqueryEnroll(ctx context.Context, update HostDetailUpdate) (*Host, error) {
	update, err := cleanHostDetailUpdate(update)
	if err != nil {
		return nil, err
	}

	row, err := s.q.UpsertHostOnOsqueryEnroll(ctx, upsertOsqueryEnrollParams(update))
	if err != nil {
		return nil, err
	}
	return hostFromRecord(hostRecord(row)), nil
}

// List returns active hosts and the total count matching params.
func (s *HostStore) List(ctx context.Context, params HostListParams) ([]Host, int, error) {
	params = cleanHostListParams(params)
	whereSQL, args := hostWhere(params)

	countSQL := `SELECT count(*) FROM hosts h` + whereSQL
	var total int
	if err := s.db.Pool().QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	orderSQL := hostOrder(params.OrderKey, params.OrderDirection)
	limitIndex := len(args) + 1
	args = append(args, int32(params.PerPage), int32((params.Page-1)*params.PerPage))
	rows, err := s.db.Pool().Query(ctx, hostListSQL(whereSQL, orderSQL, limitIndex), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	hosts := make([]Host, 0)
	for rows.Next() {
		var row hostRecord
		if err := rows.Scan(
			&row.ID,
			&row.HardwareUUID,
			&row.DisplayName,
			&row.Hostname,
			&row.ComputerName,
			&row.HardwareSerial,
			&row.HardwareModel,
			&row.HardwareVersion,
			&row.OSName,
			&row.Platform,
			&row.PlatformLike,
			&row.OSVersion,
			&row.OSBuild,
			&row.OsqueryVersion,
			&row.OrbitVersion,
			&row.CPUType,
			&row.CPUSubtype,
			&row.OrbitNodeKey,
			&row.OsqueryNodeKey,
			&row.CPUBrand,
			&row.CPULogicalCores,
			&row.CPUPhysicalCores,
			&row.PhysicalMemory,
			&row.HardwareVendor,
			&row.KernelVersion,
			&row.UptimeSeconds,
			&row.LastRestartedAt,
			&row.DiskSpaceAvailableBytes,
			&row.DiskSpaceTotalBytes,
			&row.PublicIP,
			&row.PrimaryIP,
			&row.PrimaryMAC,
			&row.DistributedInterval,
			&row.ConfigTLSRefresh,
			&row.EnrolledAt,
			&row.LastSeenAt,
			&row.DetailUpdatedAt,
			&row.DetailQueryHash,
			&row.LabelUpdatedAt,
			&row.SoftwareUpdatedAt,
			&row.CreatedAt,
			&row.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		hosts = append(hosts, *hostFromRecord(row))
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return hosts, total, nil
}

func cleanHostListParams(params HostListParams) HostListParams {
	params.ListParams = CleanListParams(params.ListParams)
	params.Status = strings.TrimSpace(params.Status)
	params.Platform = strings.TrimSpace(params.Platform)
	return params
}

func hostWhere(params HostListParams) (string, []any) {
	clauses := []string{"h.deleted_at IS NULL"}
	args := make([]any, 0)

	if params.Q != "" {
		args = append(args, "%"+params.Q+"%")
		placeholder := fmt.Sprintf("$%d", len(args))
		clauses = append(clauses, `(
			h.display_name ILIKE `+placeholder+`
			OR h.hostname ILIKE `+placeholder+`
			OR h.computer_name ILIKE `+placeholder+`
			OR h.hardware_serial ILIKE `+placeholder+`
			OR h.hardware_uuid ILIKE `+placeholder+`
			OR h.hardware_model ILIKE `+placeholder+`
			OR h.os_version ILIKE `+placeholder+`
			OR EXISTS (
				SELECT 1 FROM host_emails he
				WHERE he.host_id = h.id AND he.email ILIKE `+placeholder+`
			)
		)`)
	}
	if params.Platform != "" {
		args = append(args, params.Platform)
		clauses = append(clauses, fmt.Sprintf("h.platform = $%d", len(args)))
	}
	if params.Status != "" {
		switch params.Status {
		case "online":
			clauses = append(clauses, "h.last_seen_at >= now() - interval '5 minutes'")
		case "offline":
			clauses = append(clauses, "(h.last_seen_at IS NULL OR h.last_seen_at < now() - interval '5 minutes')")
		}
	}
	if params.LabelID > 0 {
		args = append(args, params.LabelID)
		clauses = append(clauses, fmt.Sprintf(`EXISTS (
			SELECT 1 FROM label_membership lm
			WHERE lm.host_id = h.id AND lm.label_id = $%d
		)`, len(args)))
	}
	if params.SoftwareID > 0 {
		args = append(args, params.SoftwareID)
		clauses = append(clauses, fmt.Sprintf(`EXISTS (
			SELECT 1 FROM host_software hs
			WHERE hs.host_id = h.id AND hs.software_id = $%d
		)`, len(args)))
	}
	if params.SoftwareTitleID > 0 {
		args = append(args, params.SoftwareTitleID)
		clauses = append(clauses, fmt.Sprintf(`EXISTS (
			SELECT 1
			FROM host_software hs
			JOIN software s ON s.id = hs.software_id
			WHERE hs.host_id = h.id AND s.title_id = $%d
		)`, len(args)))
	}

	return " WHERE " + strings.Join(clauses, " AND "), args
}

func hostOrder(orderKey string, direction string) string {
	directionSQL := orderSQLAsc
	if direction == orderDesc {
		directionSQL = orderSQLDesc
	}
	switch orderKey {
	case "platform":
		return "ORDER BY lower(h.platform) " + directionSQL + ", lower(h.display_name), h.id"
	case "hardware_serial":
		return "ORDER BY lower(h.hardware_serial) " + directionSQL + ", lower(h.display_name), h.id"
	case "os_version":
		return "ORDER BY lower(h.os_version) " + directionSQL + ", lower(h.display_name), h.id"
	case "last_seen_at":
		return "ORDER BY h.last_seen_at " + directionSQL + " NULLS LAST, lower(h.display_name), h.id"
	default:
		return "ORDER BY lower(h.display_name) " + directionSQL + ", h.id"
	}
}

func hostListSQL(whereSQL string, orderSQL string, limitIndex int) string {
	return `
SELECT
	h.id,
	h.hardware_uuid,
	h.display_name,
	h.hostname,
	h.computer_name,
	h.hardware_serial,
	h.hardware_model,
	h.hardware_version,
	h.os_name,
	h.platform,
	h.platform_like,
	h.os_version,
	h.os_build,
	h.osquery_version,
	h.orbit_version,
	h.cpu_type,
	h.cpu_subtype,
	COALESCE(h.orbit_node_key, '')::text AS orbit_node_key,
	COALESCE(h.osquery_node_key, '')::text AS osquery_node_key,
	h.cpu_brand,
	h.cpu_logical_cores,
	h.cpu_physical_cores,
	h.physical_memory,
	h.hardware_vendor,
	h.kernel_version,
	h.uptime_seconds,
	h.last_restarted_at,
	h.disk_space_available_bytes,
	h.disk_space_total_bytes,
	h.public_ip,
	h.primary_ip,
	h.primary_mac,
	h.distributed_interval,
	h.config_tls_refresh,
	h.enrolled_at,
	h.last_seen_at,
	h.detail_updated_at,
	h.detail_query_hash,
	h.label_updated_at,
	h.software_updated_at,
	h.created_at,
	h.updated_at
FROM hosts h
` + whereSQL + `
` + orderSQL + `
LIMIT $` + strconv.Itoa(limitIndex) + ` OFFSET $` + strconv.Itoa(limitIndex+1)
}

// GetByID returns a single active host by database ID.
func (s *HostStore) GetByID(ctx context.Context, id int64) (*Host, error) {
	row, err := s.q.GetHostByID(ctx, sqlc.GetHostByIDParams{ID: id})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return hostFromRecord(hostRecord(row)), nil
}

// GetByOrbitNodeKey returns an active host and refreshes last_seen_at.
func (s *HostStore) GetByOrbitNodeKey(ctx context.Context, nodeKey string) (*Host, error) {
	nodeKey = strings.TrimSpace(nodeKey)
	if nodeKey == "" {
		return nil, ErrNotFound
	}
	row, err := s.q.TouchHostByOrbitNodeKey(ctx, sqlc.TouchHostByOrbitNodeKeyParams{
		OrbitNodeKey: nodeKey,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return hostFromRecord(hostRecord(row)), nil
}

// GetByOsqueryNodeKey returns an active host and refreshes last_seen_at.
func (s *HostStore) GetByOsqueryNodeKey(ctx context.Context, nodeKey string) (*Host, error) {
	nodeKey = strings.TrimSpace(nodeKey)
	if nodeKey == "" {
		return nil, ErrNotFound
	}
	row, err := s.q.TouchHostByOsqueryNodeKey(ctx, sqlc.TouchHostByOsqueryNodeKeyParams{
		OsqueryNodeKey: nodeKey,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return hostFromRecord(hostRecord(row)), nil
}

// ApplyDetail updates the host fields reported by successful detail queries.
func (s *HostStore) ApplyDetail(ctx context.Context, hostID int64, update HostDetailUpdate) error {
	return s.q.ApplyHostDetail(ctx, sqlc.ApplyHostDetailParams{
		ID:                      hostID,
		Hostname:                update.Hostname,
		ComputerName:            update.ComputerName,
		HardwareSerial:          update.HardwareSerial,
		HardwareModel:           update.HardwareModel,
		HardwareVersion:         update.HardwareVersion,
		OSName:                  update.OSName,
		OSVersion:               update.OSVersion,
		OSBuild:                 update.OSBuild,
		Platform:                update.Platform,
		PlatformLike:            update.PlatformLike,
		OsqueryVersion:          update.OsqueryVersion,
		OrbitVersion:            update.OrbitVersion,
		CPUType:                 update.CPUType,
		CPUSubtype:              update.CPUSubtype,
		CPUBrand:                update.CPUBrand,
		CPULogicalCores:         int32(update.CPULogicalCores),
		CPUPhysicalCores:        int32(update.CPUPhysicalCores),
		PhysicalMemory:          update.PhysicalMemory,
		HardwareVendor:          update.HardwareVendor,
		KernelVersion:           update.KernelVersion,
		UptimeSeconds:           update.UptimeSeconds,
		LastRestartedAt:         update.LastRestartedAt,
		DiskSpaceAvailableBytes: update.DiskSpaceAvailableBytes,
		DiskSpaceTotalBytes:     update.DiskSpaceTotalBytes,
		PublicIP:                update.PublicIP,
		PrimaryIP:               update.PrimaryIP,
		PrimaryMAC:              update.PrimaryMAC,
		DistributedInterval:     update.DistributedInterval,
		ConfigTLSRefresh:        update.ConfigTLSRefresh,
	})
}

// ReplaceUsers replaces the local user inventory for hostID.
func (s *HostStore) ReplaceUsers(ctx context.Context, hostID int64, users []HostUser) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.DeleteHostUsers(ctx, sqlc.DeleteHostUsersParams{HostID: hostID}); err != nil {
			return err
		}
		for _, user := range users {
			if user.UID == "" || user.Username == "" {
				continue
			}
			if err := q.InsertHostUser(ctx, sqlc.InsertHostUserParams{
				HostID:      hostID,
				UID:         user.UID,
				Username:    user.Username,
				Type:        user.Type,
				Description: user.Description,
				Directory:   user.Directory,
				Shell:       user.Shell,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

// ReplaceBatteries replaces the battery inventory for hostID.
func (s *HostStore) ReplaceBatteries(ctx context.Context, hostID int64, batteries []HostBattery) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.DeleteHostBatteries(ctx, sqlc.DeleteHostBatteriesParams{HostID: hostID}); err != nil {
			return err
		}
		for _, battery := range batteries {
			if battery.SerialNumber == "" {
				continue
			}
			if err := q.InsertHostBattery(ctx, sqlc.InsertHostBatteryParams{
				HostID:           hostID,
				SerialNumber:     battery.SerialNumber,
				Manufacturer:     battery.Manufacturer,
				Model:            battery.Model,
				Chemistry:        battery.Chemistry,
				CycleCount:       battery.CycleCount,
				Health:           battery.Health,
				DesignedCapacity: battery.DesignedCapacity,
				MaxCapacity:      battery.MaxCapacity,
				CurrentCapacity:  battery.CurrentCapacity,
				PercentRemaining: battery.PercentRemaining,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

// ListUsers returns local users reported for hostID.
func (s *HostStore) ListUsers(ctx context.Context, hostID int64) ([]HostUser, error) {
	rows, err := s.q.ListHostUsers(ctx, sqlc.ListHostUsersParams{HostID: hostID})
	if err != nil {
		return nil, err
	}
	users := make([]HostUser, 0, len(rows))
	for _, row := range rows {
		users = append(users, HostUser{
			ID:          row.ID,
			HostID:      row.HostID,
			UID:         row.UID,
			Username:    row.Username,
			Type:        row.Type,
			Description: row.Description,
			Directory:   row.Directory,
			Shell:       row.Shell,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		})
	}
	return users, nil
}

// ListBatteries returns batteries reported for hostID.
func (s *HostStore) ListBatteries(ctx context.Context, hostID int64) ([]HostBattery, error) {
	rows, err := s.q.ListHostBatteries(ctx, sqlc.ListHostBatteriesParams{HostID: hostID})
	if err != nil {
		return nil, err
	}
	batteries := make([]HostBattery, 0, len(rows))
	for _, row := range rows {
		batteries = append(batteries, HostBattery{
			ID:               row.ID,
			HostID:           row.HostID,
			SerialNumber:     row.SerialNumber,
			Manufacturer:     row.Manufacturer,
			Model:            row.Model,
			Chemistry:        row.Chemistry,
			CycleCount:       row.CycleCount,
			Health:           row.Health,
			DesignedCapacity: row.DesignedCapacity,
			MaxCapacity:      row.MaxCapacity,
			CurrentCapacity:  row.CurrentCapacity,
			PercentRemaining: row.PercentRemaining,
			CreatedAt:        row.CreatedAt,
			UpdatedAt:        row.UpdatedAt,
		})
	}
	return batteries, nil
}

// MarkDetailFresh records that all built-in detail queries completed.
func (s *HostStore) MarkDetailFresh(ctx context.Context, hostID int64, detailQueryHash string) error {
	return s.q.MarkHostDetailFresh(ctx, sqlc.MarkHostDetailFreshParams{ID: hostID, DetailQueryHash: detailQueryHash})
}

func cleanOrbitEnrollParams(params EnrollParams) (EnrollParams, error) {
	params.HardwareUUID = strings.TrimSpace(params.HardwareUUID)
	params.OrbitNodeKey = strings.TrimSpace(params.OrbitNodeKey)
	if params.HardwareUUID == "" {
		return EnrollParams{}, errors.New("hardware_uuid is required")
	}
	if params.OrbitNodeKey == "" {
		return EnrollParams{}, errors.New("orbit_node_key is required")
	}
	return params, nil
}

func cleanHostDetailUpdate(update HostDetailUpdate) (HostDetailUpdate, error) {
	update.HardwareUUID = strings.TrimSpace(update.HardwareUUID)
	update.OsqueryNodeKey = strings.TrimSpace(update.OsqueryNodeKey)
	if update.HardwareUUID == "" {
		return HostDetailUpdate{}, errors.New("hardware_uuid is required")
	}
	if update.OsqueryNodeKey == "" {
		return HostDetailUpdate{}, errors.New("osquery_node_key is required")
	}
	return update, nil
}

func upsertOrbitEnrollParams(params EnrollParams) sqlc.UpsertHostOnOrbitEnrollParams {
	return sqlc.UpsertHostOnOrbitEnrollParams{
		HardwareUUID:   params.HardwareUUID,
		DisplayName:    displayNameFor(params),
		Hostname:       params.Hostname,
		ComputerName:   params.ComputerName,
		HardwareSerial: params.HardwareSerial,
		HardwareModel:  params.HardwareModel,
		Platform:       params.Platform,
		PlatformLike:   params.PlatformLike,
		OrbitNodeKey:   params.OrbitNodeKey,
	}
}

func upsertOsqueryEnrollParams(update HostDetailUpdate) sqlc.UpsertHostOnOsqueryEnrollParams {
	return sqlc.UpsertHostOnOsqueryEnrollParams{
		HardwareUUID:     update.HardwareUUID,
		DisplayName:      displayNameFromValues(update.HardwareUUID, update.Hostname, update.ComputerName),
		Hostname:         update.Hostname,
		ComputerName:     update.ComputerName,
		HardwareSerial:   update.HardwareSerial,
		HardwareModel:    update.HardwareModel,
		HardwareVersion:  update.HardwareVersion,
		OSName:           update.OSName,
		OSVersion:        update.OSVersion,
		OSBuild:          update.OSBuild,
		Platform:         update.Platform,
		PlatformLike:     update.PlatformLike,
		OsqueryVersion:   update.OsqueryVersion,
		OsqueryNodeKey:   update.OsqueryNodeKey,
		OrbitVersion:     update.OrbitVersion,
		CPUBrand:         update.CPUBrand,
		CPULogicalCores:  update.CPULogicalCores,
		CPUPhysicalCores: update.CPUPhysicalCores,
		PhysicalMemory:   update.PhysicalMemory,
		HardwareVendor:   update.HardwareVendor,
		KernelVersion:    update.KernelVersion,
	}
}

type hostRecord struct {
	ID                      int64
	HardwareUUID            string
	DisplayName             string
	Hostname                string
	ComputerName            string
	HardwareSerial          string
	HardwareModel           string
	HardwareVersion         string
	OSName                  string
	Platform                string
	PlatformLike            string
	OSVersion               string
	OSBuild                 string
	OsqueryVersion          string
	OrbitVersion            string
	CPUType                 string
	CPUSubtype              string
	OrbitNodeKey            string
	OsqueryNodeKey          string
	CPUBrand                string
	CPULogicalCores         int
	CPUPhysicalCores        int
	PhysicalMemory          int64
	HardwareVendor          string
	KernelVersion           string
	UptimeSeconds           *int64
	LastRestartedAt         *time.Time
	DiskSpaceAvailableBytes *int64
	DiskSpaceTotalBytes     *int64
	PublicIP                *netip.Addr
	PrimaryIP               *netip.Addr
	PrimaryMAC              string
	DistributedInterval     *int32
	ConfigTLSRefresh        *int32
	EnrolledAt              *time.Time
	LastSeenAt              *time.Time
	DetailUpdatedAt         *time.Time
	DetailQueryHash         string
	LabelUpdatedAt          *time.Time
	SoftwareUpdatedAt       *time.Time
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

func hostFromRecord(row hostRecord) *Host {
	return &Host{
		ID:                      row.ID,
		HardwareUUID:            row.HardwareUUID,
		DisplayName:             row.DisplayName,
		Hostname:                row.Hostname,
		ComputerName:            row.ComputerName,
		HardwareSerial:          row.HardwareSerial,
		HardwareModel:           row.HardwareModel,
		HardwareVersion:         row.HardwareVersion,
		OSName:                  row.OSName,
		Platform:                row.Platform,
		PlatformLike:            row.PlatformLike,
		OSVersion:               row.OSVersion,
		OSBuild:                 row.OSBuild,
		OsqueryVersion:          row.OsqueryVersion,
		OrbitVersion:            row.OrbitVersion,
		CPUType:                 row.CPUType,
		CPUSubtype:              row.CPUSubtype,
		OrbitNodeKey:            row.OrbitNodeKey,
		OsqueryNodeKey:          row.OsqueryNodeKey,
		CPUBrand:                row.CPUBrand,
		CPULogicalCores:         row.CPULogicalCores,
		CPUPhysicalCores:        row.CPUPhysicalCores,
		PhysicalMemory:          row.PhysicalMemory,
		HardwareVendor:          row.HardwareVendor,
		KernelVersion:           row.KernelVersion,
		UptimeSeconds:           row.UptimeSeconds,
		LastRestartedAt:         row.LastRestartedAt,
		DiskSpaceAvailableBytes: row.DiskSpaceAvailableBytes,
		DiskSpaceTotalBytes:     row.DiskSpaceTotalBytes,
		PublicIP:                row.PublicIP,
		PrimaryIP:               row.PrimaryIP,
		PrimaryMAC:              row.PrimaryMAC,
		DistributedInterval:     row.DistributedInterval,
		ConfigTLSRefresh:        row.ConfigTLSRefresh,
		EnrolledAt:              row.EnrolledAt,
		LastSeenAt:              row.LastSeenAt,
		DetailUpdatedAt:         row.DetailUpdatedAt,
		DetailQueryHash:         row.DetailQueryHash,
		LabelUpdatedAt:          row.LabelUpdatedAt,
		SoftwareUpdatedAt:       row.SoftwareUpdatedAt,
		CreatedAt:               row.CreatedAt,
		UpdatedAt:               row.UpdatedAt,
	}
}

// HostIDString formats a database host ID for API responses.
func HostIDString(id int64) string {
	return strconv.FormatInt(id, 10)
}

// displayNameFor picks the most user-friendly identifier from enroll params.
func displayNameFor(p EnrollParams) string {
	return displayNameFromValues(p.HardwareUUID, p.Hostname, p.ComputerName)
}

func displayNameFromValues(hardwareUUID, hostname, computerName string) string {
	if v := strings.TrimSpace(computerName); v != "" {
		return v
	}
	if v := strings.TrimSpace(hostname); v != "" {
		return v
	}
	return strings.TrimSpace(hardwareUUID)
}
