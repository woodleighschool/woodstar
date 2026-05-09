package models

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
)

// Host is an enrolled Mac. Labels, Users, and Batteries are populated only for
// callers that need the detail view.
type Host struct {
	sqlc.Host
	Labels    []Label
	Users     []HostUser
	Batteries []HostBattery
}

// HostUser is one local account reported by osquery.
type HostUser = sqlc.HostUser

// HostBattery is one battery reported by osquery.
type HostBattery = sqlc.HostBattery

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
// Only HardwareUUID and OrbitNodeKey are required.
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

// UpsertOnOrbitEnroll inserts a new host or refreshes an existing one keyed by
// hardware UUID. Re-enrollment overwrites the orbit node key so prior keys
// stop authenticating. Newly-enrolled hosts are added to the All Hosts label.
func (s *HostStore) UpsertOnOrbitEnroll(ctx context.Context, params EnrollParams) (*Host, error) {
	params, err := cleanOrbitEnrollParams(params)
	if err != nil {
		return nil, err
	}
	row, err := s.q.UpsertHostOnOrbitEnroll(ctx, sqlc.UpsertHostOnOrbitEnrollParams{
		HardwareUUID:   params.HardwareUUID,
		DisplayName:    displayName(params.HardwareUUID, params.Hostname, params.ComputerName),
		Hostname:       params.Hostname,
		ComputerName:   params.ComputerName,
		HardwareSerial: params.HardwareSerial,
		HardwareModel:  params.HardwareModel,
		Platform:       params.Platform,
		PlatformLike:   params.PlatformLike,
		OrbitNodeKey:   params.OrbitNodeKey,
	})
	if err != nil {
		return nil, err
	}
	if err := s.q.AddHostToAllHostsLabel(ctx, sqlc.AddHostToAllHostsLabelParams{HostID: row.ID}); err != nil {
		return nil, err
	}
	return &Host{Host: row}, nil
}

// UpsertOnOsqueryEnroll refreshes the osquery node key and host inventory.
// Newly-enrolled hosts are added to the All Hosts label.
func (s *HostStore) UpsertOnOsqueryEnroll(ctx context.Context, update HostDetailUpdate) (*Host, error) {
	update, err := cleanHostDetailUpdate(update)
	if err != nil {
		return nil, err
	}
	row, err := s.q.UpsertHostOnOsqueryEnroll(ctx, sqlc.UpsertHostOnOsqueryEnrollParams{
		HardwareUUID:     update.HardwareUUID,
		DisplayName:      displayName(update.HardwareUUID, update.Hostname, update.ComputerName),
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
	})
	if err != nil {
		return nil, err
	}
	if err := s.q.AddHostToAllHostsLabel(ctx, sqlc.AddHostToAllHostsLabelParams{HostID: row.ID}); err != nil {
		return nil, err
	}
	return &Host{Host: row}, nil
}

// List returns active hosts and the total count matching params.
func (s *HostStore) List(ctx context.Context, params HostListParams) ([]Host, int, error) {
	params = cleanHostListParams(params)
	listArgs := sqlc.ListHostsParams{
		Q:               params.Q,
		Platform:        params.Platform,
		Status:          params.Status,
		LabelID:         params.LabelID,
		SoftwareID:      params.SoftwareID,
		SoftwareTitleID: params.SoftwareTitleID,
		OrderKey:        params.OrderKey,
		OrderDirection:  params.OrderDirection,
		LimitRows:       int32(params.PerPage),
		OffsetRows:      int32((params.Page - 1) * params.PerPage),
	}
	rows, err := s.q.ListHosts(ctx, listArgs)
	if err != nil {
		return nil, 0, err
	}
	count, err := s.q.CountHosts(ctx, sqlc.CountHostsParams{
		Q:               params.Q,
		Platform:        params.Platform,
		Status:          params.Status,
		LabelID:         params.LabelID,
		SoftwareID:      params.SoftwareID,
		SoftwareTitleID: params.SoftwareTitleID,
	})
	if err != nil {
		return nil, 0, err
	}
	hosts := make([]Host, len(rows))
	for i, row := range rows {
		hosts[i] = Host{Host: row}
	}
	return hosts, int(count), nil
}

// GetByID returns a single active host by database ID.
func (s *HostStore) GetByID(ctx context.Context, id int64) (*Host, error) {
	row, err := s.q.GetHostByID(ctx, sqlc.GetHostByIDParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &Host{Host: row}, nil
}

// GetByOrbitNodeKey returns an active host and refreshes last_seen_at.
func (s *HostStore) GetByOrbitNodeKey(ctx context.Context, nodeKey string) (*Host, error) {
	nodeKey = strings.TrimSpace(nodeKey)
	if nodeKey == "" {
		return nil, ErrNotFound
	}
	row, err := s.q.TouchHostByOrbitNodeKey(ctx, sqlc.TouchHostByOrbitNodeKeyParams{OrbitNodeKey: nodeKey})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &Host{Host: row}, nil
}

// GetByOsqueryNodeKey returns an active host and refreshes last_seen_at.
func (s *HostStore) GetByOsqueryNodeKey(ctx context.Context, nodeKey string) (*Host, error) {
	nodeKey = strings.TrimSpace(nodeKey)
	if nodeKey == "" {
		return nil, ErrNotFound
	}
	row, err := s.q.TouchHostByOsqueryNodeKey(ctx, sqlc.TouchHostByOsqueryNodeKeyParams{OsqueryNodeKey: nodeKey})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &Host{Host: row}, nil
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
	return s.q.ListHostUsers(ctx, sqlc.ListHostUsersParams{HostID: hostID})
}

// ListBatteries returns batteries reported for hostID.
func (s *HostStore) ListBatteries(ctx context.Context, hostID int64) ([]HostBattery, error) {
	return s.q.ListHostBatteries(ctx, sqlc.ListHostBatteriesParams{HostID: hostID})
}

// MarkDetailFresh records that all built-in detail queries completed.
func (s *HostStore) MarkDetailFresh(ctx context.Context, hostID int64, detailQueryHash string) error {
	return s.q.MarkHostDetailFresh(ctx, sqlc.MarkHostDetailFreshParams{ID: hostID, DetailQueryHash: detailQueryHash})
}

func cleanHostListParams(params HostListParams) HostListParams {
	params.ListParams = CleanListParams(params.ListParams)
	params.Status = strings.TrimSpace(params.Status)
	params.Platform = strings.TrimSpace(params.Platform)
	return params
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

// displayName picks the most user-friendly identifier from enrollment values.
func displayName(hardwareUUID, hostname, computerName string) string {
	if v := strings.TrimSpace(computerName); v != "" {
		return v
	}
	if v := strings.TrimSpace(hostname); v != "" {
		return v
	}
	return strings.TrimSpace(hardwareUUID)
}
