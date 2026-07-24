// Package wire defines the Munki distribution point wire protocol.
package wire

import (
	"strconv"
	"strings"
)

const (
	// ProtocolVersion is the only MDP protocol version this binary supports.
	ProtocolVersion = 1

	// Subprotocol is the WebSocket subprotocol selected during an MDP upgrade.
	Subprotocol = "woodstar-mdp.v1"

	// ProtocolHeader identifies the protocol required by a rejected upgrade.
	ProtocolHeader = "Woodstar-MDP-Protocol"

	// BuildVersionHeader carries a Woodstar binary version during an upgrade.
	BuildVersionHeader = "Woodstar-Version"

	// MaxBuildVersionLength bounds diagnostic build-version values.
	MaxBuildVersionLength = 128

	subprotocolPrefix = "woodstar-mdp.v"
)

const (
	MessageHello      = "hello"
	MessageDesiredSet = "desired_set"
)

const (
	EventPackageSyncing = "package_syncing"
	EventPackageCurrent = "package_current"
	EventPackageError   = "package_error"
)

// ValidBuildVersion reports whether version is safe to exchange in the
// connection handshake.
func ValidBuildVersion(version string) bool {
	if version == "" || len(version) > MaxBuildVersionLength {
		return false
	}
	for i := range len(version) {
		if version[i] < '!' || version[i] > '~' {
			return false
		}
	}
	return true
}

// ParseSubprotocolVersion extracts the numeric version from an MDP WebSocket
// subprotocol.
func ParseSubprotocolVersion(subprotocol string) (int, bool) {
	value, ok := strings.CutPrefix(subprotocol, subprotocolPrefix)
	if !ok {
		return 0, false
	}
	version, err := strconv.Atoi(value)
	return version, err == nil && version >= 0
}

// PointIdentity is the distribution point identity assigned to a worker.
type PointIdentity struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// ServerMessage is one control message sent from central Woodstar to a worker.
type ServerMessage struct {
	Type              string           `json:"type"`
	DistributionPoint PointIdentity    `json:"distribution_point,omitzero"`
	Packages          []DesiredPackage `json:"packages,omitzero"`
}

// DesiredPackage identifies installer bytes a worker must mirror.
type DesiredPackage struct {
	PackageID int64  `json:"package_id"`
	Filename  string `json:"filename"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

// PackageEvent reports one worker mirror-state transition to central Woodstar.
type PackageEvent struct {
	Type      string `json:"type"`
	PackageID int64  `json:"package_id"`
	SHA256    string `json:"sha256,omitempty"`
	Error     string `json:"error,omitempty"`
}
