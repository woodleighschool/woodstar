package rules

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/openapischema"
	"github.com/woodleighschool/woodstar/internal/validation"
)

var (
	sha256IdentifierRE    = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
	cdhashIdentifierRE    = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)
	signingIDIdentifierRE = regexp.MustCompile(`^(?:[A-Z0-9]{10}|platform):[a-zA-Z0-9.-]+$`)
	teamIDIdentifierRE    = regexp.MustCompile(`^[A-Z0-9]{10}$`)
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
	return openapischema.StringEnum(RuleTypeValues...)
}

func (Policy) Schema(_ huma.Registry) *huma.Schema {
	return openapischema.StringEnum(PolicyValues...)
}

type RuleListParams struct {
	dbutil.ListParams

	RuleType RuleType `validate:"omitempty,oneof=binary certificate teamid signingid cdhash bundle"`
}

func (params *RuleListParams) normalize() {
	params.ListParams = dbutil.NormalizeListParams(params.ListParams)
	params.RuleType = RuleType(strings.TrimSpace(string(params.RuleType)))
}

func (params *RuleListParams) validate() error {
	if err := validation.Struct(params); err != nil {
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	return nil
}

type RuleMutation struct {
	RuleType      RuleType    `json:"rule_type"                validate:"required,oneof=binary certificate teamid signingid cdhash bundle"`
	Identifier    string      `json:"identifier"               validate:"required,notblank"                                                minLength:"1"`
	Name          string      `json:"name"                     validate:"required,notblank"                                                minLength:"1"`
	Description   string      `json:"description,omitempty"`
	CustomMessage string      `json:"custom_message,omitempty"`
	CustomURL     string      `json:"custom_url,omitempty"     validate:"omitempty,https_url"                                                            format:"uri"`
	Targets       RuleTargets `json:"targets"`
}

func (p *RuleMutation) Validate() error {
	if err := validation.Struct(p); err != nil {
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	if err := validateRuleIdentifier(p.RuleType, p.Identifier); err != nil {
		return err
	}
	if err := p.Targets.validate(); err != nil {
		return err
	}
	return nil
}

func (p *RuleMutation) normalize() {
	p.RuleType = RuleType(strings.TrimSpace(string(p.RuleType)))
	p.Identifier = strings.TrimSpace(p.Identifier)
	p.Name = strings.TrimSpace(p.Name)
	p.Description = strings.TrimSpace(p.Description)
	p.CustomMessage = strings.TrimSpace(p.CustomMessage)
	p.CustomURL = strings.TrimSpace(p.CustomURL)
	p.Targets = normalizeRuleTargets(p.Targets)
}

func validateRuleIdentifier(ruleType RuleType, identifier string) error {
	switch ruleType {
	case RuleTypeBinary:
		if !sha256IdentifierRE.MatchString(identifier) {
			return fmt.Errorf("%w: identifier must be a 64 character SHA-256 hex hash", dbutil.ErrInvalidInput)
		}
	case RuleTypeCertificate:
		if !sha256IdentifierRE.MatchString(identifier) {
			return fmt.Errorf(
				"%w: identifier must be a 64 character certificate SHA-256 hex fingerprint",
				dbutil.ErrInvalidInput,
			)
		}
	case RuleTypeBundle:
		if !sha256IdentifierRE.MatchString(identifier) {
			return fmt.Errorf("%w: identifier must be a 64 character bundle SHA-256 hex hash", dbutil.ErrInvalidInput)
		}
	case RuleTypeCDHash:
		if !cdhashIdentifierRE.MatchString(identifier) {
			return fmt.Errorf("%w: identifier must be a 40 character CDHash hex value", dbutil.ErrInvalidInput)
		}
	case RuleTypeSigningID:
		if !signingIDIdentifierRE.MatchString(identifier) {
			return fmt.Errorf(
				"%w: identifier must be TEAMID:bundle.identifier or platform:bundle.identifier",
				dbutil.ErrInvalidInput,
			)
		}
	case RuleTypeTeamID:
		if !teamIDIdentifierRE.MatchString(identifier) {
			return fmt.Errorf("%w: identifier must be a 10 character uppercase Team ID", dbutil.ErrInvalidInput)
		}
	}
	return nil
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
