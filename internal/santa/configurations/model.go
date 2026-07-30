package configurations

import (
	"fmt"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/openapischema"
	"github.com/woodleighschool/woodstar/internal/validation"
)

type (
	ClientMode         string
	ReportedClientMode string
)

const ReportedClientModeUnknown ReportedClientMode = "unknown"

const (
	ClientModeMonitor    ClientMode = "monitor"
	ClientModeLockdown   ClientMode = "lockdown"
	ClientModeStandalone ClientMode = "standalone"
)

const (
	ReportedClientModeMonitor    ReportedClientMode = "monitor"
	ReportedClientModeLockdown   ReportedClientMode = "lockdown"
	ReportedClientModeStandalone ReportedClientMode = "standalone"
)

var (
	ClientModeValues = []ClientMode{
		ClientModeMonitor,
		ClientModeLockdown,
		ClientModeStandalone,
	}
	ReportedClientModeValues = []ReportedClientMode{
		ReportedClientModeUnknown,
		ReportedClientModeMonitor,
		ReportedClientModeLockdown,
		ReportedClientModeStandalone,
	}
)

type RemovableMediaAction string

type FileAccessAction string

const (
	RemovableMediaActionAllow   RemovableMediaAction = "allow"
	RemovableMediaActionBlock   RemovableMediaAction = "block"
	RemovableMediaActionRemount RemovableMediaAction = "remount"
)

const (
	FileAccessActionNone      FileAccessAction = "none"
	FileAccessActionAuditOnly FileAccessAction = "audit_only"
	FileAccessActionDisable   FileAccessAction = "disable"
)

var RemovableMediaActionValues = []RemovableMediaAction{
	RemovableMediaActionAllow,
	RemovableMediaActionBlock,
	RemovableMediaActionRemount,
}

var FileAccessActionValues = []FileAccessAction{
	FileAccessActionNone,
	FileAccessActionAuditOnly,
	FileAccessActionDisable,
}

type ConfigurationListParams struct {
	dbutil.ListParams
}

// RemovableMediaPolicy is the optional USB policy. The zero value (Action == "")
// means "no policy"; the wire shape omits zero values via json:"omitzero".
type RemovableMediaPolicy struct {
	Action       RemovableMediaAction `json:"action,omitempty"        validate:"omitempty,oneof=allow block remount"`
	RemountFlags []string             `json:"remount_flags,omitempty" validate:"excluded_unless=Action remount,required_if=Action remount,dive,required" doc:"Mount flags required when action is remount."`
}

func (ClientMode) Schema(_ huma.Registry) *huma.Schema {
	return openapischema.StringEnum(ClientModeValues...)
}

func (ReportedClientMode) Schema(_ huma.Registry) *huma.Schema {
	return openapischema.StringEnum(ReportedClientModeValues...)
}

func (RemovableMediaAction) Schema(_ huma.Registry) *huma.Schema {
	return openapischema.StringEnum(RemovableMediaActionValues...)
}

func (FileAccessAction) Schema(_ huma.Registry) *huma.Schema {
	return openapischema.StringEnum(FileAccessActionValues...)
}

// IsZero reports whether the policy is the no-policy zero value.
func (p RemovableMediaPolicy) IsZero() bool {
	return p.Action == "" && len(p.RemountFlags) == 0
}

// ConfigurationMutation is the complete editable Santa configuration policy.
// The admin SPA defaults every knob to Santa's own default and sends an
// explicit value; the backend validates but does not substitute defaults.
type ConfigurationMutation struct {
	Name                          string               `json:"name"                                      validate:"required,notblank"                          minLength:"1"`
	Description                   string               `json:"description,omitempty"`
	ClientMode                    ClientMode           `json:"client_mode"                               validate:"required,oneof=monitor lockdown standalone"`
	EnableBundles                 bool                 `json:"enable_bundles"`
	EnableTransitiveRules         bool                 `json:"enable_transitive_rules"`
	EnableAllEventUpload          bool                 `json:"enable_all_event_upload"`
	DisableUnknownEventUpload     bool                 `json:"disable_unknown_event_upload"`
	OverrideFileAccessAction      FileAccessAction     `json:"override_file_access_action"               validate:"required,oneof=none audit_only disable"`
	FullSyncIntervalSeconds       int32                `json:"full_sync_interval_seconds"                validate:"gte=60"                                                   minimum:"60"`
	BatchSize                     int32                `json:"batch_size"                                validate:"gte=5,lte=100"                                            minimum:"5"  maximum:"100"`
	AllowedPathRegex              string               `json:"allowed_path_regex,omitempty"`
	BlockedPathRegex              string               `json:"blocked_path_regex,omitempty"`
	RemovableMediaPolicy          RemovableMediaPolicy `json:"removable_media_policy,omitzero"`
	EncryptedRemovableMediaPolicy RemovableMediaPolicy `json:"encrypted_removable_media_policy,omitzero"`
	EventDetailURL                string               `json:"event_detail_url,omitempty"                validate:"omitempty,https_url"                                                                 format:"uri"`
	EventDetailText               string               `json:"event_detail_text,omitempty"`
	Targets                       ConfigurationTargets `json:"targets"`
}

// Validate enforces caller-facing rules before storage.
func (p *ConfigurationMutation) Validate() error {
	if err := validation.Struct(p); err != nil {
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	if err := p.Targets.validate(); err != nil {
		return err
	}
	return nil
}

func (p *ConfigurationMutation) normalize() {
	p.Name = strings.TrimSpace(p.Name)
	p.Description = strings.TrimSpace(p.Description)
	p.AllowedPathRegex = strings.TrimSpace(p.AllowedPathRegex)
	p.BlockedPathRegex = strings.TrimSpace(p.BlockedPathRegex)
	p.EventDetailURL = strings.TrimSpace(p.EventDetailURL)
	p.EventDetailText = strings.TrimSpace(p.EventDetailText)
	normalizeRemovableMediaPolicy(&p.RemovableMediaPolicy)
	normalizeRemovableMediaPolicy(&p.EncryptedRemovableMediaPolicy)
	p.Targets = normalizeConfigurationTargets(p.Targets)
}

func normalizeRemovableMediaPolicy(policy *RemovableMediaPolicy) {
	policy.Action = RemovableMediaAction(strings.TrimSpace(string(policy.Action)))
	for i := range policy.RemountFlags {
		policy.RemountFlags[i] = strings.TrimSpace(policy.RemountFlags[i])
	}
}

type Configuration struct {
	ID                            int64                `json:"id"`
	Name                          string               `json:"name"`
	Description                   string               `json:"description"`
	Position                      int32                `json:"position"`
	ClientMode                    ClientMode           `json:"client_mode"`
	EnableBundles                 bool                 `json:"enable_bundles"`
	EnableTransitiveRules         bool                 `json:"enable_transitive_rules"`
	EnableAllEventUpload          bool                 `json:"enable_all_event_upload"`
	DisableUnknownEventUpload     bool                 `json:"disable_unknown_event_upload"`
	OverrideFileAccessAction      FileAccessAction     `json:"override_file_access_action"`
	FullSyncIntervalSeconds       int32                `json:"full_sync_interval_seconds"`
	BatchSize                     int32                `json:"batch_size"`
	AllowedPathRegex              string               `json:"allowed_path_regex,omitempty"`
	BlockedPathRegex              string               `json:"blocked_path_regex,omitempty"`
	RemovableMediaPolicy          RemovableMediaPolicy `json:"removable_media_policy,omitzero"`
	EncryptedRemovableMediaPolicy RemovableMediaPolicy `json:"encrypted_removable_media_policy,omitzero"`
	EventDetailURL                string               `json:"event_detail_url,omitempty"`
	EventDetailText               string               `json:"event_detail_text,omitempty"`
	Targets                       ConfigurationTargets `json:"targets"`
	CreatedAt                     time.Time            `json:"created_at"`
	UpdatedAt                     time.Time            `json:"updated_at"`
}

type ConfigurationMatch struct {
	Configuration

	MatchedViaLabel *LabelMatch `json:"matched_via_label,omitempty"`
}

type LabelMatch struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
