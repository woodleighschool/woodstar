package destination

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/woodleighschool/stemma/plugin"

	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/munki/software"
)

type binding struct {
	URL          string                 `json:"url"`
	Name         string                 `json:"name"`
	SoftwareID   int64                  `json:"software_id"`
	PackageID    int64                  `json:"package_id,omitempty"`
	Version      string                 `json:"version,omitempty"`
	Packages     map[string]publication `json:"packages,omitempty"`
	Publications plugin.Publications    `json:"publications"`
	Icon         *pendingUpload         `json:"icon,omitempty"`
	Upload       *pendingUpload         `json:"upload,omitempty"`
	Creating     bool                   `json:"creating,omitempty"`
}

type softwareDetail struct {
	software.Software

	Targets software.Targets `json:"targets"`
}

type observation struct {
	Software *softwareDetail
	Package  *packages.Package
}

type publication struct {
	ID      int64    `json:"id"`
	Version string   `json:"version"`
	SHA256  string   `json:"sha256,omitempty"`
	Derived []string `json:"derived,omitempty"`
}

type pendingUpload struct {
	ObjectID int64  `json:"object_id"`
	SHA256   string `json:"sha256"`
	Size     int64  `json:"size"`
	Complete bool   `json:"complete"`
}

func (remote *client) observe(ctx context.Context, rawBinding json.RawMessage) (observation, error) {
	if len(rawBinding) != 0 && strings.TrimSpace(string(rawBinding)) != "null" {
		if err := decode(rawBinding, &remote.state); err != nil {
			return observation{}, fmt.Errorf("binding: %w", err)
		}
	}
	saved := &remote.state
	if saved.Creating {
		return observation{}, errors.New("prior create has an uncertain result; restore its durable binding before retrying")
	}
	if saved.Name != "" && (saved.Name != remote.config.Name || strings.TrimRight(saved.URL, "/") != strings.TrimRight(remote.config.URL, "/")) {
		return observation{}, errors.New("binding belongs to another origin or software name")
	}
	saved.Name, saved.URL = remote.config.Name, strings.TrimRight(remote.config.URL, "/")
	if saved.Packages == nil {
		saved.Packages = map[string]publication{}
	}
	found, err := remote.findSoftware(ctx, saved.SoftwareID, remote.config.Name)
	if err != nil || found == nil {
		return observation{}, err
	}
	if saved.SoftwareID == 0 {
		return observation{}, errors.New("software exists without a durable binding; restore its binding before publishing")
	}
	result := observation{Software: found}
	owned, known := saved.Packages[remote.fingerprint]
	items, err := list[packages.Package](ctx, remote, "/api/munki/packages", url.Values{"software_id": {strconv.FormatInt(found.ID, 10)}})
	if err != nil {
		return result, err
	}
	for _, item := range items {
		if item.Software.ID != found.ID {
			return result, errors.New("package discovery escaped its software scope")
		}
		if known && item.ID == owned.ID {
			result.Package = &item
		}
		if item.Version == remote.config.Version && (!known || item.ID != owned.ID) {
			return result, errors.New("native package version exists without the expected owned payload binding")
		}
	}
	if known && result.Package == nil {
		return result, errors.New("bound package no longer exists")
	}
	if result.Package != nil {
		endpoint := "/api/munki/packages/" + strconv.FormatInt(result.Package.ID, 10)
		if err := remote.request(ctx, http.MethodGet, endpoint, nil, &result.Package); err != nil {
			return result, err
		}
		pkg := result.Package
		if !owned.matches(*pkg, found.ID) {
			return result, errors.New("bound package no longer matches its owned identity and payload")
		}
	}
	return result, nil
}

func (remote *client) findSoftware(ctx context.Context, id int64, name string) (*softwareDetail, error) {
	if id > 0 {
		var found softwareDetail
		err := remote.request(ctx, http.MethodGet, "/api/munki/software/"+strconv.FormatInt(id, 10), nil, &found)
		if err == nil {
			if found.ID != id || found.Name != name {
				return nil, errors.New("bound software no longer matches its immutable name")
			}
			return &found, nil
		}
		return nil, err
	}
	items, err := list[software.Software](ctx, remote, "/api/munki/software", url.Values{"q": {name}})
	if err != nil {
		return nil, err
	}
	id = 0
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
	var found softwareDetail
	if err := remote.request(ctx, http.MethodGet, "/api/munki/software/"+strconv.FormatInt(id, 10), nil, &found); err != nil {
		return nil, err
	}
	if found.ID != id || found.Name != name {
		return nil, errors.New("discovered software identity changed")
	}
	return &found, nil
}

func list[T any](ctx context.Context, remote *client, endpoint string, query url.Values) ([]T, error) {
	var items []T
	query.Set("per_page", "1000")
	for page := 1; page <= 10000; page++ {
		query.Set("page", strconv.Itoa(page))
		var response struct {
			Items []T `json:"items"`
			Count int `json:"count"`
		}
		if err := remote.request(ctx, http.MethodGet, endpoint+"?"+query.Encode(), nil, &response); err != nil {
			return nil, err
		}
		items = append(items, response.Items...)
		if len(items) >= response.Count {
			return items, nil
		}
		if len(response.Items) == 0 {
			return nil, errors.New("incomplete discovery page")
		}
	}
	return nil, errors.New("discovery exceeds page limit")
}

func (remote *client) response(observed observation, changes []plugin.Change) plugin.ReconcileResponse {
	if observed.Software != nil {
		remote.state.SoftwareID = observed.Software.ID
	}
	if observed.Package != nil {
		remote.state.PackageID, remote.state.Version = observed.Package.ID, observed.Package.Version
	}
	return plugin.ReconcileResponse{Changes: changes, Binding: raw(remote.state), Origins: remote.origins}
}

func (p publication) matches(pkg packages.Package, softwareID int64) bool {
	return pkg.ID == p.ID && pkg.Software.ID == softwareID && pkg.Version == p.Version && (p.SHA256 == "" || pkg.InstallerFile != nil && pkg.InstallerFile.SHA256 == p.SHA256)
}
