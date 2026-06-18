package events

import (
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/humaschema"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
)

// ExecutionDecision is Santa's policy decision for an executed binary.
type ExecutionDecision string

// DecisionFilter is an execution-event filter value accepted by the admin API.
type DecisionFilter string

// FileAccessDecision is Santa's policy decision for a file-access event.
type FileAccessDecision string

// SigningStatus is Santa's code-signing status for an executable.
type SigningStatus string

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

	FileAccessDecisionUnknown                FileAccessDecision = "unknown"
	FileAccessDecisionDenied                 FileAccessDecision = "denied"
	FileAccessDecisionDeniedInvalidSignature FileAccessDecision = "denied_invalid_signature"
	FileAccessDecisionAuditOnly              FileAccessDecision = "audit_only"

	SigningStatusUnspecified SigningStatus = "unspecified"
	SigningStatusUnsigned    SigningStatus = "unsigned"
	SigningStatusInvalid     SigningStatus = "invalid"
	SigningStatusAdhoc       SigningStatus = "adhoc"
	SigningStatusDevelopment SigningStatus = "development"
	SigningStatusProduction  SigningStatus = "production"
)

var ExecutionDecisionValues = []ExecutionDecision{
	ExecutionDecisionUnknown,
	ExecutionDecisionAllowUnknown,
	ExecutionDecisionAllowBinary,
	ExecutionDecisionAllowCertificate,
	ExecutionDecisionAllowScope,
	ExecutionDecisionAllowTeamID,
	ExecutionDecisionAllowSigningID,
	ExecutionDecisionAllowCDHash,
	ExecutionDecisionBlockUnknown,
	ExecutionDecisionBlockBinary,
	ExecutionDecisionBlockCertificate,
	ExecutionDecisionBlockScope,
	ExecutionDecisionBlockTeamID,
	ExecutionDecisionBlockSigningID,
	ExecutionDecisionBlockCDHash,
	ExecutionDecisionBundleBinary,
}

var DecisionFilterValues = decisionFilterValues()

var FileAccessDecisionValues = []FileAccessDecision{
	FileAccessDecisionUnknown,
	FileAccessDecisionDenied,
	FileAccessDecisionDeniedInvalidSignature,
	FileAccessDecisionAuditOnly,
}

var SigningStatusValues = []SigningStatus{
	SigningStatusUnspecified,
	SigningStatusUnsigned,
	SigningStatusInvalid,
	SigningStatusAdhoc,
	SigningStatusDevelopment,
	SigningStatusProduction,
}

func decisionFilterValues() []DecisionFilter {
	values := []DecisionFilter{DecisionFilterAllowed, DecisionFilterBlocked}
	for _, decision := range ExecutionDecisionValues {
		values = append(values, DecisionFilter(decision))
	}
	return values
}

func (ExecutionDecision) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(ExecutionDecisionValues...)
}

func (DecisionFilter) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(DecisionFilterValues...)
}

func (FileAccessDecision) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(FileAccessDecisionValues...)
}

func (SigningStatus) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(SigningStatusValues...)
}

// EventListParams contains filters shared by Santa event list endpoints.
type EventListParams struct {
	dbutil.ListParams

	HostID int64
	Since  *time.Time
}

// ExecutionEventListParams contains filters for execution-event lists.
type ExecutionEventListParams struct {
	EventListParams

	Decisions []DecisionFilter
	User      string
}

// FileAccessEventListParams contains filters for file-access event lists.
type FileAccessEventListParams struct {
	EventListParams

	Decisions []FileAccessDecision
}

// ExecutionEvent is an observed Santa execution decision.
type ExecutionEvent struct {
	ID              int64             `json:"id"`
	HostID          int64             `json:"host_id"`
	Host            HostSummary       `json:"host"`
	Executable      Executable        `json:"executable"`
	FilePath        string            `json:"file_path"`
	ExecutingUser   string            `json:"executing_user"`
	PID             int32             `json:"pid"`
	PPID            int32             `json:"ppid"`
	ParentName      string            `json:"parent_name"`
	LoggedInUsers   []string          `json:"logged_in_users"`
	CurrentSessions []string          `json:"current_sessions"`
	Decision        ExecutionDecision `json:"decision"`
	OccurredAt      time.Time         `json:"occurred_at"`
	IngestedAt      time.Time         `json:"ingested_at"`
}

// HostSummary is the host identity attached to Santa event rows.
type HostSummary struct {
	ID              int64                             `json:"id"`
	DisplayName     string                            `json:"display_name"`
	Hostname        string                            `json:"hostname"`
	ComputerName    string                            `json:"computer_name"`
	Hardware        HostSummaryHardware               `json:"hardware"`
	SantaMachineID  string                            `json:"santa_machine_id"`
	SantaVersion    string                            `json:"santa_version"`
	SantaClientMode configurations.ReportedClientMode `json:"santa_client_mode"`
}

type HostSummaryHardware struct {
	Serial          string `json:"serial"`
	ModelIdentifier string `json:"model_identifier"`
}

// ExecutionEventInput is a Santa execution event ready for persistence.
type ExecutionEventInput struct {
	FileSHA256              string
	FilePath                string
	FileName                string
	ExecutingUser           string
	OccurredAt              time.Time
	LoggedInUsers           []string
	CurrentSessions         []string
	Decision                ExecutionDecision
	BundleID                string
	BundlePath              string
	BundleExecutableRelPath string
	BundleName              string
	BundleVersion           string
	BundleVersionString     string
	BundleHash              string
	BundleHashMillis        int32
	BundleBinaryCount       int32
	PID                     int32
	PPID                    int32
	ParentName              string
	SigningID               string
	TeamID                  string
	CDHash                  string
	CodesigningFlags        uint32
	SigningStatus           SigningStatus
	SecureSigningTime       time.Time
	SigningTime             time.Time
	Entitlements            []byte
	SigningChain            []CertificateInput
}

// CertificateInput is a certificate entry reported by Santa sync.
type CertificateInput struct {
	SHA256     string
	CommonName string
	Org        string
	OU         string
	ValidFrom  uint32
	ValidUntil uint32
}

// Executable is metadata for the binary involved in an execution event.
type Executable struct {
	ID                      int64               `json:"id"`
	SHA256                  string              `json:"sha256"`
	FileName                string              `json:"file_name"`
	BundleID                string              `json:"file_bundle_id"`
	BundlePath              string              `json:"file_bundle_path"`
	BundleExecutableRelPath string              `json:"file_bundle_executable_rel_path"`
	BundleName              string              `json:"file_bundle_name"`
	BundleVersion           string              `json:"file_bundle_version"`
	BundleVersionString     string              `json:"file_bundle_version_string"`
	BundleHash              string              `json:"file_bundle_hash"`
	BundleHashMillis        int32               `json:"file_bundle_hash_millis"`
	BundleBinaryCount       int32               `json:"file_bundle_binary_count"`
	SigningID               string              `json:"signing_id"`
	TeamID                  string              `json:"team_id"`
	CDHash                  string              `json:"cdhash"`
	CodesigningFlags        uint32              `json:"codesigning_flags"`
	SigningStatus           SigningStatus       `json:"signing_status"`
	SecureSigningTime       *time.Time          `json:"secure_signing_time,omitempty"`
	SigningTime             *time.Time          `json:"signing_time,omitempty"`
	Entitlements            map[string]any      `json:"entitlements,omitempty"`
	SigningChain            []SigningChainEntry `json:"signing_chain,omitempty"`
}

// SigningChainEntry is a certificate entry exposed through the admin API.
type SigningChainEntry struct {
	SHA256             string     `json:"sha256"`
	CommonName         string     `json:"common_name,omitempty"`
	Organization       string     `json:"organization,omitempty"`
	OrganizationalUnit string     `json:"organizational_unit,omitempty"`
	ValidFrom          *time.Time `json:"valid_from,omitempty"`
	ValidUntil         *time.Time `json:"valid_until,omitempty"`
}

// ProcessInput is a process-chain entry from Santa sync.
type ProcessInput struct {
	PID          int32
	FilePath     string
	FileSHA256   string
	SigningID    string
	TeamID       string
	CDHash       string
	SigningChain []CertificateInput
}

// FileAccessEventInput is a Santa file-access event ready for persistence.
type FileAccessEventInput struct {
	RuleVersion  string
	RuleName     string
	Target       string
	Decision     FileAccessDecision
	OccurredAt   time.Time
	ProcessChain []ProcessInput
}

type Bundle struct {
	ID                   int64      `json:"id"`
	SHA256               string     `json:"sha256"`
	BundleID             string     `json:"bundle_id"`
	Name                 string     `json:"name"`
	Path                 string     `json:"path"`
	ExecutableRelPath    string     `json:"executable_rel_path"`
	Version              string     `json:"version"`
	VersionString        string     `json:"version_string"`
	BinaryCount          int32      `json:"binary_count"`
	CollectedBinaryCount int32      `json:"collected_binary_count"`
	HashMillis           int32      `json:"hash_millis"`
	UploadedAt           *time.Time `json:"uploaded_at,omitempty"`
}

// Process is a process-chain entry exposed through the admin API.
type Process struct {
	PID          int32               `json:"pid"`
	FilePath     string              `json:"file_path"`
	FileName     string              `json:"file_name"`
	FileSHA256   string              `json:"file_sha256"`
	SigningID    string              `json:"signing_id"`
	TeamID       string              `json:"team_id"`
	CDHash       string              `json:"cdhash"`
	SigningChain []SigningChainEntry `json:"signing_chain,omitempty"`
}

// FileAccessEvent is an observed Santa file-access policy decision.
type FileAccessEvent struct {
	ID             int64              `json:"id"`
	HostID         int64              `json:"host_id"`
	Host           HostSummary        `json:"host"`
	RuleVersion    string             `json:"rule_version"`
	RuleName       string             `json:"rule_name"`
	Target         string             `json:"target"`
	Decision       FileAccessDecision `json:"decision"`
	PrimaryProcess Process            `json:"primary_process"`
	ProcessChain   []Process          `json:"process_chain,omitempty"`
	OccurredAt     time.Time          `json:"occurred_at"`
	IngestedAt     time.Time          `json:"ingested_at"`
}
