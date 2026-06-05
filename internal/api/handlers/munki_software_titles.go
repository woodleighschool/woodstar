package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/munki"
)

const (
	munkiSoftwareTitlePath   = "/api/munki/software-titles"
	munkiSoftwareTitleIDPath = "/api/munki/software-titles/{id}"
	munkiSoftwareTitleLabel  = "Munki software title"
)

type munkiSoftwareTitleGetInput struct {
	ID int64 `path:"id"`
}

type munkiSoftwareTitleCreateInput struct {
	Body munkiSoftwareTitleMutation
}

type munkiSoftwareTitlePatchInput struct {
	ID   int64 `path:"id"`
	Body munkiSoftwareTitleMutation
}

type munkiSoftwareTitleListOutput struct {
	Body Page[munkiSoftwareTitle]
}

type munkiSoftwareTitleOutput struct {
	Body munkiSoftwareTitle
}

type munkiSoftwareTitleDetailOutput struct {
	Body munkiSoftwareTitleDetail
}

type munkiSoftwareTitleMutation struct {
	Name           string `json:"name"`
	DisplayName    string `json:"display_name,omitempty"`
	Description    string `json:"description,omitempty"`
	Category       string `json:"category,omitempty"`
	Developer      string `json:"developer,omitempty"`
	IconArtifactID *int64 `json:"icon_artifact_id,omitempty"`
}

type munkiSoftwareTitleDetail struct {
	ID             int64             `json:"id"`
	Name           string            `json:"name"`
	DisplayName    string            `json:"display_name"`
	Description    string            `json:"description"`
	Category       string            `json:"category"`
	Developer      string            `json:"developer"`
	IconArtifactID *int64            `json:"icon_artifact_id,omitempty"`
	IconURL        string            `json:"icon_url,omitempty"`
	Packages       []munkiPackage    `json:"packages"`
	Assignments    []munkiAssignment `json:"assignments"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

type munkiSoftwareTitle struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	DisplayName    string    `json:"display_name"`
	Description    string    `json:"description"`
	Category       string    `json:"category"`
	Developer      string    `json:"developer"`
	IconArtifactID *int64    `json:"icon_artifact_id,omitempty"`
	IconURL        string    `json:"icon_url,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func registerMunkiSoftwareTitles(api huma.API, store *munki.Store) {
	registerListMunkiSoftwareTitles(api, store)
	registerCreateMunkiSoftwareTitle(api, store)
	registerGetMunkiSoftwareTitle(api, store)
	registerPatchMunkiSoftwareTitle(api, store)
}

func registerListMunkiSoftwareTitles(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-munki-software-titles",
		Method:      http.MethodGet,
		Path:        munkiSoftwareTitlePath,
		Tags:        []string{munkiTag},
		Summary:     "List Munki software titles",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *munkiListInput) (*munkiSoftwareTitleListOutput, error) {
		rows, count, err := store.ListSoftwareTitles(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError(munkiSoftwareTitleLabel, err)
		}
		return &munkiSoftwareTitleListOutput{
			Body: Page[munkiSoftwareTitle]{Items: munkiSoftwareTitlesFromDomain(rows), Count: count},
		}, nil
	})
}

func registerCreateMunkiSoftwareTitle(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-munki-software-title",
		Method:        http.MethodPost,
		Path:          munkiSoftwareTitlePath,
		Tags:          []string{munkiTag},
		Summary:       "Create a Munki software title",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiSoftwareTitleCreateInput) (*munkiSoftwareTitleOutput, error) {
		title, err := store.CreateSoftwareTitle(ctx, input.Body.domain())
		if err != nil {
			return nil, resourceMutationError(munkiSoftwareTitleLabel, err)
		}
		return &munkiSoftwareTitleOutput{Body: munkiSoftwareTitleFromDomain(*title)}, nil
	})
}

func registerGetMunkiSoftwareTitle(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-munki-software-title",
		Method:      http.MethodGet,
		Path:        munkiSoftwareTitleIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Get a Munki software title",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *munkiSoftwareTitleGetInput) (*munkiSoftwareTitleDetailOutput, error) {
		detail, err := store.LoadSoftwareTitleDetail(ctx, input.ID)
		if err != nil {
			return nil, resourceMutationError(munkiSoftwareTitleLabel, err)
		}
		return &munkiSoftwareTitleDetailOutput{Body: munkiSoftwareTitleDetailFromDomain(*detail)}, nil
	})
}

func registerPatchMunkiSoftwareTitle(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "update-munki-software-title",
		Method:      http.MethodPatch,
		Path:        munkiSoftwareTitleIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Update a Munki software title",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiSoftwareTitlePatchInput) (*munkiSoftwareTitleOutput, error) {
		title, err := store.UpdateSoftwareTitle(ctx, input.ID, input.Body.domain())
		if err != nil {
			return nil, resourceMutationError(munkiSoftwareTitleLabel, err)
		}
		return &munkiSoftwareTitleOutput{Body: munkiSoftwareTitleFromDomain(*title)}, nil
	})
}

func munkiSoftwareTitleDetailFromDomain(detail munki.SoftwareTitleDetail) munkiSoftwareTitleDetail {
	return munkiSoftwareTitleDetail{
		ID:             detail.ID,
		Name:           detail.Name,
		DisplayName:    detail.DisplayName,
		Description:    detail.Description,
		Category:       detail.Category,
		Developer:      detail.Developer,
		IconArtifactID: detail.IconArtifactID,
		IconURL:        munkiSoftwareIconURL(detail.SoftwareTitle),
		Packages:       munkiPackagesFromDomain(detail.Packages),
		Assignments:    munkiAssignmentsFromDomain(detail.Assignments),
		CreatedAt:      detail.CreatedAt,
		UpdatedAt:      detail.UpdatedAt,
	}
}

func (body munkiSoftwareTitleMutation) domain() munki.SoftwareTitleMutation {
	return munki.SoftwareTitleMutation{
		Name:           body.Name,
		DisplayName:    body.DisplayName,
		Description:    body.Description,
		Category:       body.Category,
		Developer:      body.Developer,
		IconArtifactID: body.IconArtifactID,
	}
}

func munkiSoftwareTitleFromDomain(title munki.SoftwareTitle) munkiSoftwareTitle {
	return munkiSoftwareTitle{
		ID:             title.ID,
		Name:           title.Name,
		DisplayName:    title.DisplayName,
		Description:    title.Description,
		Category:       title.Category,
		Developer:      title.Developer,
		IconArtifactID: title.IconArtifactID,
		IconURL:        munkiSoftwareIconURL(title),
		CreatedAt:      title.CreatedAt,
		UpdatedAt:      title.UpdatedAt,
	}
}

func munkiSoftwareTitlesFromDomain(rows []munki.SoftwareTitle) []munkiSoftwareTitle {
	items := make([]munkiSoftwareTitle, len(rows))
	for i, row := range rows {
		items[i] = munkiSoftwareTitleFromDomain(row)
	}
	return items
}

func munkiSoftwareIconURL(title munki.SoftwareTitle) string {
	if title.IconArtifactID == nil {
		return ""
	}
	return fmt.Sprintf("/api/munki/artifacts/%d/content", *title.IconArtifactID)
}
