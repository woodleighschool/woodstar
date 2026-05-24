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

type RemovableMediaPolicy struct {
	Action       RemovableMediaAction `json:"action"                  enum:"allow,block,remount"`
	RemountFlags []string             `json:"remount_flags,omitempty"                            doc:"Mount flags required when action is remount."`
}

// ConfigurationMutation is the complete editable Santa configuration policy.
// Optional nil fields clear that setting; updates replace the full editable shape rather than patching individual fields.
type ConfigurationMutation struct {
	Name                          string                `json:"name"`
	ClientMode                    ClientMode            `json:"client_mode,omitempty"                      enum:"monitor,lockdown,standalone"`
	EnableBundles                 *bool                 `json:"enable_bundles,omitempty"`
	EnableTransitiveRules         *bool                 `json:"enable_transitive_rules,omitempty"`
	EnableAllEventUpload          *bool                 `json:"enable_all_event_upload,omitempty"`
	FullSyncIntervalSeconds       *int                  `json:"full_sync_interval_seconds,omitempty"`
	BatchSize                     *int                  `json:"batch_size,omitempty"`
	AllowedPathRegex              *string               `json:"allowed_path_regex,omitempty"`
	BlockedPathRegex              *string               `json:"blocked_path_regex,omitempty"`
	RemovableMediaPolicy          *RemovableMediaPolicy `json:"removable_media_policy,omitempty"`
	EncryptedRemovableMediaPolicy *RemovableMediaPolicy `json:"encrypted_removable_media_policy,omitempty"`
	EventDetailURL                *string               `json:"event_detail_url,omitempty"`
	EventDetailText               *string               `json:"event_detail_text,omitempty"`
	LabelIDs                      []int64               `json:"label_ids,omitempty"`
}

type Configuration struct {
	ID                            int64                 `json:"id"`
	Name                          string                `json:"name"`
	Position                      int                   `json:"position"`
	ClientMode                    ClientMode            `json:"client_mode"                                enum:"monitor,lockdown,standalone"`
	EnableBundles                 *bool                 `json:"enable_bundles,omitempty"`
	EnableTransitiveRules         *bool                 `json:"enable_transitive_rules,omitempty"`
	EnableAllEventUpload          *bool                 `json:"enable_all_event_upload,omitempty"`
	FullSyncIntervalSeconds       *int                  `json:"full_sync_interval_seconds,omitempty"`
	BatchSize                     *int                  `json:"batch_size,omitempty"`
	AllowedPathRegex              *string               `json:"allowed_path_regex,omitempty"`
	BlockedPathRegex              *string               `json:"blocked_path_regex,omitempty"`
	RemovableMediaPolicy          *RemovableMediaPolicy `json:"removable_media_policy,omitempty"`
	EncryptedRemovableMediaPolicy *RemovableMediaPolicy `json:"encrypted_removable_media_policy,omitempty"`
	EventDetailURL                *string               `json:"event_detail_url,omitempty"`
	EventDetailText               *string               `json:"event_detail_text,omitempty"`
	LabelIDs                      []int64               `json:"label_ids"`
	CreatedAt                     time.Time             `json:"created_at"`
	UpdatedAt                     time.Time             `json:"updated_at"`
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
