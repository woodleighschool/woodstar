package references

import (
	"context"
	"strings"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	santarules "github.com/woodleighschool/woodstar/internal/santa/rules"
)

// Store loads Santa read models for inventory reference views.
type Store struct {
	q *sqlc.Queries
}

func NewStore(db *database.DB) *Store {
	return &Store{q: db.Queries()}
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
	rows, err := s.q.ListSoftwareReferenceFacts(
		ctx,
		sqlc.ListSoftwareReferenceFactsParams{SoftwareTitleID: softwareTitleID},
	)
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
			row.ExecutableSha256,
			row.CdhashSha256,
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

func softwareReferenceCountParams(facts softwareFacts) sqlc.GetSoftwareReferenceExecutionCountsParams {
	return sqlc.GetSoftwareReferenceExecutionCountsParams{
		Paths:             facts.paths,
		ExecutableSha256s: facts.executableSHA256s,
		Cdhashes:          facts.cdhashes,
		TeamIds:           facts.teamIDs,
		SigningIds:        facts.signingIDs,
		BundleIds:         facts.bundleIDs,
	}
}

func softwareReferenceBundlesParams(facts softwareFacts) sqlc.ListSoftwareReferenceBundlesParams {
	return sqlc.ListSoftwareReferenceBundlesParams{
		ExecutableSha256s: facts.executableSHA256s,
		Cdhashes:          facts.cdhashes,
		TeamIds:           facts.teamIDs,
		SigningIds:        facts.signingIDs,
		BundleIds:         facts.bundleIDs,
		Paths:             facts.paths,
	}
}

func softwareReferenceExecutablesParams(facts softwareFacts) sqlc.ListSoftwareReferenceExecutablesParams {
	return sqlc.ListSoftwareReferenceExecutablesParams{
		ExecutableSha256s: facts.executableSHA256s,
		Cdhashes:          facts.cdhashes,
		TeamIds:           facts.teamIDs,
		SigningIds:        facts.signingIDs,
		BundleIds:         facts.bundleIDs,
		Paths:             facts.paths,
	}
}

func softwareReferenceSigningIdentityParams(facts softwareFacts) sqlc.ListSoftwareReferenceSigningIdentitiesParams {
	return sqlc.ListSoftwareReferenceSigningIdentitiesParams{
		ExecutableSha256s: facts.executableSHA256s,
		Cdhashes:          facts.cdhashes,
		TeamIds:           facts.teamIDs,
		SigningIds:        facts.signingIDs,
		BundleIds:         facts.bundleIDs,
		Paths:             facts.paths,
	}
}

func softwareReferenceCertificatesParams(facts softwareFacts) sqlc.ListSoftwareReferenceCertificatesParams {
	return sqlc.ListSoftwareReferenceCertificatesParams{
		ExecutableSha256s: facts.executableSHA256s,
		Cdhashes:          facts.cdhashes,
		TeamIds:           facts.teamIDs,
		SigningIds:        facts.signingIDs,
		BundleIds:         facts.bundleIDs,
		Paths:             facts.paths,
	}
}

func softwareReferenceRulesParams(facts softwareFacts) sqlc.ListSoftwareReferenceRulesParams {
	return sqlc.ListSoftwareReferenceRulesParams{
		ExecutableSha256s: facts.executableSHA256s,
		Cdhashes:          facts.cdhashes,
		TeamIds:           facts.teamIDs,
		SigningIds:        facts.signingIDs,
		BundleIds:         facts.bundleIDs,
		Paths:             facts.paths,
	}
}

func (s *Store) executionCounts(ctx context.Context, facts softwareFacts) (int32, int32, error) {
	row, err := s.q.GetSoftwareReferenceExecutionCounts(ctx, softwareReferenceCountParams(facts))
	return row.ExecutionCount, row.BlockCount, err
}

func (s *Store) bundles(ctx context.Context, facts softwareFacts) ([]BundleReference, error) {
	rows, err := s.q.ListSoftwareReferenceBundles(ctx, softwareReferenceBundlesParams(facts))
	if err != nil {
		return nil, err
	}
	bundles := make([]BundleReference, len(rows))
	for i, row := range rows {
		bundles[i] = BundleReference{
			SHA256:               row.Sha256,
			BundleID:             row.BundleID,
			Name:                 row.Name,
			Path:                 row.Path,
			Version:              row.Version,
			VersionString:        row.VersionString,
			BinaryCount:          row.BinaryCount,
			CollectedBinaryCount: row.CollectedBinaryCount,
			HashMillis:           row.HashMillis,
			UploadedAt:           row.UploadedAt,
			Complete:             row.Complete,
		}
	}
	return bundles, nil
}

func (s *Store) executables(ctx context.Context, facts softwareFacts) ([]ExecutableReference, error) {
	rows, err := s.q.ListSoftwareReferenceExecutables(ctx, softwareReferenceExecutablesParams(facts))
	if err != nil {
		return nil, err
	}
	executables := make([]ExecutableReference, len(rows))
	for i, row := range rows {
		executables[i] = ExecutableReference{
			SHA256:         row.Sha256,
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
	return executables, nil
}

func (s *Store) signingIdentities(ctx context.Context, facts softwareFacts) ([]SigningIdentityReference, error) {
	rows, err := s.q.ListSoftwareReferenceSigningIdentities(ctx, softwareReferenceSigningIdentityParams(facts))
	if err != nil {
		return nil, err
	}
	identities := make([]SigningIdentityReference, len(rows))
	for i, row := range rows {
		identities[i] = SigningIdentityReference{
			TargetType:      santarules.RuleType(row.TargetType),
			Identifier:      row.Identifier,
			Name:            row.Name,
			ExecutableCount: row.ExecutableCount,
			RuleCount:       row.RuleCount,
		}
	}
	return identities, nil
}

func (s *Store) certificates(ctx context.Context, facts softwareFacts) ([]CertificateReference, error) {
	rows, err := s.q.ListSoftwareReferenceCertificates(ctx, softwareReferenceCertificatesParams(facts))
	if err != nil {
		return nil, err
	}
	certificates := make([]CertificateReference, len(rows))
	for i, row := range rows {
		certificates[i] = CertificateReference{
			SHA256:             row.Sha256,
			CommonName:         row.CommonName,
			Organization:       row.Organization,
			OrganizationalUnit: row.OrganizationalUnit,
			ValidFrom:          row.ValidFrom,
			ValidUntil:         row.ValidUntil,
			RuleCount:          row.RuleCount,
		}
	}
	return certificates, nil
}

func (s *Store) rules(ctx context.Context, facts softwareFacts) ([]RuleReference, error) {
	rows, err := s.q.ListSoftwareReferenceRules(ctx, softwareReferenceRulesParams(facts))
	if err != nil {
		return nil, err
	}
	rules := make([]RuleReference, len(rows))
	for i, row := range rows {
		rules[i] = RuleReference{
			ID:            row.ID,
			RuleType:      santarules.RuleType(row.RuleType),
			Identifier:    row.Identifier,
			Name:          row.Name,
			CustomMessage: row.CustomMessage,
			CustomURL:     row.CustomURL,
		}
	}
	return rules, nil
}
