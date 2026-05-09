package models

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/woodleighschool/woodstar/internal/database"
)

// LabelScopeMode describes how a target set uses labels.
type LabelScopeMode string

const (
	ScopeNone       LabelScopeMode = ""
	ScopeIncludeAny LabelScopeMode = "include_any"
	ScopeIncludeAll LabelScopeMode = "include_all"
	ScopeExcludeAny LabelScopeMode = "exclude_any"
)

// LabelScope is the shared label targeting shape for queries, checks, and campaigns.
type LabelScope struct {
	Mode     LabelScopeMode
	LabelIDs []int64
}

// TargetSelection is the live targeting shape.
type TargetSelection struct {
	HostIDs  []int64
	LabelIDs []int64
}

// TargetResolver resolves live target selections against host and label state.
type TargetResolver struct {
	db *database.DB
}

func NewTargetResolver(db *database.DB) *TargetResolver {
	return &TargetResolver{db: db}
}

// NormalizeLabelScope sorts label IDs, removes invalid duplicates, and collapses empty scopes.
func NormalizeLabelScope(scope LabelScope) LabelScope {
	scope.LabelIDs = cleanPositiveIDs(scope.LabelIDs)
	switch scope.Mode {
	case ScopeNone, ScopeIncludeAny, ScopeIncludeAll, ScopeExcludeAny:
	default:
		scope.Mode = ScopeNone
	}
	if len(scope.LabelIDs) == 0 {
		scope.Mode = ScopeNone
	}
	return scope
}

// ResolveSelectedTargets returns active host ids for a live target selection.
func (r *TargetResolver) ResolveSelectedTargets(ctx context.Context, selection TargetSelection) ([]int64, error) {
	hostIDs := cleanPositiveIDs(selection.HostIDs)
	labelIDs := cleanPositiveIDs(selection.LabelIDs)
	if len(labelIDs) == 0 {
		return hostIDs, nil
	}
	if r == nil || r.db == nil {
		return nil, errors.New("target resolver is not configured")
	}
	matches, err := resolveSelectedLabelTargets(ctx, r.db, labelIDs)
	if err != nil {
		return nil, err
	}
	return mergePositiveIDs(hostIDs, matches), nil
}

// HostMatchesScope reports whether a host satisfies a label scope.
func HostMatchesScope(ctx context.Context, db *database.DB, scope LabelScope, hostID int64) (bool, error) {
	scope = NormalizeLabelScope(scope)
	if scope.Mode == ScopeNone {
		return true, nil
	}

	var count int
	err := db.Pool().QueryRow(
		ctx,
		`SELECT count(*)
		 FROM label_membership
		 WHERE host_id = $1 AND label_id = ANY($2)`,
		hostID,
		scope.LabelIDs,
	).Scan(&count)
	if err != nil {
		return false, err
	}

	switch scope.Mode {
	case ScopeNone:
		return true, nil
	case ScopeIncludeAny:
		return count > 0, nil
	case ScopeIncludeAll:
		return count == len(scope.LabelIDs), nil
	case ScopeExcludeAny:
		return count == 0, nil
	default:
		return true, nil
	}
}

// HostsMatchingScope returns active hosts satisfying the given scope and platform.
func HostsMatchingScope(ctx context.Context, db *database.DB, scope LabelScope, platform string) ([]int64, error) {
	scope = NormalizeLabelScope(scope)
	platform = strings.TrimSpace(platform)

	where := "deleted_at IS NULL"
	args := []any{}
	if platform != "" {
		args = append(args, platform)
		where += fmt.Sprintf(` AND (
			platform = $%d
			OR ($%d = 'darwin' AND platform IN ('darwin', 'macos'))
			OR ($%d = 'linux' AND platform <> '' AND platform NOT IN ('darwin', 'macos', 'windows', 'chrome'))
		)`, len(args), len(args), len(args))
	}
	if scope.Mode != ScopeNone {
		args = append(args, scope.LabelIDs)
		switch scope.Mode {
		case ScopeNone:
		case ScopeIncludeAny:
			where += fmt.Sprintf(
				" AND EXISTS (SELECT 1 FROM label_membership lm WHERE lm.host_id = hosts.id AND lm.label_id = ANY($%d))",
				len(args),
			)
		case ScopeIncludeAll:
			where += fmt.Sprintf(
				" AND (SELECT count(*) FROM label_membership lm WHERE lm.host_id = hosts.id AND lm.label_id = ANY($%d)) = %d",
				len(args),
				len(scope.LabelIDs),
			)
		case ScopeExcludeAny:
			where += fmt.Sprintf(
				" AND NOT EXISTS (SELECT 1 FROM label_membership lm WHERE lm.host_id = hosts.id AND lm.label_id = ANY($%d))",
				len(args),
			)
		}
	}

	rows, err := db.Pool().Query(ctx, "SELECT id FROM hosts WHERE "+where+" ORDER BY id", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hostIDs := make([]int64, 0)
	for rows.Next() {
		var hostID int64
		if err := rows.Scan(&hostID); err != nil {
			return nil, err
		}
		hostIDs = append(hostIDs, hostID)
	}
	return hostIDs, rows.Err()
}

func resolveSelectedLabelTargets(ctx context.Context, db *database.DB, labelIDs []int64) ([]int64, error) {
	rows, err := db.Pool().Query(ctx,
		`SELECT id, name, label_type
		 FROM labels
		 WHERE id = ANY($1::bigint[])
		 ORDER BY id`,
		labelIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	builtinIDs := make([]int64, 0, len(labelIDs))
	regularIDs := make([]int64, 0, len(labelIDs))
	for rows.Next() {
		var id int64
		var name string
		var labelType string
		if err := rows.Scan(&id, &name, &labelType); err != nil {
			return nil, err
		}
		if labelType == LabelTypeBuiltin {
			if name == "All Hosts" {
				return allActiveHostIDs(ctx, db)
			}
			builtinIDs = append(builtinIDs, id)
			continue
		}
		regularIDs = append(regularIDs, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	switch {
	case len(builtinIDs) > 0 && len(regularIDs) > 0:
		return hostsMatchingBuiltinAndRegularLabels(ctx, db, builtinIDs, regularIDs)
	case len(builtinIDs) > 0:
		return hostsMatchingAnyLabel(ctx, db, builtinIDs)
	case len(regularIDs) > 0:
		return hostsMatchingAnyLabel(ctx, db, regularIDs)
	default:
		return nil, nil
	}
}

func allActiveHostIDs(ctx context.Context, db *database.DB) ([]int64, error) {
	rows, err := db.Pool().Query(ctx, `SELECT id FROM hosts WHERE deleted_at IS NULL ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHostIDs(rows)
}

func hostsMatchingAnyLabel(ctx context.Context, db *database.DB, labelIDs []int64) ([]int64, error) {
	rows, err := db.Pool().Query(ctx,
		`SELECT DISTINCT h.id
		 FROM hosts h
		 JOIN label_membership lm ON lm.host_id = h.id
		 WHERE h.deleted_at IS NULL AND lm.label_id = ANY($1::bigint[])
		 ORDER BY h.id`,
		labelIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHostIDs(rows)
}

func hostsMatchingBuiltinAndRegularLabels(
	ctx context.Context,
	db *database.DB,
	builtinIDs []int64,
	regularIDs []int64,
) ([]int64, error) {
	rows, err := db.Pool().Query(ctx,
		`SELECT DISTINCT h.id
		 FROM hosts h
		 WHERE h.deleted_at IS NULL
		   AND EXISTS (
		       SELECT 1 FROM label_membership lm
		       WHERE lm.host_id = h.id AND lm.label_id = ANY($1::bigint[])
		   )
		   AND EXISTS (
		       SELECT 1 FROM label_membership lm
		       WHERE lm.host_id = h.id AND lm.label_id = ANY($2::bigint[])
		   )
		 ORDER BY h.id`,
		builtinIDs,
		regularIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHostIDs(rows)
}

type hostIDRows interface {
	Next() bool
	Scan(...any) error
	Err() error
}

func scanHostIDs(rows hostIDRows) ([]int64, error) {
	hostIDs := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		hostIDs = append(hostIDs, id)
	}
	return hostIDs, rows.Err()
}

func mergePositiveIDs(a, b []int64) []int64 {
	seen := make(map[int64]struct{}, len(a)+len(b))
	out := make([]int64, 0, len(a)+len(b))
	for _, id := range a {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	for _, id := range b {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// queryMatchesHost reports whether a query's platform and min osquery version
// constraints are satisfied by host. Empty constraints match every host.
func queryMatchesHost(platform *string, minOsqueryVersion *string, host Host) bool {
	if platform != nil && !platformMatches(*platform, host.Platform) {
		return false
	}
	if minOsqueryVersion != nil && *minOsqueryVersion != "" {
		got := canonicalSemver(host.OsqueryVersion)
		want := canonicalSemver(*minOsqueryVersion)
		if got == "" || want == "" {
			return false
		}
		if semver.Compare(got, want) < 0 {
			return false
		}
	}
	return true
}

// canonicalSemver returns the canonical "v"-prefixed form of a version string,
// or empty if the input is not a valid semver. osquery emits unprefixed
// versions like "5.22.1", which semver.Canonical accepts after we add "v".
func canonicalSemver(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if !strings.HasPrefix(value, "v") {
		value = "v" + value
	}
	return semver.Canonical(value)
}

func cleanPositiveIDs(ids []int64) []int64 {
	out := make([]int64, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}
