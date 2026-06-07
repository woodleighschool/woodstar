package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki/assignments"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
)

const (
	munkiSoftwareTitlePath   = "/api/munki/software-titles"
	munkiSoftwareTitleIDPath = "/api/munki/software-titles/{id}"
	munkiSoftwareTitleLabel  = "Munki software title"
	munkiAssignmentLabel     = "Munki assignment"
	munkiExcludesLabel       = "Munki assignment exclude labels"
)

type munkiSoftwareTitleGetInput struct {
	ID int64 `path:"id"`
}

type munkiSoftwareTitleCreateInput struct {
	Body munkisoftware.SoftwareTitleMutation
}

type munkiSoftwareTitlePatchInput struct {
	ID   int64 `path:"id"`
	Body munkisoftware.SoftwareTitleMutation
}

type munkiSoftwareTitleDeleteInput struct {
	ID int64 `path:"id"`
}

type munkiSoftwareTitleBulkDeleteInput struct {
	Body bulkIDsBody
}

type munkiSoftwareTitleListOutput struct {
	Body Page[munkiSoftwareTitle]
}

type munkiSoftwareTitleDetailOutput struct {
	Body munkiSoftwareTitleDetail
}

type munkiSoftwareTitleDetail struct {
	munkisoftware.SoftwareTitle
	IconURL         string                   `json:"icon_url,omitempty"`
	Packages        []munkiPackage           `json:"packages"`
	Includes        []assignments.Assignment `json:"includes"`
	ExcludeLabelIDs []int64                  `json:"exclude_label_ids"`
}

type munkiSoftwareTitle struct {
	munkisoftware.SoftwareTitle
	IconURL string `json:"icon_url,omitempty"`
}

func registerMunkiSoftwareTitles(
	api huma.API,
	store *munkisoftware.Store,
	packageStore *packages.Store,
	assignmentStore *assignments.Store,
) {
	registerListMunkiSoftwareTitles(api, store)
	registerCreateMunkiSoftwareTitle(api, store, packageStore, assignmentStore)
	registerGetMunkiSoftwareTitle(api, store, packageStore, assignmentStore)
	registerPatchMunkiSoftwareTitle(api, store, packageStore, assignmentStore)
	registerDeleteMunkiSoftwareTitle(api, store)
	registerBulkDeleteMunkiSoftwareTitles(api, store)
}

func registerListMunkiSoftwareTitles(api huma.API, store *munkisoftware.Store) {
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

func registerCreateMunkiSoftwareTitle(
	api huma.API,
	store *munkisoftware.Store,
	packageStore *packages.Store,
	assignmentStore *assignments.Store,
) {
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
	}, func(ctx context.Context, input *munkiSoftwareTitleCreateInput) (*munkiSoftwareTitleDetailOutput, error) {
		title, err := store.Create(ctx, input.Body)
		if err != nil {
			return nil, resourceMutationError(munkiSoftwareTitleLabel, err)
		}
		return loadMunkiSoftwareTitleDetail(ctx, title.ID, store, packageStore, assignmentStore)
	})
}

func registerGetMunkiSoftwareTitle(
	api huma.API,
	store *munkisoftware.Store,
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
		return loadMunkiSoftwareTitleDetail(ctx, input.ID, store, packageStore, assignmentStore)
	})
}

func registerPatchMunkiSoftwareTitle(
	api huma.API,
	store *munkisoftware.Store,
	packageStore *packages.Store,
	assignmentStore *assignments.Store,
) {
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
	}, func(ctx context.Context, input *munkiSoftwareTitlePatchInput) (*munkiSoftwareTitleDetailOutput, error) {
		title, err := store.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, resourceMutationError(munkiSoftwareTitleLabel, err)
		}
		return loadMunkiSoftwareTitleDetail(ctx, title.ID, store, packageStore, assignmentStore)
	})
}

func registerDeleteMunkiSoftwareTitle(api huma.API, store *munkisoftware.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-munki-software-title",
		Method:      http.MethodDelete,
		Path:        munkiSoftwareTitleIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Delete a Munki software title",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *munkiSoftwareTitleDeleteInput) (*struct{}, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		if err := store.Delete(ctx, input.ID); err != nil {
			return nil, resourceMutationError(munkiSoftwareTitleLabel, err)
		}
		return &struct{}{}, nil
	})
}

func registerBulkDeleteMunkiSoftwareTitles(api huma.API, store *munkisoftware.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "bulk-delete-munki-software-titles",
		Method:      http.MethodPost,
		Path:        munkiSoftwareTitlePath + "/bulk-delete",
		Tags:        []string{munkiTag},
		Summary:     "Delete Munki software titles",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *munkiSoftwareTitleBulkDeleteInput) (*struct{}, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		if _, err := store.DeleteMany(ctx, input.Body.IDs); err != nil {
			return nil, resourceMutationError(munkiSoftwareTitleLabel, err)
		}
		return &struct{}{}, nil
	})
}

func munkiSoftwareTitleDetailFromDomain(
	title munkisoftware.SoftwareTitle,
	packageRows []packages.Package,
	includeRows []assignments.Assignment,
	excludeLabelIDs []int64,
) munkiSoftwareTitleDetail {
	return munkiSoftwareTitleDetail{
		SoftwareTitle:   title,
		IconURL:         munkiSoftwareIconURL(title),
		Packages:        munkiPackagesFromDomain(packageRows),
		Includes:        includeRows,
		ExcludeLabelIDs: excludeLabelIDs,
	}
}

func loadMunkiSoftwareTitleDetail(
	ctx context.Context,
	id int64,
	store *munkisoftware.Store,
	packageStore *packages.Store,
	assignmentStore *assignments.Store,
) (*munkiSoftwareTitleDetailOutput, error) {
	title, err := store.GetByID(ctx, id)
	if err != nil {
		return nil, resourceMutationError(munkiSoftwareTitleLabel, err)
	}
	packageRows, _, err := packageStore.List(ctx, packages.PackageListParams{
		ListParams: dbutil.ListParams{PageSize: 1000},
		SoftwareID: id,
	})
	if err != nil {
		return nil, resourceMutationError(munkiPackageLabel, err)
	}
	assignmentRows, err := assignmentStore.ListForSoftwareTitle(ctx, id)
	if err != nil {
		return nil, resourceMutationError(munkiAssignmentLabel, err)
	}
	excludeLabelIDs, err := assignmentStore.ExcludeLabelIDs(ctx, id)
	if err != nil {
		return nil, resourceMutationError(munkiExcludesLabel, err)
	}
	return &munkiSoftwareTitleDetailOutput{
		Body: munkiSoftwareTitleDetailFromDomain(*title, packageRows, assignmentRows, excludeLabelIDs),
	}, nil
}

func munkiSoftwareTitleFromDomain(title munkisoftware.SoftwareTitle) munkiSoftwareTitle {
	return munkiSoftwareTitle{
		SoftwareTitle: title,
		IconURL:       munkiSoftwareIconURL(title),
	}
}

func munkiSoftwareTitlesFromDomain(rows []munkisoftware.SoftwareTitle) []munkiSoftwareTitle {
	items := make([]munkiSoftwareTitle, len(rows))
	for i, row := range rows {
		items[i] = munkiSoftwareTitleFromDomain(row)
	}
	return items
}

func munkiSoftwareIconURL(title munkisoftware.SoftwareTitle) string {
	if title.IconArtifactID == nil {
		return ""
	}
	return fmt.Sprintf("/api/munki/artifacts/%d/content", *title.IconArtifactID)
}
