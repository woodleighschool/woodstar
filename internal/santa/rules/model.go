package rules

import (
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

type RuleType string

const (
	RuleTypeBinary      RuleType = "binary"
	RuleTypeCertificate RuleType = "certificate"
	RuleTypeTeamID      RuleType = "teamid"
	RuleTypeSigningID   RuleType = "signingid"
	RuleTypeCDHash      RuleType = "cdhash"
)

type Policy string

const (
	PolicyAllowlist         Policy = "allowlist"
	PolicyAllowlistCompiler Policy = "allowlist_compiler"
	PolicyBlocklist         Policy = "blocklist"
	PolicySilentBlocklist   Policy = "silent_blocklist"
	PolicyCEL               Policy = "cel"
)

type RuleListParams struct {
	dbutil.ListParams

	RuleType RuleType
}

type RuleMutation struct {
	RuleType        RuleType           `json:"rule_type"                   enum:"binary,certificate,teamid,signingid,cdhash"`
	Identifier      string             `json:"identifier"`
	Name            string             `json:"name,omitempty"`
	CustomMessage   string             `json:"custom_message,omitempty"`
	CustomURL       string             `json:"custom_url,omitempty"`
	Includes        []RuleIncludeWrite `json:"includes,omitempty"`
	ExcludeLabelIDs []int64            `json:"exclude_label_ids,omitempty"`
}

type RuleIncludeWrite struct {
	Policy        Policy  `json:"policy"                   enum:"allowlist,allowlist_compiler,blocklist,silent_blocklist,cel"`
	CELExpression string  `json:"cel_expression,omitempty"`
	LabelIDs      []int64 `json:"label_ids"`
}

type Rule struct {
	ID              int64         `json:"id"`
	RuleType        RuleType      `json:"rule_type"         enum:"binary,certificate,teamid,signingid,cdhash"`
	Identifier      string        `json:"identifier"`
	Name            string        `json:"name"`
	CustomMessage   string        `json:"custom_message"`
	CustomURL       string        `json:"custom_url"`
	Includes        []RuleInclude `json:"includes"`
	ExcludeLabelIDs []int64       `json:"exclude_label_ids"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

type RuleInclude struct {
	ID            int64   `json:"id"`
	Position      int     `json:"position"`
	Policy        Policy  `json:"policy"                   enum:"allowlist,allowlist_compiler,blocklist,silent_blocklist,cel"`
	CELExpression string  `json:"cel_expression,omitempty"`
	LabelIDs      []int64 `json:"label_ids"`
}

type EffectiveRule struct {
	RuleID           int64    `json:"rule_id"`
	RuleType         RuleType `json:"rule_type"                enum:"binary,certificate,teamid,signingid,cdhash"`
	Identifier       string   `json:"identifier"`
	Policy           Policy   `json:"policy"                   enum:"allowlist,allowlist_compiler,blocklist,silent_blocklist,cel"`
	CELExpression    string   `json:"cel_expression,omitempty"`
	CustomMessage    string   `json:"custom_message,omitempty"`
	CustomURL        string   `json:"custom_url,omitempty"`
	MatchedIncludeID int64    `json:"matched_include_id"`
}

type EffectiveRuleStatus struct {
	EffectiveRule
	Applied     bool   `json:"applied"`
	PayloadHash string `json:"payload_hash"`
}

type EffectiveRuleListParams struct {
	dbutil.ListParams
}
