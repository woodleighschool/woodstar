package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/munki"
)

const (
	munkiAssignmentPath   = "/api/munki/assignments"
	munkiAssignmentIDPath = "/api/munki/assignments/{id}"
	munkiAssignmentLabel  = "Munki assignment"
)

type munkiAssignmentListInput struct {
	ListQueryInput
	SoftwareID int64 `query:"software_id,omitempty"`
}

type munkiAssignmentGetInput struct {
	ID int64 `path:"id"`
}

type munkiAssignmentCreateInput struct {
	Body munkiAssignmentMutation
}

type munkiAssignmentPatchInput struct {
	ID   int64 `path:"id"`
	Body munkiAssignmentMutation
}

type munkiAssignmentReorderInput struct {
	ID   int64 `path:"id"`
	Body munkiAssignmentReorderBody
}

type munkiAssignmentReorderBody struct {
	OrderedIDs []int64 `json:"ordered_ids"`
}

type munkiAssignmentListOutput struct {
	Body Page[munkiAssignment]
}

type munkiAssignmentOutput struct {
	Body munkiAssignment
}

type munkiAssignmentMutation struct {
	SoftwareID       int64                   `json:"software_id"`
	Priority         int32                   `json:"priority"`
	LabelID          int64                   `json:"label_id"`
	Effect           munki.AssignmentEffect  `json:"effect"`
	Action           *munki.AssignmentAction `json:"action,omitempty"`
	OptionalInstall  bool                    `json:"optional_install,omitempty"`
	FeaturedItem     bool                    `json:"featured_item,omitempty"`
	PackageSelection *munki.PackageSelection `json:"package_selection,omitempty"`
	PinnedPackageID  *int64                  `json:"pinned_package_id,omitempty"`
}

type munkiAssignment struct {
	ID                   int64                   `json:"id"`
	SoftwareID           int64                   `json:"software_id"`
	SoftwareDisplayName  string                  `json:"software_display_name"`
	Priority             int32                   `json:"priority"`
	LabelID              int64                   `json:"label_id"`
	Effect               munki.AssignmentEffect  `json:"effect"`
	Action               *munki.AssignmentAction `json:"action,omitempty"`
	OptionalInstall      bool                    `json:"optional_install"`
	FeaturedItem         bool                    `json:"featured_item"`
	PackageSelection     *munki.PackageSelection `json:"package_selection,omitempty"`
	PinnedPackageID      *int64                  `json:"pinned_package_id,omitempty"`
	PinnedPackageName    string                  `json:"pinned_package_name,omitempty"`
	PinnedPackageVersion string                  `json:"pinned_package_version,omitempty"`
	CreatedAt            time.Time               `json:"created_at"`
	UpdatedAt            time.Time               `json:"updated_at"`
}

func (input munkiAssignmentListInput) params() munki.AssignmentListParams {
	return munki.AssignmentListParams{
		ListParams: input.ListQueryInput.params(),
		SoftwareID: input.SoftwareID,
	}
}

func registerMunkiAssignments(api huma.API, store *munki.Store) {
	registerListMunkiAssignments(api, store)
	registerCreateMunkiAssignment(api, store)
	registerGetMunkiAssignment(api, store)
	registerPatchMunkiAssignment(api, store)
	registerReorderMunkiAssignments(api, store)
}

func registerListMunkiAssignments(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-munki-assignments",
		Method:      http.MethodGet,
		Path:        munkiAssignmentPath,
		Tags:        []string{munkiTag},
		Summary:     "List Munki assignments",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *munkiAssignmentListInput) (*munkiAssignmentListOutput, error) {
		rows, count, err := store.ListAssignments(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError(munkiAssignmentLabel, err)
		}
		return &munkiAssignmentListOutput{
			Body: Page[munkiAssignment]{Items: munkiAssignmentsFromDomain(rows), Count: count},
		}, nil
	})
}

func registerCreateMunkiAssignment(api huma.API, store *munki.Store) {
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
		assignment, err := store.CreateAssignment(ctx, input.Body.domain())
		if err != nil {
			return nil, resourceMutationError(munkiAssignmentLabel, err)
		}
		return &munkiAssignmentOutput{Body: munkiAssignmentFromDomain(*assignment)}, nil
	})
}

func registerGetMunkiAssignment(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-munki-assignment",
		Method:      http.MethodGet,
		Path:        munkiAssignmentIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Get a Munki assignment",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *munkiAssignmentGetInput) (*munkiAssignmentOutput, error) {
		assignment, err := store.GetAssignment(ctx, input.ID)
		if err != nil {
			return nil, resourceMutationError(munkiAssignmentLabel, err)
		}
		return &munkiAssignmentOutput{Body: munkiAssignmentFromDomain(*assignment)}, nil
	})
}

func registerPatchMunkiAssignment(api huma.API, store *munki.Store) {
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
		assignment, err := store.UpdateAssignment(ctx, input.ID, input.Body.domain())
		if err != nil {
			return nil, resourceMutationError(munkiAssignmentLabel, err)
		}
		return &munkiAssignmentOutput{Body: munkiAssignmentFromDomain(*assignment)}, nil
	})
}

func registerReorderMunkiAssignments(api huma.API, store *munki.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "reorder-munki-assignments",
		Method:      http.MethodPut,
		Path:        "/api/munki/software-titles/{id}/assignments/order",
		Tags:        []string{munkiTag},
		Summary:     "Reorder Munki assignments",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *munkiAssignmentReorderInput) (*struct{}, error) {
		if err := store.ReorderAssignments(ctx, input.ID, input.Body.OrderedIDs); err != nil {
			return nil, resourceMutationError(munkiAssignmentLabel, err)
		}
		return &struct{}{}, nil
	})
}

func munkiAssignmentFromDomain(assignment munki.Assignment) munkiAssignment {
	return munkiAssignment{
		ID:                   assignment.ID,
		SoftwareID:           assignment.SoftwareID,
		SoftwareDisplayName:  assignment.SoftwareDisplayName,
		Priority:             assignment.Priority,
		LabelID:              assignment.LabelID,
		Effect:               assignment.Effect,
		Action:               assignment.Action,
		OptionalInstall:      assignment.OptionalInstall,
		FeaturedItem:         assignment.FeaturedItem,
		PackageSelection:     assignment.PackageSelection,
		PinnedPackageID:      assignment.PinnedPackageID,
		PinnedPackageName:    assignment.PinnedPackageName,
		PinnedPackageVersion: assignment.PinnedPackageVersion,
		CreatedAt:            assignment.CreatedAt,
		UpdatedAt:            assignment.UpdatedAt,
	}
}

func munkiAssignmentsFromDomain(rows []munki.Assignment) []munkiAssignment {
	items := make([]munkiAssignment, len(rows))
	for i, row := range rows {
		items[i] = munkiAssignmentFromDomain(row)
	}
	return items
}

func (body munkiAssignmentMutation) domain() munki.AssignmentMutation {
	return munki.AssignmentMutation{
		SoftwareID:       body.SoftwareID,
		Priority:         body.Priority,
		LabelID:          body.LabelID,
		Effect:           body.Effect,
		Action:           body.Action,
		OptionalInstall:  body.OptionalInstall,
		FeaturedItem:     body.FeaturedItem,
		PackageSelection: body.PackageSelection,
		PinnedPackageID:  body.PinnedPackageID,
	}
}
