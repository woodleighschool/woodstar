package references_test

import (
	"strings"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/santa/references"
	santarules "github.com/woodleighschool/woodstar/internal/santa/rules"
)

func TestSoftwareReferenceJoinsSoftwareInventoryToSantaEvidence(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := references.NewStore(db)
	hostStore := hosts.NewStore(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.DetailUpdate{
		HardwareUUID: "santa-reference-host",
		OrbitNodeKey: "santa-reference-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	var titleID int64
	if err := db.Pool().QueryRow(ctx, `
		INSERT INTO software_titles (name, display_name, source, bundle_identifier)
		VALUES ('Reference App', 'Reference App', 'apps', 'com.example.reference')
		RETURNING id
	`).Scan(&titleID); err != nil {
		t.Fatalf("insert software title: %v", err)
	}
	var softwareID int64
	if err := db.Pool().QueryRow(ctx, `
		INSERT INTO software (title_id, name, version, source, bundle_identifier)
		VALUES ($1, 'Reference App', '1.0', 'apps', 'com.example.reference')
		RETURNING id
	`, titleID).Scan(&softwareID); err != nil {
		t.Fatalf("insert software: %v", err)
	}

	executableSHA := strings.Repeat("1", 64)
	bundleHash := strings.Repeat("2", 64)
	certificateSHA := strings.Repeat("3", 64)
	if _, err := db.Pool().Exec(ctx, `
		INSERT INTO host_software_installed_paths (
			host_id,
			software_id,
			installed_path,
			team_identifier,
			cdhash_sha256,
			executable_sha256,
			executable_path
		)
		VALUES (
			$1,
			$2,
			'/Applications/Reference App.app',
			'TEAMREF',
			'ref-cdhash',
			$3,
			'/Applications/Reference App.app/Contents/MacOS/Reference App'
		)
	`, host.ID, softwareID, executableSHA); err != nil {
		t.Fatalf("insert software path: %v", err)
	}

	var executableID int64
	if err := db.Pool().QueryRow(ctx, `
		INSERT INTO santa_executables (
			sha256,
			file_name,
			file_bundle_id,
			file_bundle_path,
			file_bundle_name,
			file_bundle_version,
			signing_id,
			team_id,
			cdhash
		)
		VALUES (
			$1,
			'Reference App',
			'com.example.reference',
			'/Applications/Reference App.app',
			'Reference App',
			'1.0',
			'TEAMREF:com.example.reference',
			'TEAMREF',
			'ref-cdhash'
		)
		RETURNING id
	`, executableSHA).Scan(&executableID); err != nil {
		t.Fatalf("insert Santa executable: %v", err)
	}
	var bundleID int64
	if err := db.Pool().QueryRow(ctx, `
		INSERT INTO santa_bundles (
			sha256,
			bundle_id,
			name,
			path,
			version,
			binary_count,
			uploaded_at
		)
		VALUES ($1, 'com.example.reference', 'Reference App', '/Applications/Reference App.app', '1.0', 1, now())
		RETURNING id
	`, bundleHash).Scan(&bundleID); err != nil {
		t.Fatalf("insert Santa bundle: %v", err)
	}
	if _, err := db.Pool().Exec(ctx, `
		INSERT INTO santa_bundle_executables (bundle_id, executable_id)
		VALUES ($1, $2)
	`, bundleID, executableID); err != nil {
		t.Fatalf("link bundle executable: %v", err)
	}

	var certificateID int64
	if err := db.Pool().QueryRow(ctx, `
		INSERT INTO santa_certificates (sha256, common_name, organization, organizational_unit)
		VALUES ($1, 'Developer ID Application: Example', 'Example', 'TEAMREF')
		RETURNING id
	`, certificateSHA).Scan(&certificateID); err != nil {
		t.Fatalf("insert certificate: %v", err)
	}
	var chainID int64
	if err := db.Pool().QueryRow(ctx, `
		INSERT INTO santa_signing_chains (sha256)
		VALUES ($1)
		RETURNING id
	`, strings.Repeat("4", 64)).Scan(&chainID); err != nil {
		t.Fatalf("insert signing chain: %v", err)
	}
	if _, err := db.Pool().Exec(ctx, `
		INSERT INTO santa_signing_chain_entries (signing_chain_id, position, certificate_id)
		VALUES ($1, 0, $2);
		INSERT INTO santa_executable_signing_chains (executable_id, signing_chain_id)
		VALUES ($3, $1)
	`, chainID, certificateID, executableID); err != nil {
		t.Fatalf("link signing chain: %v", err)
	}
	if _, err := db.Pool().Exec(ctx, `
		INSERT INTO santa_execution_events (
			host_id,
			executable_id,
			file_path,
			executing_user,
			decision,
			occurred_at
		)
		VALUES ($1, $2, '/Applications/Reference App.app/Contents/MacOS/Reference App', 'alice', 'block_binary', $3)
	`, host.ID, executableID, time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("insert execution event: %v", err)
	}
	if _, err := db.Pool().Exec(ctx, `
		INSERT INTO santa_rules (rule_type, identifier, name)
		VALUES
			('teamid', 'TEAMREF', 'Team Rule'),
			('bundle', $1, 'Bundle Rule'),
			('certificate', $2, 'Certificate Rule')
	`, bundleHash, certificateSHA); err != nil {
		t.Fatalf("insert rules: %v", err)
	}

	ref, err := store.GetSoftwareReference(ctx, titleID)
	if err != nil {
		t.Fatalf("get software reference: %v", err)
	}
	if ref.ExecutionCount != 1 || ref.BlockCount != 1 {
		t.Fatalf("counts = %d/%d, want one blocked execution", ref.ExecutionCount, ref.BlockCount)
	}
	if len(ref.Bundles) != 1 || ref.Bundles[0].SHA256 != bundleHash || !ref.Bundles[0].Complete {
		t.Fatalf("bundles = %+v, want complete related bundle", ref.Bundles)
	}
	if len(ref.Executables) != 1 ||
		ref.Executables[0].SHA256 != executableSHA ||
		ref.Executables[0].BlockCount != 1 {
		t.Fatalf("executables = %+v, want matched executable with blocked count", ref.Executables)
	}
	if !hasSigningIdentity(ref.SigningIdentities, santarules.RuleTypeTeamID, "TEAMREF", 1) ||
		!hasSigningIdentity(ref.SigningIdentities, santarules.RuleTypeSigningID, "TEAMREF:com.example.reference", 0) ||
		!hasSigningIdentity(ref.SigningIdentities, santarules.RuleTypeCDHash, "ref-cdhash", 0) {
		t.Fatalf("signing identities = %+v", ref.SigningIdentities)
	}
	if len(ref.Certificates) != 1 ||
		ref.Certificates[0].SHA256 != certificateSHA ||
		ref.Certificates[0].RuleCount != 1 {
		t.Fatalf("certificates = %+v, want matching certificate and rule", ref.Certificates)
	}
	if !hasReferenceRule(ref.Rules, santarules.RuleTypeTeamID, "TEAMREF") ||
		!hasReferenceRule(ref.Rules, santarules.RuleTypeBundle, bundleHash) ||
		!hasReferenceRule(ref.Rules, santarules.RuleTypeCertificate, certificateSHA) {
		t.Fatalf("rules = %+v, want matching Santa rules", ref.Rules)
	}
}

func hasSigningIdentity(
	identities []references.SigningIdentityReference,
	targetType santarules.RuleType,
	identifier string,
	ruleCount int32,
) bool {
	for _, identity := range identities {
		if identity.TargetType == targetType && identity.Identifier == identifier && identity.RuleCount == ruleCount {
			return true
		}
	}
	return false
}

func hasReferenceRule(rules []references.RuleReference, ruleType santarules.RuleType, identifier string) bool {
	for _, rule := range rules {
		if rule.RuleType == ruleType && rule.Identifier == identifier {
			return true
		}
	}
	return false
}
