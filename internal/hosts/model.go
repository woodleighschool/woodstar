package hosts

import (
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/openapischema"
	"github.com/woodleighschool/woodstar/internal/validation"
)

// HostListParams filters host list results.
type HostListParams struct {
	dbutil.ListParams

	Status          HostStatus `validate:"omitempty,oneof=online offline"`
	LabelID         int64      `validate:"gte=0"`
	SoftwareTitleID int64      `validate:"gte=0"`
	SoftwareID      int64      `validate:"gte=0"`
	IDs             []int64    `validate:"unique,dive,gt=0"`
}

func (params *HostListParams) normalize() {
	params.ListParams = dbutil.NormalizeListParams(params.ListParams)
	params.Status = HostStatus(strings.TrimSpace(string(params.Status)))
}

func (params *HostListParams) validate() error {
	if err := validation.Struct(params); err != nil {
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	return nil
}

// InventoryUpdate is inventory reported by enrolling agents and osquery detail queries.
type InventoryUpdate struct {
	Hostname        string
	ComputerName    string
	Hardware        HostHardware
	OS              HostOS
	Storage         HostStorage
	Network         InventoryNetwork
	Agents          HostAgents
	Timestamps      InventoryTimestamps
	OrbitNodeKey    string
	OsqueryNodeKey  string
	EnrollmentAgent string
}

// InventoryNetwork is network inventory before PostgreSQL inet parsing.
type InventoryNetwork struct {
	PrimaryIP    string
	PrimaryMAC   string
	LastRemoteIP string
}

// InventoryTimestamps carries timestamp inventory updates.
type InventoryTimestamps struct {
	LastRestartedAt *time.Time
}

// HostStatus is whether a host has checked in during the online window.
type HostStatus string

const (
	HostStatusOnline  HostStatus = "online"
	HostStatusOffline HostStatus = "offline"
)

var hostStatusValues = []HostStatus{HostStatusOnline, HostStatusOffline}

// Schema returns the OpenAPI schema for HostStatus.
func (HostStatus) Schema(_ huma.Registry) *huma.Schema {
	return openapischema.StringEnum(hostStatusValues...)
}

// Host is an enrolled Mac. Used for list rows and as the base of HostDetail.
type Host struct {
	ID                 int64                   `json:"id"`
	DisplayName        string                  `json:"display_name"`
	Status             HostStatus              `json:"status"`
	Hostname           string                  `json:"hostname"`
	ComputerName       string                  `json:"computer_name"`
	Enrollment         HostEnrollment          `json:"enrollment"`
	Hardware           HostHardware            `json:"hardware"`
	OS                 HostOS                  `json:"os"`
	Storage            HostStorage             `json:"storage"`
	Network            HostNetwork             `json:"network"`
	Agents             HostAgents              `json:"agents"`
	PrimaryUser        *HostPrimaryUser        `json:"primary_user,omitempty"`
	PrimaryUserSources []HostPrimaryUserSource `json:"primary_user_sources"`
	Timestamps         HostTimestamps          `json:"timestamps"`

	OrbitNodeKey       string `json:"-"`
	OsqueryNodeKey     string `json:"-"`
	InventoryQueryHash string `json:"-"`
}

type HostEnrollment struct {
	Agent      string     `json:"agent"`
	EnrolledAt *time.Time `json:"enrolled_at,omitempty"`
}

type HostHardware struct {
	UUID            string  `json:"uuid"`
	Serial          string  `json:"serial"`
	Vendor          string  `json:"vendor"`
	ModelIdentifier string  `json:"model_identifier"`
	MemoryBytes     int64   `json:"memory_bytes"`
	CPU             HostCPU `json:"cpu"`
}

type HostCPU struct {
	Architecture  string `json:"architecture"`
	Subtype       string `json:"subtype"`
	Brand         string `json:"brand"`
	PhysicalCores int32  `json:"physical_cores"`
	LogicalCores  int32  `json:"logical_cores"`
}

type HostOS struct {
	Platform      string `json:"platform"`
	Name          string `json:"name"`
	Version       string `json:"version"`
	Build         string `json:"build"`
	KernelVersion string `json:"kernel_version"`
}

type HostStorage struct {
	BootVolume HostBootVolume `json:"boot_volume"`
}

type HostBootVolume struct {
	AvailableBytes *int64 `json:"available_bytes,omitempty"`
	TotalBytes     *int64 `json:"total_bytes,omitempty"`
}

type HostNetwork struct {
	PrimaryIP    *netip.Addr `json:"primary_ip,omitempty"`
	PrimaryMAC   string      `json:"primary_mac"`
	LastRemoteIP *netip.Addr `json:"last_remote_ip,omitempty"`
}

type HostAgents struct {
	Osquery HostOsqueryAgent `json:"osquery"`
	Orbit   HostOrbitAgent   `json:"orbit"`
}

type HostOsqueryAgent struct {
	Version                    string `json:"version"`
	DistributedIntervalSeconds *int32 `json:"distributed_interval_seconds,omitempty"`
	ConfigRefreshSeconds       *int32 `json:"config_refresh_seconds,omitempty"`
}

type HostOrbitAgent struct {
	Version string `json:"version"`
}

type HostTimestamps struct {
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	LastSeenAt         *time.Time `json:"last_seen_at,omitempty"`
	InventoryUpdatedAt *time.Time `json:"inventory_updated_at,omitempty"`
	LastRestartedAt    *time.Time `json:"last_restarted_at,omitempty"`
}

// HostDetail is a host plus its loaded children.
type HostDetail struct {
	Host

	Labels       []labels.Label    `json:"labels"`
	Users        []HostUser        `json:"users"`
	Batteries    []HostBattery     `json:"batteries"`
	Certificates []HostCertificate `json:"certificates"`
}

type HostPrimaryUser struct {
	Email      string            `json:"email"`
	Username   string            `json:"username"`
	Name       string            `json:"name"`
	Department string            `json:"department"`
	Groups     []string          `json:"groups"`
	Source     PrimaryUserSource `json:"source"`
}

// HostUser is one local account reported by osquery.
type HostUser struct {
	ID          int64  `json:"-"           db:"id"`
	HostID      int64  `json:"-"           db:"host_id"`
	UID         string `json:"uid"         db:"uid"`
	Username    string `json:"username"    db:"username"`
	Type        string `json:"type"        db:"type"`
	Description string `json:"description" db:"description"`
	Directory   string `json:"directory"   db:"directory"`
	Shell       string `json:"shell"       db:"shell"`
}

// HostBattery is one battery reported by osquery.
type HostBattery struct {
	ID               int64    `json:"-"                           db:"id"`
	HostID           int64    `json:"-"                           db:"host_id"`
	SerialNumber     string   `json:"serial_number"               db:"serial_number"`
	Manufacturer     string   `json:"manufacturer"                db:"manufacturer"`
	Model            string   `json:"model"                       db:"model"`
	Chemistry        string   `json:"chemistry"                   db:"chemistry"`
	CycleCount       *int32   `json:"cycle_count,omitempty"       db:"cycle_count"`
	Health           string   `json:"health"                      db:"health"`
	DesignedCapacity *int32   `json:"designed_capacity,omitempty" db:"designed_capacity"`
	MaxCapacity      *int32   `json:"max_capacity,omitempty"      db:"max_capacity"`
	CurrentCapacity  *int32   `json:"current_capacity,omitempty"  db:"current_capacity"`
	PercentRemaining *float64 `json:"percent_remaining,omitempty" db:"percent_remaining"`
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
}

// CertificateName is the structured subject or issuer name for a certificate.
type CertificateName struct {
	Country            string `json:"country"`
	Organization       string `json:"organization"`
	OrganizationalUnit string `json:"organizational_unit"`
	CommonName         string `json:"common_name"`
}

// HostPrimaryUserSource is a primary-user observation or manual override for a host.
type HostPrimaryUserSource struct {
	ID        int64             `json:"-"`
	HostID    int64             `json:"-"`
	Email     string            `json:"email"`
	Source    PrimaryUserSource `json:"source"`
	CreatedAt time.Time         `json:"-"`
	UpdatedAt time.Time         `json:"-"`
}
