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
	URL        string `json:"url"`
	Name       string `json:"name"`
	SoftwareID int64  `json:"software_id"`
	PackageID  int64  `json:"package_id,omitempty"`
	Version    string `json:"version,omitempty"`
}

type softwareDetail struct {
	software.Software

	Targets software.Targets `json:"targets"`
}

type observation struct {
	Software *softwareDetail
	Package  *packages.Package
}

func (remote *client) observe(ctx context.Context, rawBinding json.RawMessage) (observation, error) {
	var saved binding
	if len(rawBinding) != 0 && strings.TrimSpace(string(rawBinding)) != "null" {
		if err := decode(rawBinding, &saved); err != nil {
			return observation{}, fmt.Errorf("binding: %w", err)
		}
		if saved.Name != remote.config.Name || strings.TrimRight(saved.URL, "/") != strings.TrimRight(remote.config.URL, "/") {
			return observation{}, errors.New("binding belongs to another origin or software name")
		}
	}
	found, err := remote.findSoftware(ctx, saved.SoftwareID, remote.config.Name)
	if err != nil || found == nil {
		return observation{}, err
	}
	result := observation{Software: found}
	items, err := list[packages.Package](ctx, remote, "/api/munki/packages", url.Values{"q": {remote.config.Version}, "software_id": {strconv.FormatInt(found.ID, 10)}})
	if err != nil {
		return result, err
	}
	for _, item := range items {
		if item.Version != remote.config.Version || item.Software.ID != found.ID {
			continue
		}
		if result.Package != nil {
			return result, errors.New("ambiguous package discovery")
		}
		result.Package = &item
	}
	if result.Package != nil {
		endpoint := "/api/munki/packages/" + strconv.FormatInt(result.Package.ID, 10)
		if err := remote.request(ctx, http.MethodGet, endpoint, nil, &result.Package); err != nil {
			return result, err
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
		var status httpError
		if !errors.As(err, &status) || status.status != http.StatusNotFound {
			return nil, err
		}
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
	response := plugin.ReconcileResponse{Changes: changes}
	if observed.Software != nil {
		saved := binding{URL: strings.TrimRight(remote.config.URL, "/"), Name: remote.config.Name, SoftwareID: observed.Software.ID, Version: remote.config.Version}
		if observed.Package != nil {
			saved.PackageID = observed.Package.ID
		}
		response.Binding = raw(saved)
	}
	return response
}
