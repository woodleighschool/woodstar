package handlers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/entra"
)

const entraTag = "Entra"

type entraListInput struct {
	ListQueryInput
	Values []string `query:"values,omitempty"`
}

type entraUsersOutput struct {
	Body Page[entra.EntraUser]
}

type entraGroupsOutput struct {
	Body Page[entra.EntraGroup]
}

type entraDepartmentsOutput struct {
	Body Page[entra.EntraDepartment]
}

func (i entraListInput) params() entra.ListParams {
	return entra.ListParams{
		ListParams: i.ListQueryInput.params(),
		Values:     dbutil.SplitListValues(i.Values),
	}
}

func RegisterEntra(api huma.API, store *entra.Store) {
	registerEntraUsers(api, store)
	registerEntraGroups(api, store)
	registerEntraDepartments(api, store)
}

func registerEntraUsers(api huma.API, store *entra.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-entra-users",
		Method:      http.MethodGet,
		Path:        "/api/entra/users",
		Tags:        []string{entraTag},
		Summary:     "List Entra-populated users",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *entraListInput) (*entraUsersOutput, error) {
		rows, count, err := store.ListUsers(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError("entra user", err)
		}
		return &entraUsersOutput{Body: Page[entra.EntraUser]{Items: rows, Count: count}}, nil
	})
}

func registerEntraGroups(api huma.API, store *entra.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-entra-groups",
		Method:      http.MethodGet,
		Path:        "/api/entra/groups",
		Tags:        []string{entraTag},
		Summary:     "List Entra groups",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *entraListInput) (*entraGroupsOutput, error) {
		rows, count, err := store.ListGroups(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError("entra group", err)
		}
		return &entraGroupsOutput{Body: Page[entra.EntraGroup]{Items: rows, Count: count}}, nil
	})
}

func registerEntraDepartments(api huma.API, store *entra.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-entra-departments",
		Method:      http.MethodGet,
		Path:        "/api/entra/departments",
		Tags:        []string{entraTag},
		Summary:     "List Entra-populated departments",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *entraListInput) (*entraDepartmentsOutput, error) {
		rows, count, err := store.ListDepartments(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError("entra department", err)
		}
		return &entraDepartmentsOutput{Body: Page[entra.EntraDepartment]{Items: rows, Count: count}}, nil
	})
}
