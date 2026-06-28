package rules

import (
	"fmt"
	"regexp"
	"slices"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api/schema"
	"github.com/woodleighschool/woodstar/internal/dbutil"
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
	return schema.StringEnum(RuleTypeValues...)
}

func (Policy) Schema(_ huma.Registry) *huma.Schema {
	return schema.StringEnum(PolicyValues...)
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

func (p RuleMutation) Validate() error {
	if !validRuleType(p.RuleType) {
		return fmt.Errorf("%w: rule_type is required", dbutil.ErrInvalidInput)
	}
	if p.Identifier == "" {
		return fmt.Errorf("%w: identifier is required", dbutil.ErrInvalidInput)
	}
	if p.Name == "" {
		return fmt.Errorf("%w: name is required", dbutil.ErrInvalidInput)
	}
	if err := validateRuleIdentifier(p.RuleType, p.Identifier); err != nil {
		return err
	}
	if err := p.Targets.validate(); err != nil {
		return err
	}
	return nil
}

func validRuleType(ruleType RuleType) bool {
	return slices.Contains(RuleTypeValues, ruleType)
}

func validPolicy(policy Policy) bool {
	return slices.Contains(PolicyValues, policy)
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
