package configurations

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

const syncPolicyDigestVersion = 1

type syncPolicyDigestInput struct {
	Version       int                            `json:"version"`
	Configuration *syncPolicyDigestConfiguration `json:"configuration,omitempty"`
}

type syncPolicyDigestConfiguration struct {
	ID       int64        `json:"id"`
	Settings SyncSettings `json:"settings"`
}

// SyncPolicyDigest identifies the selected configuration and the settings Woodstar sends to Santa.
func SyncPolicyDigest(configuration *Configuration) (string, error) {
	input := syncPolicyDigestInput{Version: syncPolicyDigestVersion}
	if configuration != nil {
		input.Configuration = &syncPolicyDigestConfiguration{
			ID:       configuration.ID,
			Settings: configuration.SyncSettings,
		}
	}

	payload, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}
