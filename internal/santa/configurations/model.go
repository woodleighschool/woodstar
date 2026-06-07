package configurations

import (
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/humaschema"
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

const (
	RemovableMediaActionAllow   RemovableMediaAction = "allow"
	RemovableMediaActionBlock   RemovableMediaAction = "block"
	RemovableMediaActionRemount RemovableMediaAction = "remount"
)

var RemovableMediaActionValues = []RemovableMediaAction{
	RemovableMediaActionAllow,
	RemovableMediaActionBlock,
	RemovableMediaActionRemount,
}

type ConfigurationListParams struct {
	dbutil.ListParams
}

// RemovableMediaPolicy is the optional USB policy. The zero value (Action == "")
// means "no policy"; the wire shape omits zero values via json:"omitzero".
type RemovableMediaPolicy struct {
	Action       RemovableMediaAction `json:"action,omitempty"`
	RemountFlags []string             `json:"remount_flags,omitempty" doc:"Mount flags required when action is remount."`
}

func (ClientMode) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(ClientModeValues...)
}

func (ReportedClientMode) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(ReportedClientModeValues...)
}

func (RemovableMediaAction) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(RemovableMediaActionValues...)
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
	Description                   string               `json:"description,omitempty"`
	ClientMode                    ClientMode           `json:"client_mode"`
	EnableBundles                 bool                 `json:"enable_bundles"`
	EnableTransitiveRules         bool                 `json:"enable_transitive_rules"`
	EnableAllEventUpload          bool                 `json:"enable_all_event_upload"`
	FullSyncIntervalSeconds       int32                `json:"full_sync_interval_seconds"                minimum:"60"`
	BatchSize                     int32                `json:"batch_size"                                minimum:"5"  maximum:"100"`
	AllowedPathRegex              string               `json:"allowed_path_regex,omitempty"`
	BlockedPathRegex              string               `json:"blocked_path_regex,omitempty"`
	RemovableMediaPolicy          RemovableMediaPolicy `json:"removable_media_policy,omitzero"`
	EncryptedRemovableMediaPolicy RemovableMediaPolicy `json:"encrypted_removable_media_policy,omitzero"`
	EventDetailURL                string               `json:"event_detail_url,omitempty"`
	EventDetailText               string               `json:"event_detail_text,omitempty"`
	Targets                       ConfigurationTargets `json:"targets"`
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
