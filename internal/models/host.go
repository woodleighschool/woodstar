package models

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
)

// Host is an enrolled Mac.
type Host struct {
	ID               int64
	HardwareUUID     string
	DisplayName      string
	Hostname         string
	ComputerName     string
	HardwareSerial   string
	HardwareModel    string
	Platform         string
	PlatformLike     string
	OSVersion        string
	OsqueryVersion   string
	OrbitVersion     string
	OrbitNodeKey     string
	OsqueryNodeKey   string
	CPUBrand         string
	CPULogicalCores  int
	CPUPhysicalCores int
	PhysicalMemory   int64
	HardwareVendor   string
	KernelVersion    string
	EnrolledAt       *time.Time
	LastSeenAt       *time.Time
	DetailUpdatedAt  *time.Time
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

	Platform        string
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
	HardwareUUID     string
	Hostname         string
	ComputerName     string
	HardwareSerial   string
	HardwareModel    string
	OSVersion        string
	Platform         string
	PlatformLike     string
	KernelVersion    string
	HardwareVendor   string
	OrbitVersion     string
	CPUBrand         string
	CPULogicalCores  int
	CPUPhysicalCores int
	PhysicalMemory   int64
	OsqueryVersion   string
	OsqueryNodeKey   string
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
	args = append(args, int32(params.PerPage), int32(params.Page*params.PerPage))
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
			&row.Platform,
			&row.PlatformLike,
			&row.OSVersion,
			&row.OsqueryVersion,
			&row.OrbitVersion,
			&row.OrbitNodeKey,
			&row.OsqueryNodeKey,
			&row.CPUBrand,
			&row.CPULogicalCores,
			&row.CPUPhysicalCores,
			&row.PhysicalMemory,
			&row.HardwareVendor,
			&row.KernelVersion,
			&row.EnrolledAt,
			&row.LastSeenAt,
			&row.DetailUpdatedAt,
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
	directionSQL := "ASC"
	if direction == "desc" {
		directionSQL = "DESC"
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
	h.platform,
	h.platform_like,
	h.os_version,
	h.osquery_version,
	h.orbit_version,
	h.orbit_node_key,
	h.osquery_node_key,
	h.cpu_brand,
	h.cpu_logical_cores,
	h.cpu_physical_cores,
	h.physical_memory,
	h.hardware_vendor,
	h.kernel_version,
	h.enrolled_at,
	h.last_seen_at,
	h.detail_updated_at,
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
		ID:               hostID,
		Hostname:         update.Hostname,
		ComputerName:     update.ComputerName,
		HardwareSerial:   update.HardwareSerial,
		HardwareModel:    update.HardwareModel,
		OSVersion:        update.OSVersion,
		Platform:         update.Platform,
		PlatformLike:     update.PlatformLike,
		OsqueryVersion:   update.OsqueryVersion,
		OrbitVersion:     update.OrbitVersion,
		CPUBrand:         update.CPUBrand,
		CPULogicalCores:  int32(update.CPULogicalCores),
		CPUPhysicalCores: int32(update.CPUPhysicalCores),
		PhysicalMemory:   update.PhysicalMemory,
		HardwareVendor:   update.HardwareVendor,
		KernelVersion:    update.KernelVersion,
	})
}

// MarkDetailFresh records that all built-in detail queries completed.
func (s *HostStore) MarkDetailFresh(ctx context.Context, hostID int64) error {
	return s.q.MarkHostDetailFresh(ctx, sqlc.MarkHostDetailFreshParams{ID: hostID})
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
		OSVersion:        update.OSVersion,
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
	ID               int64
	HardwareUUID     string
	DisplayName      string
	Hostname         string
	ComputerName     string
	HardwareSerial   string
	HardwareModel    string
	Platform         string
	PlatformLike     string
	OSVersion        string
	OsqueryVersion   string
	OrbitVersion     string
	OrbitNodeKey     string
	OsqueryNodeKey   string
	CPUBrand         string
	CPULogicalCores  int
	CPUPhysicalCores int
	PhysicalMemory   int64
	HardwareVendor   string
	KernelVersion    string
	EnrolledAt       *time.Time
	LastSeenAt       *time.Time
	DetailUpdatedAt  *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func hostFromRecord(row hostRecord) *Host {
	return &Host{
		ID:               row.ID,
		HardwareUUID:     row.HardwareUUID,
		DisplayName:      row.DisplayName,
		Hostname:         row.Hostname,
		ComputerName:     row.ComputerName,
		HardwareSerial:   row.HardwareSerial,
		HardwareModel:    row.HardwareModel,
		Platform:         row.Platform,
		PlatformLike:     row.PlatformLike,
		OSVersion:        row.OSVersion,
		OsqueryVersion:   row.OsqueryVersion,
		OrbitVersion:     row.OrbitVersion,
		OrbitNodeKey:     row.OrbitNodeKey,
		OsqueryNodeKey:   row.OsqueryNodeKey,
		CPUBrand:         row.CPUBrand,
		CPULogicalCores:  row.CPULogicalCores,
		CPUPhysicalCores: row.CPUPhysicalCores,
		PhysicalMemory:   row.PhysicalMemory,
		HardwareVendor:   row.HardwareVendor,
		KernelVersion:    row.KernelVersion,
		EnrolledAt:       row.EnrolledAt,
		LastSeenAt:       row.LastSeenAt,
		DetailUpdatedAt:  row.DetailUpdatedAt,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
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
