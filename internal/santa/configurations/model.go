package configurations

import (
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

type ClientMode string

const (
	ClientModeUnknown    ClientMode = "unknown"
	ClientModeMonitor    ClientMode = "monitor"
	ClientModeLockdown   ClientMode = "lockdown"
	ClientModeStandalone ClientMode = "standalone"
)

type RemovableMediaAction string

const (
	RemovableMediaActionAllow   RemovableMediaAction = "allow"
	RemovableMediaActionBlock   RemovableMediaAction = "block"
	RemovableMediaActionRemount RemovableMediaAction = "remount"
)

type ConfigurationListParams struct {
	dbutil.ListParams
}

// RemovableMediaPolicy is the optional USB policy. The zero value (Action == "")
// means "no policy"; the wire shape omits zero values via json:"omitzero".
type RemovableMediaPolicy struct {
	Action       RemovableMediaAction `json:"action,omitempty"        enum:"allow,block,remount"`
	RemountFlags []string             `json:"remount_flags,omitempty"                            doc:"Mount flags required when action is remount."`
}

// IsZero reports whether the policy is the no-policy zero value.
func (p RemovableMediaPolicy) IsZero() bool {
	return p.Action == "" && len(p.RemountFlags) == 0
}

// ConfigurationMutation is the complete editable Santa configuration policy.
// The admin SPA defaults every knob to Santa's own default and sends an
// explicit value; the backend validates but does not substitute defaults.
type ConfigurationMutation struct {
	Name                          string               `json:"name"`
	ClientMode                    ClientMode           `json:"client_mode"                               enum:"monitor,lockdown,standalone"`
	EnableBundles                 bool                 `json:"enable_bundles"`
	EnableTransitiveRules         bool                 `json:"enable_transitive_rules"`
	EnableAllEventUpload          bool                 `json:"enable_all_event_upload"`
	FullSyncIntervalSeconds       int                  `json:"full_sync_interval_seconds"                                                   minimum:"60"`
	BatchSize                     int                  `json:"batch_size"                                                                   minimum:"5"  maximum:"100"`
	AllowedPathRegex              string               `json:"allowed_path_regex,omitempty"`
	BlockedPathRegex              string               `json:"blocked_path_regex,omitempty"`
	RemovableMediaPolicy          RemovableMediaPolicy `json:"removable_media_policy,omitzero"`
	EncryptedRemovableMediaPolicy RemovableMediaPolicy `json:"encrypted_removable_media_policy,omitzero"`
	EventDetailURL                string               `json:"event_detail_url,omitempty"`
	EventDetailText               string               `json:"event_detail_text,omitempty"`
	LabelIDs                      []int64              `json:"label_ids,omitempty"`
}

type Configuration struct {
	ID                            int64                `json:"id"`
	Name                          string               `json:"name"`
	Position                      int                  `json:"position"`
	ClientMode                    ClientMode           `json:"client_mode"                               enum:"monitor,lockdown,standalone"`
	EnableBundles                 bool                 `json:"enable_bundles"`
	EnableTransitiveRules         bool                 `json:"enable_transitive_rules"`
	EnableAllEventUpload          bool                 `json:"enable_all_event_upload"`
	FullSyncIntervalSeconds       int                  `json:"full_sync_interval_seconds"`
	BatchSize                     int                  `json:"batch_size"`
	AllowedPathRegex              string               `json:"allowed_path_regex,omitempty"`
	BlockedPathRegex              string               `json:"blocked_path_regex,omitempty"`
	RemovableMediaPolicy          RemovableMediaPolicy `json:"removable_media_policy,omitzero"`
	EncryptedRemovableMediaPolicy RemovableMediaPolicy `json:"encrypted_removable_media_policy,omitzero"`
	EventDetailURL                string               `json:"event_detail_url,omitempty"`
	EventDetailText               string               `json:"event_detail_text,omitempty"`
	LabelIDs                      []int64              `json:"label_ids"`
	CreatedAt                     time.Time            `json:"created_at"`
	UpdatedAt                     time.Time            `json:"updated_at"`
}

type ResolvedConfiguration struct {
	Configuration
	MatchedViaLabel *LabelMatch `json:"matched_via_label,omitempty"`
}

type LabelMatch struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type ConfigurationLabelConflictError struct {
	LabelID           int64  `json:"label_id"`
	ConfigurationID   int64  `json:"configuration_id"`
	ConfigurationName string `json:"configuration_name"`
}

func (e *ConfigurationLabelConflictError) Error() string {
	return "configuration label already belongs to another configuration"
}

// GetStatus lets Huma map the error to HTTP 409 without a parallel handler-layer type.
func (e *ConfigurationLabelConflictError) GetStatus() int {
	return 409
}
