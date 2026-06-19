package rules

import (
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/humaschema"
)

type RuleType string

const (
	RuleTypeBinary      RuleType = "binary"
	RuleTypeCertificate RuleType = "certificate"
	RuleTypeTeamID      RuleType = "teamid"
	RuleTypeSigningID   RuleType = "signingid"
	RuleTypeCDHash      RuleType = "cdhash"
	RuleTypeBundle      RuleType = "bundle"
)

var RuleTypeValues = []RuleType{
	RuleTypeBinary,
	RuleTypeCertificate,
	RuleTypeTeamID,
	RuleTypeSigningID,
	RuleTypeCDHash,
	RuleTypeBundle,
}

type Policy string

const (
	PolicyAllowlist          Policy = "allowlist"
	PolicyAllowlistCompiler  Policy = "allowlist_compiler"
	PolicyBlocklist          Policy = "blocklist"
	PolicySilentBlocklist    Policy = "silent_blocklist"
	PolicySilentGUIBlocklist Policy = "silent_gui_blocklist"
	PolicySilentTTYBlocklist Policy = "silent_tty_blocklist"
	PolicyCEL                Policy = "cel"
)

var PolicyValues = []Policy{
	PolicyAllowlist,
	PolicyAllowlistCompiler,
	PolicyBlocklist,
	PolicySilentBlocklist,
	PolicySilentGUIBlocklist,
	PolicySilentTTYBlocklist,
	PolicyCEL,
}

func (RuleType) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(RuleTypeValues...)
}

func (Policy) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(PolicyValues...)
}

type RuleListParams struct {
	dbutil.ListParams

	RuleType RuleType
}

type RuleMutation struct {
	RuleType      RuleType    `json:"rule_type"`
	Identifier    string      `json:"identifier"`
	Name          string      `json:"name"`
	Description   string      `json:"description,omitempty"`
	CustomMessage string      `json:"custom_message,omitempty"`
	CustomURL     string      `json:"custom_url,omitempty"`
	Targets       RuleTargets `json:"targets"`
}

type Rule struct {
	ID            int64       `json:"id"`
	RuleType      RuleType    `json:"rule_type"`
	Identifier    string      `json:"identifier"`
	Name          string      `json:"name"`
	Description   string      `json:"description"`
	CustomMessage string      `json:"custom_message"`
	CustomURL     string      `json:"custom_url"`
	Targets       RuleTargets `json:"targets"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

type HostRule struct {
	RuleID        int64    `json:"rule_id"`
	RuleType      RuleType `json:"rule_type"`
	Identifier    string   `json:"identifier"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Policy        Policy   `json:"policy"`
	CELExpression string   `json:"cel_expression,omitempty"`
	CustomMessage string   `json:"custom_message,omitempty"`
	CustomURL     string   `json:"custom_url,omitempty"`
	AppName       string   `json:"notification_app_name,omitempty"`
}

type RuleStatus struct {
	HostRule

	Applied bool `json:"applied"`
}

type RuleStatusListParams struct {
	dbutil.ListParams
}

type RuleReferenceListParams struct {
	Q        string
	RuleType RuleType
	Limit    int32
}

type RuleReferenceCandidate struct {
	RuleType                      RuleType `json:"rule_type"`
	Identifier                    string   `json:"identifier"`
	DisplayName                   string   `json:"display_name,omitempty"`
	CertificateCommonName         string   `json:"certificate_common_name,omitempty"`
	CertificateOrganization       string   `json:"certificate_organization,omitempty"`
	CertificateOrganizationalUnit string   `json:"certificate_organizational_unit,omitempty"`
	FileName                      string   `json:"file_name,omitempty"`
	BundleIdentifier              string   `json:"bundle_identifier,omitempty"`
	Path                          string   `json:"path,omitempty"`
	Version                       string   `json:"version,omitempty"`
	BinaryCount                   int32    `json:"binary_count,omitempty"`
	CollectedBinaryCount          int32    `json:"collected_binary_count,omitempty"`
	RuleCount                     int32    `json:"rule_count"`
	Complete                      bool     `json:"complete"`
}
