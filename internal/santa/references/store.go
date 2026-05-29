package references

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// Store loads Santa read models for inventory reference views.
type Store struct {
	db *database.DB
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db}
}

func (s *Store) GetSoftwareReference(ctx context.Context, softwareTitleID int64) (*SoftwareReference, error) {
	facts, err := s.loadSoftwareFacts(ctx, softwareTitleID)
	if err != nil {
		return nil, err
	}
	ref := SoftwareReference{}
	if ref.ExecutionCount, ref.BlockCount, err = s.executionCounts(ctx, facts); err != nil {
		return nil, err
	}
	if ref.Bundles, err = s.bundles(ctx, facts); err != nil {
		return nil, err
	}
	if ref.Executables, err = s.executables(ctx, facts); err != nil {
		return nil, err
	}
	if ref.SigningIdentities, err = s.signingIdentities(ctx, facts); err != nil {
		return nil, err
	}
	if ref.Certificates, err = s.certificates(ctx, facts); err != nil {
		return nil, err
	}
	if ref.Rules, err = s.rules(ctx, facts); err != nil {
		return nil, err
	}
	return &ref, nil
}

type softwareFacts struct {
	bundleIDs         []string
	paths             []string
	executableSHA256s []string
	cdhashes          []string
	teamIDs           []string
	signingIDs        []string
}

func (facts softwareFacts) args() []any {
	return []any{
		facts.executableSHA256s,
		facts.cdhashes,
		facts.teamIDs,
		facts.signingIDs,
		facts.bundleIDs,
		facts.paths,
	}
}

type softwareFactAccumulator struct {
	bundleIDs         map[string]struct{}
	paths             map[string]struct{}
	executableSHA256s map[string]struct{}
	cdhashes          map[string]struct{}
	teamIDs           map[string]struct{}
	signingIDs        map[string]struct{}
}

func newSoftwareFactAccumulator() softwareFactAccumulator {
	return softwareFactAccumulator{
		bundleIDs:         map[string]struct{}{},
		paths:             map[string]struct{}{},
		executableSHA256s: map[string]struct{}{},
		cdhashes:          map[string]struct{}{},
		teamIDs:           map[string]struct{}{},
		signingIDs:        map[string]struct{}{},
	}
}

func (s *Store) loadSoftwareFacts(ctx context.Context, softwareTitleID int64) (softwareFacts, error) {
	rows, err := s.db.Pool().Query(ctx, `
		SELECT
			COALESCE(s.bundle_identifier, ''),
			COALESCE(paths.installed_path, ''),
			COALESCE(paths.executable_path, ''),
			COALESCE(paths.executable_sha256, ''),
			COALESCE(paths.cdhash_sha256, ''),
			COALESCE(paths.team_identifier, '')
		FROM software_titles st
		LEFT JOIN software s ON s.title_id = st.id
		LEFT JOIN host_software_installed_paths paths ON paths.software_id = s.id
		WHERE st.id = $1
	`, softwareTitleID)
	if err != nil {
		return softwareFacts{}, err
	}
	defer rows.Close()

	found := false
	acc := newSoftwareFactAccumulator()
	for rows.Next() {
		found = true
		var bundleID string
		var installedPath string
		var executablePath string
		var executableSHA256 string
		var cdhash string
		var teamID string
		if err := rows.Scan(
			&bundleID,
			&installedPath,
			&executablePath,
			&executableSHA256,
			&cdhash,
			&teamID,
		); err != nil {
			return softwareFacts{}, err
		}
		acc.add(bundleID, installedPath, executablePath, executableSHA256, cdhash, teamID)
	}
	if err := rows.Err(); err != nil {
		return softwareFacts{}, err
	}
	if !found {
		return softwareFacts{}, dbutil.ErrNotFound
	}
	return acc.facts(), nil
}

func (acc softwareFactAccumulator) add(
	bundleID string,
	installedPath string,
	executablePath string,
	executableSHA256 string,
	cdhash string,
	teamID string,
) {
	addString(acc.bundleIDs, bundleID)
	addString(acc.paths, installedPath)
	addString(acc.paths, executablePath)
	addString(acc.executableSHA256s, executableSHA256)
	addString(acc.cdhashes, cdhash)
	addString(acc.teamIDs, teamID)
	if teamID != "" && bundleID != "" {
		addString(acc.signingIDs, teamID+":"+bundleID)
	}
}

func (acc softwareFactAccumulator) facts() softwareFacts {
	return softwareFacts{
		bundleIDs:         mapKeys(acc.bundleIDs),
		paths:             mapKeys(acc.paths),
		executableSHA256s: mapKeys(acc.executableSHA256s),
		cdhashes:          mapKeys(acc.cdhashes),
		teamIDs:           mapKeys(acc.teamIDs),
		signingIDs:        mapKeys(acc.signingIDs),
	}
}

func addString(values map[string]struct{}, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	values[value] = struct{}{}
}

func mapKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	return out
}

func (s *Store) executionCounts(ctx context.Context, facts softwareFacts) (int, int, error) {
	var executionCount int
	var blockCount int
	err := s.db.Pool().QueryRow(ctx, softwareMatchedExecutablesCTE+`
		SELECT
			COUNT(DISTINCT ee.id)::integer,
			(COUNT(DISTINCT ee.id) FILTER (WHERE ee.decision::text LIKE 'block_%'))::integer
		FROM santa_execution_events ee
		LEFT JOIN matched_executables me ON me.id = ee.executable_id
		WHERE me.id IS NOT NULL OR ee.file_path = ANY($6::text[])
	`, facts.args()...).Scan(&executionCount, &blockCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, 0, nil
	}
	return executionCount, blockCount, err
}

func (s *Store) bundles(ctx context.Context, facts softwareFacts) ([]BundleReference, error) {
	rows, err := s.db.Pool().Query(ctx, softwareMatchedExecutablesCTE+`
		, matched_bundles AS (
			SELECT DISTINCT b.id
			FROM santa_bundles b
			LEFT JOIN santa_bundle_executables be ON be.bundle_id = b.id
			LEFT JOIN matched_executables me ON me.id = be.executable_id
			WHERE b.bundle_id = ANY($5::text[]) OR me.id IS NOT NULL
		)
		SELECT
			b.sha256,
			b.bundle_id,
			b.name,
			b.path,
			b.version,
			b.version_string,
			b.binary_count,
			COUNT(be.executable_id)::integer,
			b.hash_millis,
			b.uploaded_at,
			b.uploaded_at IS NOT NULL
		FROM matched_bundles mb
		JOIN santa_bundles b ON b.id = mb.id
		LEFT JOIN santa_bundle_executables be ON be.bundle_id = b.id
		GROUP BY b.id
		ORDER BY lower(COALESCE(NULLIF(b.name, ''), b.bundle_id, b.sha256)), b.sha256
	`, facts.args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bundles := []BundleReference{}
	for rows.Next() {
		var bundle BundleReference
		if err := rows.Scan(
			&bundle.SHA256,
			&bundle.BundleID,
			&bundle.Name,
			&bundle.Path,
			&bundle.Version,
			&bundle.VersionString,
			&bundle.BinaryCount,
			&bundle.CollectedBinaryCount,
			&bundle.HashMillis,
			&bundle.UploadedAt,
			&bundle.Complete,
		); err != nil {
			return nil, err
		}
		bundles = append(bundles, bundle)
	}
	return bundles, rows.Err()
}

func (s *Store) executables(ctx context.Context, facts softwareFacts) ([]ExecutableReference, error) {
	rows, err := s.db.Pool().Query(ctx, softwareMatchedExecutablesCTE+`
		SELECT
			e.sha256,
			e.file_name,
			e.file_bundle_id,
			e.file_bundle_name,
			COALESCE(NULLIF(e.file_bundle_version_string, ''), e.file_bundle_version),
			e.signing_id,
			e.team_id,
			e.cdhash,
			COUNT(ee.id)::integer,
			(COUNT(ee.id) FILTER (WHERE ee.decision::text LIKE 'block_%'))::integer
		FROM matched_executables me
		JOIN santa_executables e ON e.id = me.id
		LEFT JOIN santa_execution_events ee ON ee.executable_id = e.id
		GROUP BY e.id
		ORDER BY lower(COALESCE(NULLIF(e.file_bundle_name, ''), NULLIF(e.file_name, ''), e.sha256)), e.sha256
	`, facts.args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	executables := []ExecutableReference{}
	for rows.Next() {
		var executable ExecutableReference
		if err := rows.Scan(
			&executable.SHA256,
			&executable.FileName,
			&executable.BundleID,
			&executable.BundleName,
			&executable.BundleVersion,
			&executable.SigningID,
			&executable.TeamID,
			&executable.CDHash,
			&executable.ExecutionCount,
			&executable.BlockCount,
		); err != nil {
			return nil, err
		}
		executables = append(executables, executable)
	}
	return executables, rows.Err()
}

func (s *Store) signingIdentities(ctx context.Context, facts softwareFacts) ([]SigningIdentityReference, error) {
	rows, err := s.db.Pool().Query(ctx, softwareMatchedExecutablesCTE+`
		, identities AS (
			SELECT
				'teamid'::text AS target_type,
				e.team_id AS identifier,
				COALESCE(NULLIF(e.file_bundle_name, ''), NULLIF(e.file_name, ''), e.team_id) AS name,
				e.id AS executable_id
			FROM matched_executables me
			JOIN santa_executables e ON e.id = me.id
			WHERE e.team_id <> ''
			UNION ALL
			SELECT
				'signingid'::text,
				e.signing_id,
				COALESCE(NULLIF(e.file_bundle_name, ''), NULLIF(e.file_name, ''), e.signing_id),
				e.id
			FROM matched_executables me
			JOIN santa_executables e ON e.id = me.id
			WHERE e.signing_id <> ''
			UNION ALL
			SELECT
				'cdhash'::text,
				e.cdhash,
				COALESCE(NULLIF(e.file_bundle_name, ''), NULLIF(e.file_name, ''), e.cdhash),
				e.id
			FROM matched_executables me
			JOIN santa_executables e ON e.id = me.id
			WHERE e.cdhash <> ''
		)
		SELECT
			i.target_type,
			i.identifier,
			COALESCE(NULLIF(MAX(i.name), ''), i.identifier),
			COUNT(DISTINCT i.executable_id)::integer,
			COUNT(DISTINCT r.id)::integer
		FROM identities i
		LEFT JOIN santa_rules r ON r.rule_type::text = i.target_type AND r.identifier = i.identifier
		GROUP BY i.target_type, i.identifier
		ORDER BY i.target_type, lower(COALESCE(NULLIF(MAX(i.name), ''), i.identifier)), i.identifier
	`, facts.args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	identities := []SigningIdentityReference{}
	for rows.Next() {
		var identity SigningIdentityReference
		if err := rows.Scan(
			&identity.TargetType,
			&identity.Identifier,
			&identity.Name,
			&identity.ExecutableCount,
			&identity.RuleCount,
		); err != nil {
			return nil, err
		}
		identities = append(identities, identity)
	}
	return identities, rows.Err()
}

func (s *Store) certificates(ctx context.Context, facts softwareFacts) ([]CertificateReference, error) {
	rows, err := s.db.Pool().Query(ctx, softwareMatchedExecutablesCTE+`
		, matched_certificates AS (
			SELECT DISTINCT c.id
			FROM matched_executables me
			JOIN santa_executable_signing_chains esc ON esc.executable_id = me.id
			JOIN santa_signing_chain_entries sce ON sce.signing_chain_id = esc.signing_chain_id
			JOIN santa_certificates c ON c.id = sce.certificate_id
		)
		SELECT
			c.sha256,
			c.common_name,
			c.organization,
			c.organizational_unit,
			c.valid_from,
			c.valid_until,
			COUNT(r.id)::integer
		FROM matched_certificates mc
		JOIN santa_certificates c ON c.id = mc.id
		LEFT JOIN santa_rules r ON r.rule_type = 'certificate' AND r.identifier = c.sha256
		GROUP BY c.id
		ORDER BY lower(COALESCE(NULLIF(c.common_name, ''), c.sha256)), c.sha256
	`, facts.args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	certificates := []CertificateReference{}
	for rows.Next() {
		var certificate CertificateReference
		if err := rows.Scan(
			&certificate.SHA256,
			&certificate.CommonName,
			&certificate.Organization,
			&certificate.OrganizationalUnit,
			&certificate.ValidFrom,
			&certificate.ValidUntil,
			&certificate.RuleCount,
		); err != nil {
			return nil, err
		}
		certificates = append(certificates, certificate)
	}
	return certificates, rows.Err()
}

func (s *Store) rules(ctx context.Context, facts softwareFacts) ([]RuleReference, error) {
	rows, err := s.db.Pool().Query(ctx, softwareMatchedExecutablesCTE+`
		, matched_bundles AS (
			SELECT DISTINCT b.id, b.sha256
			FROM santa_bundles b
			LEFT JOIN santa_bundle_executables be ON be.bundle_id = b.id
			LEFT JOIN matched_executables me ON me.id = be.executable_id
			WHERE b.bundle_id = ANY($5::text[]) OR me.id IS NOT NULL
		),
		matched_certificates AS (
			SELECT DISTINCT c.sha256
			FROM matched_executables me
			JOIN santa_executable_signing_chains esc ON esc.executable_id = me.id
			JOIN santa_signing_chain_entries sce ON sce.signing_chain_id = esc.signing_chain_id
			JOIN santa_certificates c ON c.id = sce.certificate_id
		),
		matched_targets AS (
			SELECT 'binary'::text AS target_type, unnest($1::text[]) AS identifier
			UNION
			SELECT 'cdhash'::text, unnest($2::text[])
			UNION
			SELECT 'teamid'::text, unnest($3::text[])
			UNION
			SELECT 'signingid'::text, unnest($4::text[])
			UNION
			SELECT 'binary'::text, e.sha256
			FROM matched_executables me
			JOIN santa_executables e ON e.id = me.id
			WHERE e.sha256 <> ''
			UNION
			SELECT 'cdhash'::text, e.cdhash
			FROM matched_executables me
			JOIN santa_executables e ON e.id = me.id
			WHERE e.cdhash <> ''
			UNION
			SELECT 'teamid'::text, e.team_id
			FROM matched_executables me
			JOIN santa_executables e ON e.id = me.id
			WHERE e.team_id <> ''
			UNION
			SELECT 'signingid'::text, e.signing_id
			FROM matched_executables me
			JOIN santa_executables e ON e.id = me.id
			WHERE e.signing_id <> ''
			UNION
			SELECT 'bundle'::text, sha256
			FROM matched_bundles
			UNION
			SELECT 'certificate'::text, sha256
			FROM matched_certificates
		)
		SELECT
			r.id,
			r.rule_type::text,
			r.identifier,
			r.name,
			r.custom_message,
			r.custom_url
		FROM santa_rules r
		WHERE EXISTS (
			SELECT 1
			FROM matched_targets mt
			WHERE mt.target_type = r.rule_type::text AND mt.identifier = r.identifier
		)
		ORDER BY r.rule_type::text, lower(COALESCE(NULLIF(r.name, ''), r.identifier)), r.identifier
	`, facts.args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules := []RuleReference{}
	for rows.Next() {
		var rule RuleReference
		if err := rows.Scan(
			&rule.ID,
			&rule.RuleType,
			&rule.Identifier,
			&rule.Name,
			&rule.CustomMessage,
			&rule.CustomURL,
		); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

const softwareMatchedExecutablesCTE = `
WITH matched_executables AS (
	SELECT DISTINCT e.id
	FROM santa_executables e
	WHERE
		e.sha256 = ANY($1::text[])
		OR e.cdhash = ANY($2::text[])
		OR e.team_id = ANY($3::text[])
		OR e.signing_id = ANY($4::text[])
		OR e.file_bundle_id = ANY($5::text[])
		OR e.file_bundle_path = ANY($6::text[])
		OR EXISTS (
			SELECT 1
			FROM santa_execution_events ee
			WHERE ee.executable_id = e.id AND ee.file_path = ANY($6::text[])
		)
)`
