package destination

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
)

func (remote *client) resolveReferences(ctx context.Context, metadata *metadata) error {
	fields, err := object(metadata.pkg.Bytes())
	if err != nil {
		return err
	}
	for field, names := range map[string][]munki.PkginfoReference{"requires": metadata.requires, "update_for": metadata.updateFor} {
		if names == nil {
			continue
		}
		values := make([]packages.PackageReferenceMutation, 0, len(names))
		for _, reference := range names {
			value, err := remote.resolveReference(ctx, reference)
			if err != nil {
				return fmt.Errorf("%s: %w", field, err)
			}
			values = append(values, value)
		}
		fields[field] = raw(values)
	}
	return json.Unmarshal(raw(fields), &metadata.pkg)
}

func (remote *client) resolveReference(ctx context.Context, reference munki.PkginfoReference) (packages.PackageReferenceMutation, error) {
	var result packages.PackageReferenceMutation
	found, err := remote.findSoftware(ctx, 0, reference.Name)
	if err != nil {
		return result, err
	}
	if found == nil {
		return result, fmt.Errorf("unknown software %q", reference.Name)
	}
	result.SoftwareID = found.ID
	if reference.Version == "" {
		return result, nil
	}
	items, err := list[packages.Package](ctx, remote, "/api/munki/packages", url.Values{"software_id": {strconv.FormatInt(found.ID, 10)}, "q": {reference.Version}})
	if err != nil {
		return result, err
	}
	for _, item := range items {
		if item.Software.ID != found.ID || item.Version != reference.Version {
			continue
		}
		if result.PackageID != 0 {
			return result, errors.New("ambiguous dependency package discovery")
		}
		result.PackageID = item.ID
	}
	if result.PackageID == 0 {
		return result, fmt.Errorf("unknown package %q", packages.MunkiVersionedSoftwareName(reference.Name, reference.Version))
	}
	return result, nil
}
