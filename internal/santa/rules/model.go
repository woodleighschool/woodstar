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
	PolicyAllowlist         Policy = "allowlist"
	PolicyAllowlistCompiler Policy = "allowlist_compiler"
	PolicyBlocklist         Policy = "blocklist"
	PolicySilentBlocklist   Policy = "silent_blocklist"
	PolicyCEL               Policy = "cel"
)

var PolicyValues = []Policy{
	PolicyAllowlist,
	PolicyAllowlistCompiler,
	PolicyBlocklist,
	PolicySilentBlocklist,
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
	RuleType        RuleType           `json:"rule_type"`
	Identifier      string             `json:"identifier"`
	Name            string             `json:"name,omitempty"`
	Description     string             `json:"description,omitempty"`
	CustomMessage   string             `json:"custom_message,omitempty"`
	CustomURL       string             `json:"custom_url,omitempty"`
	Includes        []RuleIncludeWrite `json:"includes,omitempty"`
	ExcludeLabelIDs []int64            `json:"exclude_label_ids,omitempty"`
}

type RuleIncludeWrite struct {
	Policy        Policy `json:"policy"`
	CELExpression string `json:"cel_expression,omitempty"`
	LabelID       int64  `json:"label_id"`
}

type Rule struct {
	ID              int64         `json:"id"`
	RuleType        RuleType      `json:"rule_type"`
	Identifier      string        `json:"identifier"`
	Name            string        `json:"name"`
	Description     string        `json:"description"`
	CustomMessage   string        `json:"custom_message"`
	CustomURL       string        `json:"custom_url"`
	Includes        []RuleInclude `json:"includes"`
	ExcludeLabelIDs []int64       `json:"exclude_label_ids"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

type RuleInclude struct {
	ID            int64  `json:"id"`
	Position      int32  `json:"position"`
	Policy        Policy `json:"policy"`
	CELExpression string `json:"cel_expression,omitempty"`
	LabelID       int64  `json:"label_id"`
}

type HostRule struct {
	RuleID           int64    `json:"rule_id"`
	RuleType         RuleType `json:"rule_type"`
	Identifier       string   `json:"identifier"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Policy           Policy   `json:"policy"`
	CELExpression    string   `json:"cel_expression,omitempty"`
	CustomMessage    string   `json:"custom_message,omitempty"`
	CustomURL        string   `json:"custom_url,omitempty"`
	AppName          string   `json:"notification_app_name,omitempty"`
	MatchedIncludeID int64    `json:"matched_include_id"`
}

type RuleStatus struct {
	HostRule
	Applied     bool   `json:"applied"`
	PayloadHash string `json:"payload_hash"`
}

type RuleStatusListParams struct {
	dbutil.ListParams
}

type RuleTargetListParams struct {
	Q          string
	TargetType RuleType
	Limit      int
}

type RuleTarget struct {
	TargetType           RuleType `json:"target_type"`
	Identifier           string   `json:"identifier"`
	Name                 string   `json:"name"`
	Detail               string   `json:"detail,omitempty"`
	BundleID             string   `json:"bundle_id,omitempty"`
	Version              string   `json:"version,omitempty"`
	BinaryCount          int32    `json:"binary_count,omitempty"`
	CollectedBinaryCount int32    `json:"collected_binary_count,omitempty"`
	RuleCount            int32    `json:"rule_count"`
	Complete             bool     `json:"complete"`
}
