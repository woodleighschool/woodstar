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
}

type BundleReference struct {
	SHA256               string     `json:"sha256"                 db:"sha256"`
	BundleID             string     `json:"bundle_id"              db:"bundle_id"`
	Name                 string     `json:"name"                   db:"name"`
	Path                 string     `json:"path"                   db:"path"`
	Version              string     `json:"version"                db:"version"`
	VersionString        string     `json:"version_string"         db:"version_string"`
	BinaryCount          int32      `json:"binary_count"           db:"binary_count"`
	CollectedBinaryCount int32      `json:"collected_binary_count" db:"collected_binary_count"`
	HashMillis           int32      `json:"hash_millis"            db:"hash_millis"`
	UploadedAt           *time.Time `json:"uploaded_at,omitempty"  db:"uploaded_at"`
	Complete             bool       `json:"complete"               db:"complete"`
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
	ExecutableCount int32               `json:"executable_count"`
	RuleCount       int32               `json:"rule_count"`
}

type CertificateReference struct {
	SHA256             string     `json:"sha256"                db:"sha256"`
	CommonName         string     `json:"common_name"           db:"common_name"`
	Organization       string     `json:"organization"          db:"organization"`
	OrganizationalUnit string     `json:"organizational_unit"   db:"organizational_unit"`
	ValidFrom          *time.Time `json:"valid_from,omitempty"  db:"valid_from"`
	ValidUntil         *time.Time `json:"valid_until,omitempty" db:"valid_until"`
	RuleCount          int32      `json:"rule_count"            db:"rule_count"`
}
