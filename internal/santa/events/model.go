package events

import (
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

type ExecutionDecision string

type DecisionFilter string

const (
	ExecutionDecisionUnknown          ExecutionDecision = "unknown"
	ExecutionDecisionAllowUnknown     ExecutionDecision = "allow_unknown"
	ExecutionDecisionAllowBinary      ExecutionDecision = "allow_binary"
	ExecutionDecisionAllowCertificate ExecutionDecision = "allow_certificate"
	ExecutionDecisionAllowScope       ExecutionDecision = "allow_scope"
	ExecutionDecisionAllowTeamID      ExecutionDecision = "allow_teamid"
	ExecutionDecisionAllowSigningID   ExecutionDecision = "allow_signingid"
	ExecutionDecisionAllowCDHash      ExecutionDecision = "allow_cdhash"
	ExecutionDecisionBlockUnknown     ExecutionDecision = "block_unknown"
	ExecutionDecisionBlockBinary      ExecutionDecision = "block_binary"
	ExecutionDecisionBlockCertificate ExecutionDecision = "block_certificate"
	ExecutionDecisionBlockScope       ExecutionDecision = "block_scope"
	ExecutionDecisionBlockTeamID      ExecutionDecision = "block_teamid"
	ExecutionDecisionBlockSigningID   ExecutionDecision = "block_signingid"
	ExecutionDecisionBlockCDHash      ExecutionDecision = "block_cdhash"
	ExecutionDecisionBundleBinary     ExecutionDecision = "bundle_binary"

	DecisionFilterAllowed DecisionFilter = "allowed"
	DecisionFilterBlocked DecisionFilter = "blocked"
)

type EventListParams struct {
	dbutil.ListParams

	HostID    int64
	Decisions []DecisionFilter
	Since     *time.Time
}

type ExecutionEvent struct {
	ID              int64             `json:"id"`
	HostID          int64             `json:"host_id"`
	Executable      Executable        `json:"executable"`
	FilePath        string            `json:"file_path"`
	ExecutingUser   string            `json:"executing_user"`
	LoggedInUsers   []string          `json:"logged_in_users"`
	CurrentSessions []string          `json:"current_sessions"`
	Decision        ExecutionDecision `json:"decision"`
	OccurredAt      *time.Time        `json:"occurred_at,omitempty"`
	IngestedAt      time.Time         `json:"ingested_at"`
}

type ExecutionEventInput struct {
	FileSHA256           string
	FilePath             string
	FileName             string
	ExecutingUser        string
	ExecutionTimeSeconds float64
	LoggedInUsers        []string
	CurrentSessions      []string
	Decision             ExecutionDecision
	BundleID             string
	BundlePath           string
	SigningID            string
	TeamID               string
	CDHash               string
	Entitlements         []byte
	SigningChain         []CertificateInput
}

type CertificateInput struct {
	SHA256     string
	CommonName string
	Org        string
	OU         string
	ValidFrom  uint32
	ValidUntil uint32
}

type Executable struct {
	ID         int64  `json:"id"`
	SHA256     string `json:"sha256"`
	FileName   string `json:"file_name"`
	BundleID   string `json:"file_bundle_id"`
	BundlePath string `json:"file_bundle_path"`
	SigningID  string `json:"signing_id"`
	TeamID     string `json:"team_id"`
	CDHash     string `json:"cdhash"`
}
