package references

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	santarules "github.com/woodleighschool/woodstar/internal/santa/rules"
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

type softwareFactRow struct {
	BundleIdentifier string `db:"bundle_identifier"`
	InstalledPath    string `db:"installed_path"`
	ExecutablePath   string `db:"executable_path"`
	ExecutableSHA256 string `db:"executable_sha256"`
	CdhashSHA256     string `db:"cdhash_sha256"`
	TeamIdentifier   string `db:"team_identifier"`
}

func (s *Store) loadSoftwareFacts(ctx context.Context, softwareTitleID int64) (softwareFacts, error) {
	qrows, err := s.db.Pool().
		Query(ctx, `
SELECT
    COALESCE(s.bundle_identifier, '') AS bundle_identifier,
    COALESCE(paths.installed_path, '') AS installed_path,
    COALESCE(paths.executable_path, '') AS executable_path,
    COALESCE(paths.executable_sha256, '') AS executable_sha256,
    COALESCE(paths.cdhash_sha256, '') AS cdhash_sha256,
    COALESCE(paths.team_identifier, '') AS team_identifier
FROM software_titles st
LEFT JOIN software s ON s.title_id = st.id
LEFT JOIN host_software_installed_paths paths ON paths.software_id = s.id
WHERE st.id = @software_title_id`,
			pgx.NamedArgs{"software_title_id": softwareTitleID},
		)
	if err != nil {
		return softwareFacts{}, err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[softwareFactRow])
	if err != nil {
		return softwareFacts{}, err
	}
	if len(rows) == 0 {
		return softwareFacts{}, dbutil.ErrNotFound
	}
	acc := newSoftwareFactAccumulator()
	for _, row := range rows {
		acc.add(
			row.BundleIdentifier,
			row.InstalledPath,
			row.ExecutablePath,
			row.ExecutableSHA256,
			row.CdhashSHA256,
			row.TeamIdentifier,
		)
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

func factsArgs(facts softwareFacts) pgx.NamedArgs {
	return pgx.NamedArgs{
		"executable_sha256s": facts.executableSHA256s,
		"cdhashes":           facts.cdhashes,
		"team_ids":           facts.teamIDs,
		"signing_ids":        facts.signingIDs,
		"bundle_ids":         facts.bundleIDs,
		"paths":              facts.paths,
	}
}

type executionCountRow struct {
	ExecutionCount int32 `db:"execution_count"`
	BlockCount     int32 `db:"block_count"`
}

func (s *Store) executionCounts(ctx context.Context, facts softwareFacts) (int32, int32, error) {
	row, err := dbutil.GetOne[executionCountRow](
		ctx,
		s.db.Pool(),
		matchedExecutablesCTE()+`
SELECT
    COUNT(DISTINCT ee.id)::integer AS execution_count,
    (COUNT(DISTINCT ee.id) FILTER (WHERE ee.decision::text LIKE 'block_%'))::integer AS block_count
FROM santa_execution_events ee
LEFT JOIN matched_executables me ON me.id = ee.executable_id
WHERE me.id IS NOT NULL OR ee.file_path = ANY(@paths::text[])`,
		factsArgs(facts),
	)
	if err != nil {
		return 0, 0, err
	}
	return row.ExecutionCount, row.BlockCount, nil
}

func (s *Store) bundles(ctx context.Context, facts softwareFacts) ([]BundleReference, error) {
	qrows, err := s.db.Pool().Query(ctx, matchedExecutablesCTE()+`,
matched_bundles AS (
    SELECT DISTINCT b.id
    FROM santa_bundles b
    LEFT JOIN santa_bundle_executables be ON be.bundle_id = b.id
    LEFT JOIN matched_executables me ON me.id = be.executable_id
    WHERE b.bundle_id = ANY(@bundle_ids::text[]) OR me.id IS NOT NULL
)
SELECT
    b.sha256,
    b.bundle_id,
    b.name,
    b.path,
    b.version,
    b.version_string,
    b.binary_count,
    COUNT(be.executable_id)::integer AS collected_binary_count,
    b.hash_millis,
    b.uploaded_at,
    (b.uploaded_at IS NOT NULL)::boolean AS complete
FROM matched_bundles mb
JOIN santa_bundles b ON b.id = mb.id
LEFT JOIN santa_bundle_executables be ON be.bundle_id = b.id
GROUP BY b.id
ORDER BY lower(COALESCE(NULLIF(b.name, ''), b.bundle_id, b.sha256)), b.sha256`, factsArgs(facts))
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(qrows, pgx.RowToStructByName[BundleReference])
}

type executableReferenceRow struct {
	SHA256         string `db:"sha256"`
	FileName       string `db:"file_name"`
	FileBundleID   string `db:"file_bundle_id"`
	FileBundleName string `db:"file_bundle_name"`
	BundleVersion  string `db:"bundle_version"`
	SigningID      string `db:"signing_id"`
	TeamID         string `db:"team_id"`
	Cdhash         string `db:"cdhash"`
	ExecutionCount int32  `db:"execution_count"`
	BlockCount     int32  `db:"block_count"`
}

func executableFromRow(row executableReferenceRow) ExecutableReference {
	return ExecutableReference{
		SHA256:         row.SHA256,
		FileName:       row.FileName,
		BundleID:       row.FileBundleID,
		BundleName:     row.FileBundleName,
		BundleVersion:  row.BundleVersion,
		SigningID:      row.SigningID,
		TeamID:         row.TeamID,
		CDHash:         row.Cdhash,
		ExecutionCount: row.ExecutionCount,
		BlockCount:     row.BlockCount,
	}
}

func (s *Store) executables(ctx context.Context, facts softwareFacts) ([]ExecutableReference, error) {
	qrows, err := s.db.Pool().Query(ctx, matchedExecutablesCTE()+`
SELECT
    e.sha256,
    e.file_name,
    e.file_bundle_id,
    e.file_bundle_name,
    COALESCE(NULLIF(e.file_bundle_version_string, ''), e.file_bundle_version) AS bundle_version,
    e.signing_id,
    e.team_id,
    e.cdhash,
    COUNT(ee.id)::integer AS execution_count,
    (COUNT(ee.id) FILTER (WHERE ee.decision::text LIKE 'block_%'))::integer AS block_count
FROM matched_executables me
JOIN santa_executables e ON e.id = me.id
LEFT JOIN santa_execution_events ee ON ee.executable_id = e.id
GROUP BY e.id
ORDER BY lower(COALESCE(NULLIF(e.file_bundle_name, ''), NULLIF(e.file_name, ''), e.sha256)), e.sha256`, factsArgs(facts))
	if err != nil {
		return nil, err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[executableReferenceRow])
	if err != nil {
		return nil, err
	}
	executables := make([]ExecutableReference, len(rows))
	for i, row := range rows {
		executables[i] = executableFromRow(row)
	}
	return executables, nil
}

type signingIdentityReferenceRow struct {
	RuleType        string `db:"rule_type"`
	Identifier      string `db:"identifier"`
	ExecutableCount int32  `db:"executable_count"`
	RuleCount       int32  `db:"rule_count"`
}

func signingIdentityFromRow(row signingIdentityReferenceRow) SigningIdentityReference {
	return SigningIdentityReference{
		RuleType:        santarules.RuleType(row.RuleType),
		Identifier:      row.Identifier,
		ExecutableCount: row.ExecutableCount,
		RuleCount:       row.RuleCount,
	}
}

func (s *Store) signingIdentities(ctx context.Context, facts softwareFacts) ([]SigningIdentityReference, error) {
	qrows, err := s.db.Pool().Query(ctx, matchedExecutablesCTE()+`,
identities AS (
    SELECT
        'teamid'::text AS rule_type,
        e.team_id AS identifier,
        e.id AS executable_id
    FROM matched_executables me
    JOIN santa_executables e ON e.id = me.id
    WHERE e.team_id <> ''
    UNION ALL
    SELECT
        'signingid'::text,
        e.signing_id,
        e.id
    FROM matched_executables me
    JOIN santa_executables e ON e.id = me.id
    WHERE e.signing_id <> ''
    UNION ALL
    SELECT
        'cdhash'::text,
        e.cdhash,
        e.id
    FROM matched_executables me
    JOIN santa_executables e ON e.id = me.id
    WHERE e.cdhash <> ''
)
SELECT
    i.rule_type,
    i.identifier,
    COUNT(DISTINCT i.executable_id)::integer AS executable_count,
    COUNT(DISTINCT r.id)::integer AS rule_count
FROM identities i
LEFT JOIN santa_rules r ON r.rule_type::text = i.rule_type AND r.identifier = i.identifier
GROUP BY i.rule_type, i.identifier
ORDER BY i.rule_type, i.identifier`, factsArgs(facts))
	if err != nil {
		return nil, err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[signingIdentityReferenceRow])
	if err != nil {
		return nil, err
	}
	identities := make([]SigningIdentityReference, len(rows))
	for i, row := range rows {
		identities[i] = signingIdentityFromRow(row)
	}
	return identities, nil
}

func (s *Store) certificates(ctx context.Context, facts softwareFacts) ([]CertificateReference, error) {
	qrows, err := s.db.Pool().Query(ctx, matchedExecutablesCTE()+`,
matched_certificates AS (
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
    COUNT(r.id)::integer AS rule_count
FROM matched_certificates mc
JOIN santa_certificates c ON c.id = mc.id
LEFT JOIN santa_rules r ON r.rule_type = 'certificate' AND r.identifier = c.sha256
GROUP BY c.id
ORDER BY lower(COALESCE(NULLIF(c.common_name, ''), c.sha256)), c.sha256`, factsArgs(facts))
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(qrows, pgx.RowToStructByName[CertificateReference])
}

func matchedExecutablesCTE() string {
	return `
WITH matched_executables AS (
    SELECT DISTINCT e.id
    FROM santa_executables e
    WHERE
        e.sha256 = ANY(@executable_sha256s::text[])
        OR e.cdhash = ANY(@cdhashes::text[])
        OR e.team_id = ANY(@team_ids::text[])
        OR e.signing_id = ANY(@signing_ids::text[])
        OR e.file_bundle_id = ANY(@bundle_ids::text[])
        OR e.file_bundle_path = ANY(@paths::text[])
        OR EXISTS (
            SELECT 1
            FROM santa_execution_events ee
            WHERE ee.executable_id = e.id AND ee.file_path = ANY(@paths::text[])
        )
)`
}
