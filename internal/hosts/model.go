package hosts

import (
	"net/netip"
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/scope"
)

// ListParams filters host list results.
type ListParams struct {
	dbutil.ListParams

	Status          string
	Platform        string
	LabelID         int64
	SoftwareTitleID int64
	SoftwareID      int64
}

// DetailUpdate is inventory reported by osquery detail queries.
type DetailUpdate struct {
	HardwareUUID            string
	Hostname                string
	ComputerName            string
	HardwareSerial          string
	HardwareModel           string
	HardwareVersion         string
	OSName                  string
	OSVersion               string
	OSBuild                 string
	Platform                scope.Platform
	OsqueryPlatform         string
	OsqueryPlatformLike     string
	KernelVersion           string
	HardwareVendor          string
	OrbitVersion            string
	CPUType                 string
	CPUSubtype              string
	CPUBrand                string
	CPULogicalCores         int
	CPUPhysicalCores        int
	PhysicalMemory          int64
	OrbitNodeKey            string
	OsqueryVersion          string
	OsqueryNodeKey          string
	LastRestartedAt         *time.Time
	DiskSpaceAvailableBytes *int64
	DiskSpaceTotalBytes     *int64
	PublicIP                string
	PrimaryIP               string
	PrimaryMAC              string
	DistributedInterval     *int32
	ConfigTLSRefresh        *int32
}

// Host is an enrolled Mac. Used for list rows and as the base of HostDetail.
type Host struct {
	ID                      int64               `json:"id"`
	HardwareUUID            string              `json:"hardware_uuid"`
	DisplayName             string              `json:"display_name"`
	Hostname                string              `json:"hostname"`
	ComputerName            string              `json:"computer_name"`
	HardwareSerial          string              `json:"hardware_serial"`
	HardwareModel           string              `json:"hardware_model"`
	HardwareVersion         string              `json:"hardware_version"`
	HardwareVendor          string              `json:"hardware_vendor"`
	OSName                  string              `json:"os_name"`
	OSVersion               string              `json:"os_version"`
	OSBuild                 string              `json:"os_build"`
	Platform                scope.Platform      `json:"platform" enum:"unknown,darwin,windows,linux"`
	OsqueryPlatform         string              `json:"osquery_platform"`
	OsqueryPlatformLike     string              `json:"osquery_platform_like"`
	OsqueryVersion          string              `json:"osquery_version"`
	OrbitVersion            string              `json:"orbit_version"`
	OrbitNodeKey            string              `json:"-"`
	OsqueryNodeKey          string              `json:"-"`
	CPUType                 string              `json:"cpu_type"`
	CPUSubtype              string              `json:"cpu_subtype"`
	CPUBrand                string              `json:"cpu_brand"`
	CPULogicalCores         int                 `json:"cpu_logical_cores"`
	CPUPhysicalCores        int                 `json:"cpu_physical_cores"`
	PhysicalMemory          int64               `json:"physical_memory"`
	KernelVersion           string              `json:"kernel_version"`
	LastRestartedAt         *time.Time          `json:"last_restarted_at,omitempty"`
	DiskSpaceAvailableBytes *int64              `json:"disk_space_available_bytes,omitempty"`
	DiskSpaceTotalBytes     *int64              `json:"disk_space_total_bytes,omitempty"`
	PublicIP                *netip.Addr         `json:"public_ip,omitempty"`
	PrimaryIP               *netip.Addr         `json:"primary_ip,omitempty"`
	PrimaryMAC              string              `json:"primary_mac"`
	DistributedInterval     *int32              `json:"distributed_interval,omitempty"`
	ConfigTLSRefresh        *int32              `json:"config_tls_refresh,omitempty"`
	DetailQueryHash         string              `json:"-"`
	EnrolledAt              *time.Time          `json:"enrolled_at,omitempty"`
	LastSeenAt              *time.Time          `json:"last_seen_at,omitempty"`
	DetailUpdatedAt         *time.Time          `json:"detail_updated_at,omitempty"`
	LabelUpdatedAt          *time.Time          `json:"label_updated_at,omitempty"`
	SoftwareUpdatedAt       *time.Time          `json:"software_updated_at,omitempty"`
	CreatedAt               time.Time           `json:"created_at"`
	UpdatedAt               time.Time           `json:"updated_at"`
	DeviceMappings          []HostDeviceMapping `json:"device_mappings,omitempty"`
}

// HostDetail is a host plus its loaded children.
type HostDetail struct {
	Host
	Labels       []labels.Label    `json:"labels"`
	Users        []HostUser        `json:"users"`
	Batteries    []HostBattery     `json:"batteries"`
	Certificates []HostCertificate `json:"certificates"`
}

// HostUser is one local account reported by osquery.
type HostUser struct {
	ID          int64     `json:"-"`
	HostID      int64     `json:"-"`
	UID         string    `json:"uid"`
	Username    string    `json:"username"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Directory   string    `json:"directory"`
	Shell       string    `json:"shell"`
	CreatedAt   time.Time `json:"-"`
	UpdatedAt   time.Time `json:"-"`
}

// HostBattery is one battery reported by osquery.
type HostBattery struct {
	ID               int64     `json:"-"`
	HostID           int64     `json:"-"`
	SerialNumber     string    `json:"serial_number"`
	Manufacturer     string    `json:"manufacturer"`
	Model            string    `json:"model"`
	Chemistry        string    `json:"chemistry"`
	CycleCount       *int32    `json:"cycle_count,omitempty"`
	Health           string    `json:"health"`
	DesignedCapacity *int32    `json:"designed_capacity,omitempty"`
	MaxCapacity      *int32    `json:"max_capacity,omitempty"`
	CurrentCapacity  *int32    `json:"current_capacity,omitempty"`
	PercentRemaining *float64  `json:"percent_remaining,omitempty"`
	CreatedAt        time.Time `json:"-"`
	UpdatedAt        time.Time `json:"-"`
}

// HostCertificate is one system or user certificate reported by osquery.
type HostCertificate struct {
	ID                   int64           `json:"id"`
	HostID               int64           `json:"-"`
	SHA1                 string          `json:"-"`
	CommonName           string          `json:"common_name"`
	Subject              CertificateName `json:"subject"`
	Issuer               CertificateName `json:"issuer"`
	KeyAlgorithm         string          `json:"key_algorithm"`
	KeyStrength          *int32          `json:"key_strength,omitempty"`
	KeyUsage             string          `json:"key_usage"`
	SigningAlgorithm     string          `json:"signing_algorithm"`
	NotValidAfter        *time.Time      `json:"not_valid_after,omitempty"`
	NotValidBefore       *time.Time      `json:"not_valid_before,omitempty"`
	Serial               string          `json:"serial"`
	CertificateAuthority bool            `json:"certificate_authority"`
	Source               string          `json:"source"`
	Username             string          `json:"username"`
	Path                 string          `json:"-"`
	CreatedAt            time.Time       `json:"-"`
	UpdatedAt            time.Time       `json:"-"`
}

// CertificateName is the structured subject or issuer name for a certificate.
type CertificateName struct {
	Country            string `json:"country"`
	Organization       string `json:"organization"`
	OrganizationalUnit string `json:"organizational_unit"`
	CommonName         string `json:"common_name"`
}

// HostDeviceMapping is a user/device association observed for a host.
type HostDeviceMapping struct {
	ID        int64               `json:"-"`
	HostID    int64               `json:"-"`
	Email     string              `json:"email"`
	Source    DeviceMappingSource `json:"source"`
	CreatedAt time.Time           `json:"-"`
	UpdatedAt time.Time           `json:"-"`
}
