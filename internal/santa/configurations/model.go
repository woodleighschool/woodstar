package configurations

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/listing"
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

type RemountFlag string

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

const (
	RemountFlagReadOnly  RemountFlag = "rdonly"
	RemountFlagNoExec    RemountFlag = "noexec"
	RemountFlagNoSUID    RemountFlag = "nosuid"
	RemountFlagNoBrowse  RemountFlag = "nobrowse"
	RemountFlagNoOwners  RemountFlag = "noowners"
	RemountFlagNoDev     RemountFlag = "nodev"
	RemountFlagNoJournal RemountFlag = "-j"
	RemountFlagAsync     RemountFlag = "async"
)

var RemountFlagValues = []RemountFlag{
	RemountFlagReadOnly,
	RemountFlagNoExec,
	RemountFlagNoSUID,
	RemountFlagNoBrowse,
	RemountFlagNoOwners,
	RemountFlagNoDev,
	RemountFlagNoJournal,
	RemountFlagAsync,
}

type ConfigurationListParams struct {
	ListParams listing.Params
}

type RemovableMediaPolicy struct {
	Action       RemovableMediaAction `json:"action"                  validate:"required,oneof=allow block remount"`
	RemountFlags []RemountFlag        `json:"remount_flags,omitempty" validate:"excluded_unless=Action remount,required_if=Action remount,unique,dive,oneof=rdonly noexec nosuid nobrowse noowners nodev -j async" doc:"Mount flags required when action is remount."`
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

func (RemountFlag) Schema(_ huma.Registry) *huma.Schema {
	return openapischema.StringEnum(RemountFlagValues...)
}

// SyncSettings is the complete device-facing policy emitted during preflight.
type SyncSettings struct {
	ClientMode                    ClientMode            `json:"client_mode"                               validate:"required,oneof=monitor lockdown standalone"`
	EnableBundles                 bool                  `json:"enable_bundles"`
	EnableTransitiveRules         bool                  `json:"enable_transitive_rules"`
	EnableAllEventUpload          bool                  `json:"enable_all_event_upload"`
	DisableUnknownEventUpload     bool                  `json:"disable_unknown_event_upload"`
	OverrideFileAccessAction      FileAccessAction      `json:"override_file_access_action"               validate:"required,oneof=none audit_only disable"`
	FullSyncIntervalSeconds       int32                 `json:"full_sync_interval_seconds"                validate:"gte=60"                                minimum:"60"`
	BatchSize                     int32                 `json:"batch_size"                                validate:"gte=5,lte=100"                         minimum:"5"  maximum:"100"`
	AllowedPathRegex              *string               `json:"allowed_path_regex,omitempty"`
	BlockedPathRegex              *string               `json:"blocked_path_regex,omitempty"`
	RemovableMediaPolicy          *RemovableMediaPolicy `json:"removable_media_policy,omitempty"`
	EncryptedRemovableMediaPolicy *RemovableMediaPolicy `json:"encrypted_removable_media_policy,omitempty"`
	EventDetailURL                *string               `json:"event_detail_url,omitempty"                validate:"omitempty,https_url"                 format:"uri"`
	EventDetailText               *string               `json:"event_detail_text,omitempty"`
}

// ConfigurationMutation is the complete editable Santa configuration policy.
type ConfigurationMutation struct {
	SyncSettings

	Name        string               `json:"name"                  validate:"required,notblank" minLength:"1"`
	Description string               `json:"description,omitempty"`
	Targets     ConfigurationTargets `json:"targets"`
}

// Validate enforces caller-facing rules before storage.
func (p *ConfigurationMutation) Validate() error {
	if err := validation.Struct(p); err != nil {
		return fmt.Errorf("%w: %w", fault.ErrInvalidInput, err)
	}
	if err := p.Targets.validate(); err != nil {
		return err
	}
	return nil
}

func (p *ConfigurationMutation) normalize() {
	p.Name = strings.TrimSpace(p.Name)
	p.Description = strings.TrimSpace(p.Description)
	p.AllowedPathRegex = normalizeOptionalString(p.AllowedPathRegex)
	p.BlockedPathRegex = normalizeOptionalString(p.BlockedPathRegex)
	p.EventDetailURL = normalizeOptionalString(p.EventDetailURL)
	p.EventDetailText = normalizeOptionalString(p.EventDetailText)
	normalizeRemovableMediaPolicy(p.RemovableMediaPolicy)
	normalizeRemovableMediaPolicy(p.EncryptedRemovableMediaPolicy)
	p.Targets = normalizeConfigurationTargets(p.Targets)
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		return nil
	}
	return &normalized
}

func normalizeRemovableMediaPolicy(policy *RemovableMediaPolicy) {
	if policy == nil {
		return
	}
	policy.Action = RemovableMediaAction(strings.TrimSpace(string(policy.Action)))
	for i := range policy.RemountFlags {
		policy.RemountFlags[i] = RemountFlag(strings.TrimSpace(string(policy.RemountFlags[i])))
	}
	order := make(map[RemountFlag]int, len(RemountFlagValues))
	for i, flag := range RemountFlagValues {
		order[flag] = i
	}
	slices.SortFunc(policy.RemountFlags, func(left, right RemountFlag) int {
		return cmp.Compare(order[left], order[right])
	})
}

type Configuration struct {
	SyncSettings

	ID          int64                `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Position    int32                `json:"position"`
	Targets     ConfigurationTargets `json:"targets"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}

type ConfigurationMatch struct {
	Configuration

	MatchedViaLabel *LabelMatch `json:"matched_via_label,omitempty"`
}

type LabelMatch struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
