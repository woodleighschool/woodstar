package models

import (
	"context"
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
		where += fmt.Sprintf(" AND platform = $%d", len(args))
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

// queryMatchesHost reports whether a query's platform and min osquery version
// constraints are satisfied by host. Empty constraints match every host.
func queryMatchesHost(platform *string, minOsqueryVersion *string, host Host) bool {
	if platform != nil && *platform != "" && *platform != host.Platform {
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
