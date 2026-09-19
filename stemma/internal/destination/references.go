package destination

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/stemma/internal/destination/api"
	"github.com/woodleighschool/woodstar/stemma/internal/destination/pkginfo"
)

// references names the catalog resources the pkginfo links to, so Stemma
// reconciles them on this connection first.
func (m metadata) references() []string {
	var names []string
	for _, refs := range m.links {
		for _, ref := range refs {
			names = append(names, ref.Software)
		}
	}
	slices.Sort(names)
	return slices.Compact(names)
}

// resolveReferences replaces the pkginfo's relationships with the software and
// packages the repository holds for them.
func resolveReferences(ctx context.Context, remote *api.Client, metadata *metadata) error {
	fields, err := object(metadata.pkg.Bytes())
	if err != nil {
		return err
	}
	for field, names := range map[string][]munki.PkginfoReference{"requires": metadata.requires, "update_for": metadata.updateFor} {
		if names == nil {
			continue
		}
		values := make([]packages.PackageReferenceMutation, 0, len(names)+len(metadata.links[field]))
		for _, reference := range names {
			found, err := remote.FindSoftware(ctx, reference.Name)
			if err != nil {
				return fmt.Errorf("%s: %w", field, err)
			}
			if found == nil {
				return fmt.Errorf("%s: unknown software %q", field, reference.Name)
			}
			value, err := remote.PackageReference(ctx, found, reference.Version)
			if err != nil {
				return fmt.Errorf("%s: %w", field, err)
			}
			values = append(values, value)
		}
		for _, link := range metadata.links[field] {
			found, err := linkedSoftware(ctx, remote, link, metadata.peers)
			if err != nil {
				return fmt.Errorf("%s: %w", field, err)
			}
			value, err := remote.PackageReference(ctx, found, link.Version)
			if err != nil {
				return fmt.Errorf("%s: %w", field, err)
			}
			values = append(values, value)
		}
		fields[field] = raw(values)
	}
	return json.Unmarshal(raw(fields), &metadata.pkg)
}

// linkedSoftware finds the software a catalog resource publishes here. Its
// Munki name is its declared name for this destination, or its resource name.
func linkedSoftware(ctx context.Context, remote *api.Client, link pkginfo.CatalogReference, peers map[string]json.RawMessage) (*api.SoftwareDetail, error) {
	declared, exists := peers[link.Software]
	if !exists {
		return nil, fmt.Errorf("catalog software %q does not publish to this destination", link.Software)
	}
	var peer struct {
		Pkginfo struct {
			Name string `json:"name"`
		} `json:"pkginfo"`
	}
	if err := json.Unmarshal(declared, &peer); err != nil {
		return nil, fmt.Errorf("catalog software %q: %w", link.Software, err)
	}
	// The peer publishes under the name the importer normalizes, not the one typed.
	identity := software.CreateMutation{Name: cmp.Or(peer.Pkginfo.Name, link.Software)}
	identity.Normalize()
	name := identity.Name
	found, err := remote.FindSoftware(ctx, name)
	if err != nil {
		return nil, err
	}
	if found == nil {
		return nil, fmt.Errorf("catalog software %q is not published on this connection as %q", link.Software, name)
	}
	return found, nil
}
