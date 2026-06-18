// Package mdp manages Munki distribution points: ordered mirror nodes that
// Woodstar redirects Munki clients to for package installers.
//
// Woodstar owns policy, package metadata, and the files. A distribution point
// pulls the desired installers, verifies them, reports package mirror state,
// and serves them under a per-DP grant. This package owns the admin resource,
// the MDP-facing control/content protocol, and the selection that the Munki client
// delivery path consults. The worker that runs near clients lives under
// mdp/worker and shares only the grant leaf.
package mdp

import (
	"fmt"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/humaschema"
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
	return humaschema.StringEnum(packageStatusValues...)
}

// Presence reports whether a distribution point currently holds a live
// WebSocket connection. The connection hub implements it.
type Presence interface {
	Online(id int64) bool
}

// DistributionPoint is the admin view of one ordered mirror node. The per-DP
// key is never part of this model: it is revealed once on create and rotate,
// read by the resolver to sign grants, and matched for worker bearer auth.
type DistributionPoint struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Enabled       bool      `json:"enabled"`
	Position      int32     `json:"position"`
	ClientCIDRs   []string  `json:"client_cidrs"`
	ClientBaseURL string    `json:"client_base_url"`
	Online        bool      `json:"online"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// DistributionPointDetail adds the per-package mirror state to the admin view.
type DistributionPointDetail struct {
	DistributionPoint

	Packages []PackageState `json:"packages"`
}

// PackageState is one desired package's state on a distribution point.
type PackageState struct {
	PackageID   int64         `json:"package_id"`
	DisplayName string        `json:"display_name"`
	Version     string        `json:"version"`
	IconURL     string        `json:"icon_url,omitempty"`
	Status      PackageStatus `json:"status"`
	Error       string        `json:"error,omitempty"`
}

// DistributionPointMutation is the admin-writable surface of a distribution
// point. Position, the key, and worker-reported fields are not set here.
type DistributionPointMutation struct {
	Name          string   `json:"name"`
	Enabled       bool     `json:"enabled"`
	ClientCIDRs   []string `json:"client_cidrs"`
	ClientBaseURL string   `json:"client_base_url"`
}

// ResolvedPoint is the selection result: the identity and secret needed to mint
// a grant and redirect a client, without exposing the full admin row.
type ResolvedPoint struct {
	ID            int64
	Key           string
	ClientBaseURL string
}

// DesiredPackage is one installer a distribution point should mirror. It is the
// worker's whole world for a package: a stable id, a filename, and the bytes to
// verify against.
type DesiredPackage struct {
	PackageID   int64  `json:"package_id"`
	Filename    string `json:"filename"`
	SHA256      string `json:"sha256"`
	SizeBytes   int64  `json:"size_bytes"`
	DisplayName string `json:"display_name"`
	Version     string `json:"version"`
}

// StateReport is a worker's package mirror state, sent after each desired-set
// reconciliation.
type StateReport struct {
	Packages []ReportedPackage
}

// ReportedPackage is one desired package's state on a worker.
type ReportedPackage struct {
	PackageID int64
	SHA256    string
	Status    PackageStatus
	Error     string
}

// Validate enforces the write rules a malformed row would otherwise push into
// the resolver's inet cast or the redirect URL.
func (m DistributionPointMutation) Validate() error {
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("%w: name is required", dbutil.ErrInvalidInput)
	}
	for _, cidr := range m.ClientCIDRs {
		if _, err := netip.ParsePrefix(cidr); err != nil {
			return fmt.Errorf("%w: client_cidrs %q is not a CIDR", dbutil.ErrInvalidInput, cidr)
		}
	}
	if m.ClientBaseURL != "" {
		parsed, err := url.Parse(m.ClientBaseURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return fmt.Errorf("%w: client_base_url must be an http or https URL", dbutil.ErrInvalidInput)
		}
	}
	return nil
}

func online(presence Presence, id int64) bool {
	return presence != nil && presence.Online(id)
}

func munkiIconURL(iconObjectID *int64) string {
	if iconObjectID == nil {
		return ""
	}
	return fmt.Sprintf("/api/munki/icons/%d/content", *iconObjectID)
}
