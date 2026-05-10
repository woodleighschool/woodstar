package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/software"
	"github.com/woodleighschool/woodstar/internal/store"
)

const (
	softwareTag            = "Software"
	sourceChromeExtensions = "chrome_extensions"
)

type softwareListInput struct {
	Page           int      `query:"page,omitempty"`
	PerPage        int      `query:"per_page,omitempty"`
	Q              string   `query:"q,omitempty"`
	OrderKey       string   `query:"order_key,omitempty"`
	OrderDirection string   `query:"order_direction,omitempty"`
	Source         []string `query:"source,omitempty"`
}

func (i softwareListInput) params() software.SoftwareTitleListParams {
	listParams := store.CleanListParams(store.ListParams{
		Q:              i.Q,
		Page:           i.Page,
		PerPage:        i.PerPage,
		OrderKey:       i.OrderKey,
		OrderDirection: i.OrderDirection,
	})
	return software.SoftwareTitleListParams{
		ListParams:      listParams,
		SoftwareSources: store.SplitListValues(i.Source),
	}
}

type softwareGetInput struct {
	ID string `path:"id"`
}

type softwareTitleBody struct {
	ID               int64                 `json:"id"`
	Name             string                `json:"name"`
	DisplayName      string                `json:"display_name"`
	Source           string                `json:"source"`
	ExtensionFor     string                `json:"extension_for"`
	Browser          string                `json:"browser"`
	BundleIdentifier string                `json:"bundle_identifier,omitempty"`
	HostsCount       int                   `json:"hosts_count"`
	VersionsCount    int                   `json:"versions_count"`
	Versions         []softwareVersionBody `json:"versions"`
	CountsUpdatedAt  *time.Time            `json:"counts_updated_at"`
}

type softwareVersionBody struct {
	ID               int64  `json:"id"`
	Version          string `json:"version"`
	BundleIdentifier string `json:"bundle_identifier,omitempty"`
	HostsCount       int    `json:"hosts_count"`
}

type softwareListBody struct {
	Items []softwareTitleBody `json:"items"`
	Count int                 `json:"count"`
}

type softwareListOutput struct {
	Body softwareListBody
}

type softwareGetBody struct {
	SoftwareTitle softwareTitleBody `json:"software_title"`
}

type softwareGetOutput struct {
	Body softwareGetBody
}

// RegisterSoftware registers admin software inventory endpoints.
func RegisterSoftware(api huma.API, softwareStore *software.SoftwareStore) {
	huma.Register(api, huma.Operation{
		OperationID: "list-software",
		Method:      http.MethodGet,
		Path:        "/api/software",
		Tags:        []string{softwareTag},
		Summary:     "List software titles",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *softwareListInput) (*softwareListOutput, error) {
		titles, count, err := softwareStore.ListTitles(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError("software", err)
		}
		body := softwareListBody{
			Items: make([]softwareTitleBody, 0, len(titles)),
			Count: count,
		}
		for _, title := range titles {
			body.Items = append(body.Items, softwareTitleResponse(title))
		}
		return &softwareListOutput{Body: body}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-software",
		Method:      http.MethodGet,
		Path:        "/api/software/{id}",
		Tags:        []string{softwareTag},
		Summary:     "Get a software title",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *softwareGetInput) (*softwareGetOutput, error) {
		id, err := parseResourceID(input.ID, "software title")
		if err != nil {
			return nil, err
		}
		title, err := softwareStore.GetTitle(ctx, id)
		if errors.Is(err, store.ErrNotFound) {
			return nil, huma.Error404NotFound("software title not found")
		}
		if err != nil {
			return nil, err
		}
		return &softwareGetOutput{Body: softwareGetBody{SoftwareTitle: softwareTitleResponse(*title)}}, nil
	})
}

func softwareTitleResponse(title software.SoftwareTitle) softwareTitleBody {
	versions := make([]softwareVersionBody, 0, len(title.Versions))
	for _, version := range title.Versions {
		versions = append(versions, softwareVersionBody{
			ID:               version.ID,
			Version:          version.Version,
			BundleIdentifier: version.BundleIdentifier,
			HostsCount:       version.HostsCount,
		})
	}
	return softwareTitleBody{
		ID:               title.ID,
		Name:             title.Name,
		DisplayName:      title.DisplayName,
		Source:           title.Source,
		ExtensionFor:     title.ExtensionFor,
		Browser:          browserForSoftware(title.Source, title.ExtensionFor),
		BundleIdentifier: title.BundleIdentifier,
		HostsCount:       title.HostsCount,
		VersionsCount:    title.VersionsCount,
		Versions:         versions,
		CountsUpdatedAt:  title.CountsUpdatedAt,
	}
}

func browserForSoftware(source string, extensionFor string) string {
	switch source {
	case sourceChromeExtensions, "firefox_addons", "safari_extensions":
		return extensionFor
	default:
		return ""
	}
}
