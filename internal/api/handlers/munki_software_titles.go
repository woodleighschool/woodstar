package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki/assignments"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/munki/softwaretitles"
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
	Body softwaretitles.SoftwareTitleMutation
}

type munkiSoftwareTitlePatchInput struct {
	ID   int64 `path:"id"`
	Body softwaretitles.SoftwareTitleMutation
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

type munkiSoftwareTitleDetail struct {
	softwaretitles.SoftwareTitle
	IconURL     string                   `json:"icon_url,omitempty"`
	Packages    []munkiPackage           `json:"packages"`
	Assignments []assignments.Assignment `json:"assignments"`
}

type munkiSoftwareTitle struct {
	softwaretitles.SoftwareTitle
	IconURL string `json:"icon_url,omitempty"`
}

func registerMunkiSoftwareTitles(
	api huma.API,
	store *softwaretitles.Store,
	packageStore *packages.Store,
	assignmentStore *assignments.Store,
) {
	registerListMunkiSoftwareTitles(api, store)
	registerCreateMunkiSoftwareTitle(api, store)
	registerGetMunkiSoftwareTitle(api, store, packageStore, assignmentStore)
	registerPatchMunkiSoftwareTitle(api, store)
}

func registerListMunkiSoftwareTitles(api huma.API, store *softwaretitles.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-munki-software-titles",
		Method:      http.MethodGet,
		Path:        munkiSoftwareTitlePath,
		Tags:        []string{munkiTag},
		Summary:     "List Munki software titles",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *munkiListInput) (*munkiSoftwareTitleListOutput, error) {
		rows, count, err := store.List(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError(munkiSoftwareTitleLabel, err)
		}
		return &munkiSoftwareTitleListOutput{
			Body: Page[munkiSoftwareTitle]{Items: munkiSoftwareTitlesFromDomain(rows), Count: count},
		}, nil
	})
}

func registerCreateMunkiSoftwareTitle(api huma.API, store *softwaretitles.Store) {
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
		title, err := store.Create(ctx, input.Body)
		if err != nil {
			return nil, resourceMutationError(munkiSoftwareTitleLabel, err)
		}
		return &munkiSoftwareTitleOutput{Body: munkiSoftwareTitleFromDomain(*title)}, nil
	})
}

func registerGetMunkiSoftwareTitle(
	api huma.API,
	store *softwaretitles.Store,
	packageStore *packages.Store,
	assignmentStore *assignments.Store,
) {
	huma.Register(api, huma.Operation{
		OperationID: "get-munki-software-title",
		Method:      http.MethodGet,
		Path:        munkiSoftwareTitleIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Get a Munki software title",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *munkiSoftwareTitleGetInput) (*munkiSoftwareTitleDetailOutput, error) {
		title, err := store.GetByID(ctx, input.ID)
		if err != nil {
			return nil, resourceMutationError(munkiSoftwareTitleLabel, err)
		}
		packageRows, _, err := packageStore.List(ctx, packages.PackageListParams{
			ListParams: dbutil.ListParams{PageSize: 1000},
			SoftwareID: input.ID,
		})
		if err != nil {
			return nil, resourceMutationError(munkiPackageLabel, err)
		}
		assignmentRows, _, err := assignmentStore.List(ctx, assignments.AssignmentListParams{
			ListParams: dbutil.ListParams{PageSize: 1000, Sort: "priority.asc"},
			SoftwareID: input.ID,
		})
		if err != nil {
			return nil, resourceMutationError(munkiAssignmentLabel, err)
		}
		return &munkiSoftwareTitleDetailOutput{
			Body: munkiSoftwareTitleDetailFromDomain(*title, packageRows, assignmentRows),
		}, nil
	})
}

func registerPatchMunkiSoftwareTitle(api huma.API, store *softwaretitles.Store) {
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
		title, err := store.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, resourceMutationError(munkiSoftwareTitleLabel, err)
		}
		return &munkiSoftwareTitleOutput{Body: munkiSoftwareTitleFromDomain(*title)}, nil
	})
}

func munkiSoftwareTitleDetailFromDomain(
	title softwaretitles.SoftwareTitle,
	packageRows []packages.Package,
	assignmentRows []assignments.Assignment,
) munkiSoftwareTitleDetail {
	return munkiSoftwareTitleDetail{
		SoftwareTitle: title,
		IconURL:       munkiSoftwareIconURL(title),
		Packages:      munkiPackagesFromDomain(packageRows),
		Assignments:   assignmentRows,
	}
}

func munkiSoftwareTitleFromDomain(title softwaretitles.SoftwareTitle) munkiSoftwareTitle {
	return munkiSoftwareTitle{
		SoftwareTitle: title,
		IconURL:       munkiSoftwareIconURL(title),
	}
}

func munkiSoftwareTitlesFromDomain(rows []softwaretitles.SoftwareTitle) []munkiSoftwareTitle {
	items := make([]munkiSoftwareTitle, len(rows))
	for i, row := range rows {
		items[i] = munkiSoftwareTitleFromDomain(row)
	}
	return items
}

func munkiSoftwareIconURL(title softwaretitles.SoftwareTitle) string {
	if title.IconArtifactID == nil {
		return ""
	}
	return fmt.Sprintf("/api/munki/artifacts/%d/content", *title.IconArtifactID)
}
