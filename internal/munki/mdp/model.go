// Package mdp manages Munki distribution points: ordered mirror nodes that
// Woodstar redirects Munki clients to for package installers.
//
// Woodstar owns policy, package metadata, and the files. A distribution point
// pulls the desired installers, verifies them, reports package mirror state,
// and serves them under a per-DP grant. This package owns the admin resource
// and the selection that the Munki client delivery path consults. The worker
// control protocol lives under mdp/protocol; the worker that runs near clients
// lives under mdp/worker and shares only the grant leaf.
package mdp

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/openapischema"
	"github.com/woodleighschool/woodstar/internal/validation"
)

// PackageStatus is a distribution point's mirror state for one desired package.
type PackageStatus string

const (
	PackageStatusPending PackageStatus = "pending"
	PackageStatusSyncing PackageStatus = "syncing"
	PackageStatusCurrent PackageStatus = "current"
	PackageStatusError   PackageStatus = "error"
)

var packageStatusValues = []PackageStatus{
	PackageStatusPending,
	PackageStatusSyncing,
	PackageStatusCurrent,
	PackageStatusError,
}

func (PackageStatus) Schema(_ huma.Registry) *huma.Schema {
	return openapischema.StringEnum(packageStatusValues...)
}

// DistributionPoint is the admin view of one ordered mirror node. The per-DP
// key is never part of this model: it is revealed once on create and rotate,
// read by the resolver to sign grants, and matched for worker bearer auth.
type DistributionPoint struct {
	ID            int64                    `json:"id"`
	Name          string                   `json:"name"`
	Enabled       bool                     `json:"enabled"`
	Position      int32                    `json:"position"`
	ClientCIDRs   []string                 `json:"client_cidrs"`
	ClientBaseURL string                   `json:"client_base_url"`
	Worker        *DistributionPointWorker `json:"worker,omitempty"`
	CreatedAt     time.Time                `json:"created_at"`
	UpdatedAt     time.Time                `json:"updated_at"`
}

// DistributionPointWorker describes the latest worker state known by the
// current Woodstar process.
type DistributionPointWorker struct {
	Compatible      bool   `json:"compatible"`
	ProtocolVersion *int   `json:"protocol_version,omitempty"`
	BuildVersion    string `json:"build_version,omitempty"`
}

// DistributionPointDetail adds the per-package mirror state to the admin view.
type DistributionPointDetail struct {
	DistributionPoint

	Packages []PackageState `json:"packages"`
}

// PackageState is one desired package's state on a distribution point.
type PackageState struct {
	PackageID       int64         `json:"package_id"`
	SoftwareID      int64         `json:"software_id"`
	Name            string        `json:"name"`
	Version         string        `json:"version"`
	SoftwareIconURL string        `json:"software_icon_url,omitempty"`
	Status          PackageStatus `json:"status"`
	Error           string        `json:"error,omitempty"`
}

// DistributionPointMutation is the admin-writable surface of a distribution
// point. Position, the key, and worker-reported fields are not set here.
type DistributionPointMutation struct {
	Name          string   `json:"name"            validate:"required,notblank"      minLength:"1"`
	Enabled       bool     `json:"enabled"`
	ClientCIDRs   []string `json:"client_cidrs"    validate:"dive,cidr"`
	ClientBaseURL string   `json:"client_base_url" validate:"omitempty,https_origin"               format:"uri"`
}

// ResolvedPoint is the selection result: the identity and secret needed to mint
// a grant and redirect a client, without exposing the full admin row.
type ResolvedPoint struct {
	ID            int64
	Key           string
	ClientBaseURL string
}

// DesiredPackage is one installer a distribution point should mirror: a stable
// id, the filename to store it under, and the bytes to verify against. The
// worker fetches a fresh download URL per job, so none is carried here.
type DesiredPackage struct {
	PackageID int64
	Filename  string
	SHA256    string
	SizeBytes int64
}

func (m *DistributionPointMutation) validate() error {
	if err := validation.Struct(m); err != nil {
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	return nil
}

func (m *DistributionPointMutation) normalize() {
	m.Name = strings.TrimSpace(m.Name)
	if m.ClientCIDRs == nil {
		m.ClientCIDRs = []string{}
	}
	for i := range m.ClientCIDRs {
		m.ClientCIDRs[i] = strings.TrimSpace(m.ClientCIDRs[i])
	}
	m.ClientBaseURL = normalizeClientBaseURL(m.ClientBaseURL)
}

func normalizeClientBaseURL(value string) string {
	value = strings.TrimSpace(value)
	parsed, err := url.Parse(value)
	if err != nil {
		return value
	}
	if parsed.Path == "/" {
		parsed.Path = ""
	}
	return parsed.String()
}
