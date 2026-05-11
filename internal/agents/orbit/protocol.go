package orbit

import "encoding/json"

// EnrollRequest is the JSON body Orbit POSTs to /api/fleet/orbit/enroll.
type EnrollRequest struct {
	EnrollSecret      string `json:"enroll_secret"`
	HardwareUUID      string `json:"hardware_uuid"`
	HardwareSerial    string `json:"hardware_serial,omitempty"`
	Hostname          string `json:"hostname,omitempty"`
	Platform          string `json:"platform,omitempty"`
	PlatformLike      string `json:"platform_like,omitempty"`
	OsqueryIdentifier string `json:"osquery_identifier,omitempty"`
	ComputerName      string `json:"computer_name,omitempty"`
	HardwareModel     string `json:"hardware_model,omitempty"`
}

// EnrollResponse is the JSON body returned to a successful enrollment.
// orbit_node_key is the credential Orbit uses on subsequent calls.
type EnrollResponse struct {
	OrbitNodeKey string `json:"orbit_node_key"`
}

// ConfigRequest carries Orbit's node key.
type ConfigRequest struct {
	OrbitNodeKey string `json:"orbit_node_key"`
}

// ConfigResponse is the Orbit config response.
type ConfigResponse struct {
	ScriptExecutionTimeout int                   `json:"script_execution_timeout,omitempty"`
	Flags                  json.RawMessage       `json:"command_line_startup_flags,omitempty"`
	Extensions             json.RawMessage       `json:"extensions,omitempty"`
	Notifications          ConfigNotifications   `json:"notifications,omitzero"`
	UpdateChannels         *ConfigUpdateChannels `json:"update_channels,omitempty"`
	NudgeConfig            *ConfigNudgeConfig    `json:"nudge_config,omitempty"`
}

// ConfigNotifications carries one-shot flags Orbit acts on. Mirrors Fleet's
// orbit notifications shape; populated when a feature is implemented.
type ConfigNotifications struct {
	RenewEnrollmentProfile      bool `json:"renew_enrollment_profile,omitempty"`
	RotateDiskEncryptionKey     bool `json:"rotate_disk_encryption_key,omitempty"`
	NeedsMDMMigration           bool `json:"needs_mdm_migration,omitempty"`
	NeedsProgrammaticWindowsMDM bool `json:"needs_programmatic_windows_mdm_enrollment,omitempty"`
}

// ConfigUpdateChannels names the TUF channels Orbit should track per-component.
type ConfigUpdateChannels struct {
	Orbit    string `json:"orbit,omitempty"`
	Osqueryd string `json:"osqueryd,omitempty"`
	Desktop  string `json:"desktop,omitempty"`
}

// ConfigNudgeConfig is the macOS Nudge configuration payload. Empty until Nudge
// integration ships; sent only when non-nil.
type ConfigNudgeConfig struct {
	UserExperience        json.RawMessage `json:"userExperience,omitempty"`
	OSVersionRequirements json.RawMessage `json:"osVersionRequirements,omitempty"`
}

// DeviceMappingRequest carries a profile-provided email.
type DeviceMappingRequest struct {
	OrbitNodeKey string `json:"orbit_node_key"`
	Email        string `json:"email"`
}
