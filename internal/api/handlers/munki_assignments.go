package handlers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/munki/assignments"
)

const (
	munkiAssignmentPath   = "/api/munki/assignments"
	munkiAssignmentIDPath = "/api/munki/assignments/{id}"
	munkiAssignmentLabel  = "Munki assignment"
	munkiExcludesLabel    = "Munki assignment exclude labels"
)

type munkiAssignmentListInput struct {
	ListQueryInput
	SoftwareID int64 `query:"software_id,omitempty"`
}

type munkiAssignmentGetInput struct {
	ID int64 `path:"id"`
}

type munkiAssignmentCreateInput struct {
	Body assignments.AssignmentMutation
}

type munkiAssignmentPatchInput struct {
	ID   int64 `path:"id"`
	Body assignments.AssignmentMutation
}

type munkiAssignmentReorderInput struct {
	ID   int64 `path:"id"`
	Body munkiAssignmentReorderBody
}

type munkiAssignmentReorderBody struct {
	OrderedIDs []int64 `json:"ordered_ids"`
}

type munkiAssignmentExcludesInput struct {
	ID   int64 `path:"id"`
	Body munkiAssignmentExcludesBody
}

type munkiAssignmentExcludesBody struct {
	ExcludeLabelIDs []int64 `json:"exclude_label_ids"`
}

type munkiAssignmentListOutput struct {
	Body Page[assignments.Assignment]
}

type munkiAssignmentOutput struct {
	Body assignments.Assignment
}

type munkiAssignmentExcludesOutput struct {
	Body munkiAssignmentExcludesBody
}

func (input munkiAssignmentListInput) params() assignments.AssignmentListParams {
	return assignments.AssignmentListParams{
		ListParams: input.ListQueryInput.params(),
		SoftwareID: input.SoftwareID,
	}
}

func registerMunkiAssignments(api huma.API, store *assignments.Store) {
	registerListMunkiAssignments(api, store)
	registerCreateMunkiAssignment(api, store)
	registerGetMunkiAssignment(api, store)
	registerPatchMunkiAssignment(api, store)
	registerReorderMunkiAssignments(api, store)
	registerUpdateMunkiAssignmentExcludes(api, store)
}

func registerListMunkiAssignments(api huma.API, store *assignments.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-munki-assignments",
		Method:      http.MethodGet,
		Path:        munkiAssignmentPath,
		Tags:        []string{munkiTag},
		Summary:     "List Munki assignments",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *munkiAssignmentListInput) (*munkiAssignmentListOutput, error) {
		rows, count, err := store.List(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError(munkiAssignmentLabel, err)
		}
		return &munkiAssignmentListOutput{
			Body: Page[assignments.Assignment]{Items: rows, Count: count},
		}, nil
	})
}

func registerCreateMunkiAssignment(api huma.API, store *assignments.Store) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-munki-assignment",
		Method:        http.MethodPost,
		Path:          munkiAssignmentPath,
		Tags:          []string{munkiTag},
		Summary:       "Create a Munki assignment",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiAssignmentCreateInput) (*munkiAssignmentOutput, error) {
		assignment, err := store.Create(ctx, input.Body)
		if err != nil {
			return nil, resourceMutationError(munkiAssignmentLabel, err)
		}
		return &munkiAssignmentOutput{Body: *assignment}, nil
	})
}

func registerGetMunkiAssignment(api huma.API, store *assignments.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-munki-assignment",
		Method:      http.MethodGet,
		Path:        munkiAssignmentIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Get a Munki assignment",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *munkiAssignmentGetInput) (*munkiAssignmentOutput, error) {
		assignment, err := store.GetByID(ctx, input.ID)
		if err != nil {
			return nil, resourceMutationError(munkiAssignmentLabel, err)
		}
		return &munkiAssignmentOutput{Body: *assignment}, nil
	})
}

func registerPatchMunkiAssignment(api huma.API, store *assignments.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "update-munki-assignment",
		Method:      http.MethodPatch,
		Path:        munkiAssignmentIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Update a Munki assignment",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiAssignmentPatchInput) (*munkiAssignmentOutput, error) {
		assignment, err := store.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, resourceMutationError(munkiAssignmentLabel, err)
		}
		return &munkiAssignmentOutput{Body: *assignment}, nil
	})
}

func registerReorderMunkiAssignments(api huma.API, store *assignments.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "reorder-munki-assignment-includes",
		Method:      http.MethodPut,
		Path:        "/api/munki/software-titles/{id}/includes/order",
		Tags:        []string{munkiTag},
		Summary:     "Reorder Munki assignment includes",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *munkiAssignmentReorderInput) (*struct{}, error) {
		if err := store.Reorder(ctx, input.ID, input.Body.OrderedIDs); err != nil {
			return nil, resourceMutationError(munkiAssignmentLabel, err)
		}
		return &struct{}{}, nil
	})
}

func registerUpdateMunkiAssignmentExcludes(api huma.API, store *assignments.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "update-munki-assignment-exclude-labels",
		Method:      http.MethodPut,
		Path:        "/api/munki/software-titles/{id}/exclude-labels",
		Tags:        []string{munkiTag},
		Summary:     "Update Munki assignment exclude labels",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *munkiAssignmentExcludesInput) (*munkiAssignmentExcludesOutput, error) {
		excludeLabelIDs, err := store.ReplaceExcludeLabelIDs(ctx, input.ID, input.Body.ExcludeLabelIDs)
		if err != nil {
			return nil, resourceMutationError(munkiExcludesLabel, err)
		}
		return &munkiAssignmentExcludesOutput{
			Body: munkiAssignmentExcludesBody{ExcludeLabelIDs: excludeLabelIDs},
		}, nil
	})
}
