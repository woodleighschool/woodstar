package references

import (
	"time"

	santarules "github.com/woodleighschool/woodstar/internal/santa/rules"
)

// SoftwareReference is the Santa security evidence related to one software title.
type SoftwareReference struct {
	ExecutionCount    int32                      `json:"execution_count"`
	BlockCount        int32                      `json:"block_count"`
	Bundles           []BundleReference          `json:"bundles"`
	Executables       []ExecutableReference      `json:"executables"`
	SigningIdentities []SigningIdentityReference `json:"signing_identities"`
	Certificates      []CertificateReference     `json:"certificates"`
	Rules             []RuleReference            `json:"rules"`
}

type BundleReference struct {
	SHA256               string     `json:"sha256"`
	BundleID             string     `json:"bundle_id"`
	Name                 string     `json:"name"`
	Path                 string     `json:"path"`
	Version              string     `json:"version"`
	VersionString        string     `json:"version_string"`
	BinaryCount          int32      `json:"binary_count"`
	CollectedBinaryCount int32      `json:"collected_binary_count"`
	HashMillis           int32      `json:"hash_millis"`
	UploadedAt           *time.Time `json:"uploaded_at,omitempty"`
	Complete             bool       `json:"complete"`
}

type ExecutableReference struct {
	SHA256         string `json:"sha256"`
	FileName       string `json:"file_name"`
	BundleID       string `json:"file_bundle_id,omitempty"`
	BundleName     string `json:"file_bundle_name,omitempty"`
	BundleVersion  string `json:"file_bundle_version,omitempty"`
	SigningID      string `json:"signing_id,omitempty"`
	TeamID         string `json:"team_id,omitempty"`
	CDHash         string `json:"cdhash,omitempty"`
	ExecutionCount int32  `json:"execution_count"`
	BlockCount     int32  `json:"block_count"`
}

type SigningIdentityReference struct {
	RuleType        santarules.RuleType `json:"rule_type"`
	Identifier      string              `json:"identifier"`
	Name            string              `json:"name"`
	ExecutableCount int32               `json:"executable_count"`
	RuleCount       int32               `json:"rule_count"`
}

type CertificateReference struct {
	SHA256             string     `json:"sha256"`
	CommonName         string     `json:"common_name"`
	Organization       string     `json:"organization"`
	OrganizationalUnit string     `json:"organizational_unit"`
	ValidFrom          *time.Time `json:"valid_from,omitempty"`
	ValidUntil         *time.Time `json:"valid_until,omitempty"`
	RuleCount          int32      `json:"rule_count"`
}

type RuleReference struct {
	ID            int64               `json:"id"`
	RuleType      santarules.RuleType `json:"rule_type"`
	Identifier    string              `json:"identifier"`
	Name          string              `json:"name"`
	CustomMessage string              `json:"custom_message"`
	CustomURL     string              `json:"custom_url"`
}
