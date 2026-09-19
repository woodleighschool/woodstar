package destination

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/woodleighschool/woodstar/internal/munki/packages"
)

// PackageReference resolves a relationship to the title, or with a version, to
// the title's package holding that version.
func (c *Client) PackageReference(ctx context.Context, title *SoftwareDetail, version string) (packages.PackageReferenceMutation, error) {
	result := packages.PackageReferenceMutation{SoftwareID: title.ID}
	if version == "" {
		return result, nil
	}
	items, err := list[packages.Package](ctx, c, "/api/munki/packages", url.Values{"software_id": {strconv.FormatInt(title.ID, 10)}, "q": {version}})
	if err != nil {
		return result, err
	}
	for _, item := range items {
		if item.Software.ID != title.ID || item.Version != version {
			continue
		}
		if result.PackageID != 0 {
			return result, errors.New("ambiguous dependency package discovery")
		}
		result.PackageID = item.ID
	}
	if result.PackageID == 0 {
		return result, fmt.Errorf("unknown package %q", packages.MunkiVersionedSoftwareName(title.Name, version))
	}
	return result, nil
}
