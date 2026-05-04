package models

import (
	"context"
	"errors"
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
	q *sqlc.Queries
}

// NewHostStore returns a host store backed by db.
func NewHostStore(db *database.DB) *HostStore {
	return &HostStore{q: db.Queries()}
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

// List returns all active hosts ordered most-recently-seen first.
func (s *HostStore) List(ctx context.Context) ([]Host, error) {
	rows, err := s.q.ListHosts(ctx)
	if err != nil {
		return nil, err
	}

	hosts := make([]Host, 0, len(rows))
	for _, row := range rows {
		hosts = append(hosts, *hostFromRecord(hostRecord(row)))
	}
	return hosts, nil
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
