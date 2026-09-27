package destination

import (
	"cmp"
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/woodleighschool/stemma/plugin"

	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/stemma/internal/destination/api"
)

// prune applies retention to the software's whole family as the repository
// holds it, newest first by creation. It keeps the current package and the
// newest others, and leaves any package a target, requirement or update pins.
// Native foreign keys protect reverse references this scope cannot enumerate.
func prune(ctx context.Context, remote *api.Client, metadata metadata, observed api.Observation, apply bool) ([]plugin.Change, error) {
	retention := metadata.controls.Retention
	if retention == nil || observed.Software == nil {
		return nil, nil
	}
	protected := protectedPackages(observed.Software.Targets, observed.Packages)
	var others []packages.Package
	for _, item := range observed.Packages {
		if item.Version != metadata.version {
			others = append(others, item)
		}
	}
	slices.SortFunc(others, func(a, b packages.Package) int {
		return cmp.Or(b.CreatedAt.Compare(a.CreatedAt), cmp.Compare(b.ID, a.ID))
	})
	var changes []plugin.Change
	for _, old := range others[min(retention.Keep-1, len(others)):] {
		if protected[old.ID] {
			continue
		}
		if apply {
			err := remote.DeletePackage(ctx, old.ID)
			var status api.StatusError
			if errors.As(err, &status) && status.Status == http.StatusConflict {
				continue
			}
			if err != nil {
				return changes, err
			}
		}
		changes = append(changes, plugin.Change{Kind: "retention", Field: "package", Action: "delete", Before: raw(old.Version)})
	}
	return changes, nil
}

func protectedPackages(targets software.Targets, items []packages.Package) map[int64]bool {
	protected := map[int64]bool{}
	for _, include := range targets.Include {
		if include.Package.Strategy == software.PackageSpecific && include.Package.PackageID != nil {
			protected[*include.Package.PackageID] = true
		}
	}
	for _, item := range items {
		for _, ref := range append(slices.Clone(item.Requires), item.UpdateFor...) {
			if ref.PackageID > 0 {
				protected[ref.PackageID] = true
			}
		}
	}
	return protected
}
