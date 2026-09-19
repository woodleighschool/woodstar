package destination

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/munki/software"
)

// SoftwareDetail is a software title with its deployment targets.
type SoftwareDetail struct {
	software.Software

	Targets software.Targets `json:"targets"`
}

// Observation is what the repository holds for a software title: the title,
// every package under it and the one carrying the requested version.
type Observation struct {
	Software *SoftwareDetail
	Package  *packages.Package
	Packages []packages.Package
}

// Observe identifies a publication the way the repository does: software
// names are unique, and so is a version within its software.
func (c *Client) Observe(ctx context.Context, name, version string) (Observation, error) {
	found, err := c.FindSoftware(ctx, name)
	if err != nil || found == nil {
		return Observation{}, err
	}
	result := Observation{Software: found}
	result.Packages, err = list[packages.Package](ctx, c, "/api/munki/packages", url.Values{"software_id": {strconv.FormatInt(found.ID, 10)}})
	if err != nil {
		return result, err
	}
	for _, item := range result.Packages {
		if item.Software.ID != found.ID {
			return result, errors.New("package discovery escaped its software scope")
		}
		if item.Version != version {
			continue
		}
		if result.Package != nil {
			return result, fmt.Errorf("software %q has more than one package with version %s", found.Name, item.Version)
		}
		var detail packages.Package
		if err := c.request(ctx, http.MethodGet, "/api/munki/packages/"+strconv.FormatInt(item.ID, 10), nil, &detail); err != nil {
			return result, err
		}
		if detail.ID != item.ID || detail.Software.ID != found.ID || detail.Version != item.Version {
			return result, errors.New("discovered package identity changed")
		}
		result.Package = &detail
	}
	return result, nil
}

// FindSoftware returns the software with exactly this name, or nil when the
// repository holds none.
func (c *Client) FindSoftware(ctx context.Context, name string) (*SoftwareDetail, error) {
	items, err := list[software.Software](ctx, c, "/api/munki/software", url.Values{"q": {name}})
	if err != nil {
		return nil, err
	}
	var id int64
	for _, item := range items {
		if item.Name != name {
			continue
		}
		if id != 0 {
			return nil, errors.New("ambiguous software discovery")
		}
		if item.ID <= 0 {
			return nil, errors.New("software response has no valid id")
		}
		id = item.ID
	}
	if id == 0 {
		return nil, nil
	}
	var found SoftwareDetail
	if err := c.request(ctx, http.MethodGet, "/api/munki/software/"+strconv.FormatInt(id, 10), nil, &found); err != nil {
		return nil, err
	}
	if found.ID != id || found.Name != name {
		return nil, errors.New("discovered software identity changed")
	}
	return &found, nil
}

// CreateSoftware creates a title holding only its name. The name is the
// title's identity, so a lost reply is found under it.
func (c *Client) CreateSoftware(ctx context.Context, name string) (*SoftwareDetail, error) {
	body := software.CreateMutation{Name: name}
	body.Normalize()
	var created SoftwareDetail
	if err := c.request(ctx, http.MethodPost, "/api/munki/software", body, &created); err != nil {
		found, findErr := c.FindSoftware(ctx, name)
		if findErr != nil || found == nil {
			return nil, err
		}
		created = *found
	}
	if created.ID <= 0 || created.Name != name {
		return nil, errors.New("created software does not match its requested identity")
	}
	return &created, nil
}

// UpdateSoftware applies a sparse patch to the software.
func (c *Client) UpdateSoftware(ctx context.Context, id int64, patch software.Patch) (*SoftwareDetail, error) {
	var saved SoftwareDetail
	if err := c.request(ctx, http.MethodPatch, "/api/munki/software/"+strconv.FormatInt(id, 10), patch, &saved); err != nil {
		return nil, err
	}
	return &saved, nil
}

// CreatePackage creates a package under its software.
func (c *Client) CreatePackage(ctx context.Context, mutation packages.PackageCreateMutation) (*packages.Package, error) {
	var created packages.Package
	if err := c.request(ctx, http.MethodPost, "/api/munki/packages", mutation, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

// UpdatePackage applies a sparse patch to the package.
func (c *Client) UpdatePackage(ctx context.Context, id int64, patch packages.Patch) (*packages.Package, error) {
	var saved packages.Package
	if err := c.request(ctx, http.MethodPatch, "/api/munki/packages/"+strconv.FormatInt(id, 10), patch, &saved); err != nil {
		return nil, err
	}
	return &saved, nil
}

// DeletePackage deletes the package. The repository refuses to delete a
// package that another references, answering with a conflict.
func (c *Client) DeletePackage(ctx context.Context, id int64) error {
	return c.request(ctx, http.MethodDelete, "/api/munki/packages?ids="+strconv.FormatInt(id, 10), nil, nil)
}
