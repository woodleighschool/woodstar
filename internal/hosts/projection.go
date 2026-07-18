package hosts

import (
	"net/netip"
	"time"
)

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

func hostColumnsSQL() string {
	return `
	id,
	hardware_uuid,
	display_name,
	hostname,
	computer_name,
	hardware_serial,
	hardware_model_identifier,
	hardware_vendor,
	os_name,
	os_version,
	os_build,
	os_platform,
	osquery_version,
	orbit_version,
	orbit_node_key,
	osquery_node_key,
	enrollment_agent,
	cpu_type,
	cpu_subtype,
	cpu_brand,
	cpu_logical_cores,
	cpu_physical_cores,
	memory_bytes,
	os_kernel_version,
	last_restarted_at,
	boot_volume_available_bytes,
	boot_volume_total_bytes,
	last_remote_ip,
	primary_ip,
	primary_mac,
	osquery_distributed_interval_seconds,
	osquery_config_refresh_seconds,
	inventory_query_hash,
	enrolled_at,
	last_seen_at,
	inventory_updated_at,
	created_at,
	updated_at`
}

func hostSelectSQL() string {
	return `SELECT` + hostColumnsSQL() + `
FROM hosts`
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
		PrimaryUserSources: []HostPrimaryUserSource{},
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

func statusFromLastSeen(lastSeen *time.Time, now time.Time) HostStatus {
	if lastSeen == nil || lastSeen.Before(now.Add(-hostOnlineWindow)) {
		return HostStatusOffline
	}
	return HostStatusOnline
}
