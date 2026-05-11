package hosts

import (
	"net/netip"
	"time"

	"github.com/woodleighschool/woodstar/internal/labels"
)

// Host is an enrolled Mac. Used for list rows and as the base of HostDetail.
type Host struct {
	ID                      int64       `json:"id"`
	HardwareUUID            string      `json:"hardware_uuid"`
	DisplayName             string      `json:"display_name"`
	Hostname                string      `json:"hostname"`
	ComputerName            string      `json:"computer_name"`
	HardwareSerial          string      `json:"hardware_serial"`
	HardwareModel           string      `json:"hardware_model"`
	HardwareVersion         string      `json:"hardware_version"`
	HardwareVendor          string      `json:"hardware_vendor"`
	OSName                  string      `json:"os_name"`
	OSVersion               string      `json:"os_version"`
	OSBuild                 string      `json:"os_build"`
	Platform                string      `json:"platform"`
	PlatformLike            string      `json:"platform_like"`
	OsqueryVersion          string      `json:"osquery_version"`
	OrbitVersion            string      `json:"orbit_version"`
	OrbitNodeKey            string      `json:"orbit_node_key"`
	OsqueryNodeKey          string      `json:"osquery_node_key"`
	CPUType                 string      `json:"cpu_type"`
	CPUSubtype              string      `json:"cpu_subtype"`
	CPUBrand                string      `json:"cpu_brand"`
	CPULogicalCores         int         `json:"cpu_logical_cores"`
	CPUPhysicalCores        int         `json:"cpu_physical_cores"`
	PhysicalMemory          int64       `json:"physical_memory"`
	KernelVersion           string      `json:"kernel_version"`
	UptimeSeconds           *int64      `json:"uptime_seconds"`
	LastRestartedAt         *time.Time  `json:"last_restarted_at"`
	DiskSpaceAvailableBytes *int64      `json:"disk_space_available_bytes"`
	DiskSpaceTotalBytes     *int64      `json:"disk_space_total_bytes"`
	PublicIP                *netip.Addr `json:"public_ip"`
	PrimaryIP               *netip.Addr `json:"primary_ip"`
	PrimaryMAC              string      `json:"primary_mac"`
	DistributedInterval     *int32      `json:"distributed_interval"`
	ConfigTLSRefresh        *int32      `json:"config_tls_refresh"`
	DetailQueryHash         string      `json:"detail_query_hash"`
	EnrolledAt              *time.Time  `json:"enrolled_at"`
	LastSeenAt              *time.Time  `json:"last_seen_at"`
	DetailUpdatedAt         *time.Time  `json:"detail_updated_at"`
	LabelUpdatedAt          *time.Time  `json:"label_updated_at"`
	SoftwareUpdatedAt       *time.Time  `json:"software_updated_at"`
	CreatedAt               time.Time   `json:"created_at"`
	UpdatedAt               time.Time   `json:"updated_at"`
	DeletedAt               *time.Time  `json:"deleted_at"`
}

// HostDetail is a host plus its loaded children.
type HostDetail struct {
	Host
	Labels    []labels.Label `json:"labels,omitempty"`
	Users     []HostUser     `json:"users,omitempty"`
	Batteries []HostBattery  `json:"batteries,omitempty"`
}

// HostUser is one local account reported by osquery.
type HostUser struct {
	ID          int64     `json:"id"`
	HostID      int64     `json:"host_id"`
	UID         string    `json:"uid"`
	Username    string    `json:"username"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Directory   string    `json:"directory"`
	Shell       string    `json:"shell"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// HostBattery is one battery reported by osquery.
type HostBattery struct {
	ID               int64     `json:"id"`
	HostID           int64     `json:"host_id"`
	SerialNumber     string    `json:"serial_number"`
	Manufacturer     string    `json:"manufacturer"`
	Model            string    `json:"model"`
	Chemistry        string    `json:"chemistry"`
	CycleCount       *int32    `json:"cycle_count"`
	Health           string    `json:"health"`
	DesignedCapacity *int32    `json:"designed_capacity"`
	MaxCapacity      *int32    `json:"max_capacity"`
	CurrentCapacity  *int32    `json:"current_capacity"`
	PercentRemaining *float64  `json:"percent_remaining"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// HostDeviceMapping is a user/device association observed for a host.
type HostDeviceMapping struct {
	ID        int64     `json:"id"`
	HostID    int64     `json:"host_id"`
	Email     string    `json:"email"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
