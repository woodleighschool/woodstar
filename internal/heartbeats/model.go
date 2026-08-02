package heartbeats

import (
	"net/netip"
	"time"
)

type Source string

const (
	SourceOrbit   Source = "orbit"
	SourceOsquery Source = "osquery"
	SourceMunki   Source = "munki"
	SourceSanta   Source = "santa"
)

func (s Source) valid() bool {
	switch s {
	case SourceOrbit, SourceOsquery, SourceMunki, SourceSanta:
		return true
	default:
		return false
	}
}

type Contact struct {
	RemoteIP  string
	UserAgent string
}

type Heartbeat struct {
	Source     Source      `json:"source"`
	LastSeenAt time.Time   `json:"last_seen_at"`
	RemoteIP   *netip.Addr `json:"remote_ip,omitempty"`
	UserAgent  string      `json:"user_agent"`
}
