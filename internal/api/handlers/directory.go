package handlers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/directory"
)

const directoryTag = "Directory"

type directoryListInput struct {
	ListQueryInput
	Values []string `query:"values,omitempty"`
}

type directoryUsersOutput struct {
	Body Page[directory.DirectoryUser]
}

type directoryGroupsOutput struct {
	Body Page[directory.DirectoryGroup]
}

type directoryDepartmentsOutput struct {
	Body Page[directory.Department]
}

func (i directoryListInput) params() directory.ListParams {
	return directory.ListParams{
		ListParams: i.ListQueryInput.params(),
		Values:     dbutil.SplitListValues(i.Values),
	}
}

func RegisterDirectory(api huma.API, store *directory.Store) {
	registerDirectoryUsers(api, store)
	registerDirectoryGroups(api, store)
	registerDirectoryDepartments(api, store)
}

func registerDirectoryUsers(api huma.API, store *directory.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-directory-users",
		Method:      http.MethodGet,
		Path:        "/api/directory/users",
		Tags:        []string{directoryTag},
		Summary:     "List directory users",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *directoryListInput) (*directoryUsersOutput, error) {
		rows, count, err := store.ListUsers(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError("directory user", err)
		}
		return &directoryUsersOutput{Body: Page[directory.DirectoryUser]{Items: rows, Count: count}}, nil
	})
}

func registerDirectoryGroups(api huma.API, store *directory.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-directory-groups",
		Method:      http.MethodGet,
		Path:        "/api/directory/groups",
		Tags:        []string{directoryTag},
		Summary:     "List directory groups",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *directoryListInput) (*directoryGroupsOutput, error) {
		rows, count, err := store.ListGroups(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError("directory group", err)
		}
		return &directoryGroupsOutput{Body: Page[directory.DirectoryGroup]{Items: rows, Count: count}}, nil
	})
}

func registerDirectoryDepartments(api huma.API, store *directory.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-directory-departments",
		Method:      http.MethodGet,
		Path:        "/api/directory/departments",
		Tags:        []string{directoryTag},
		Summary:     "List directory departments",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *directoryListInput) (*directoryDepartmentsOutput, error) {
		rows, count, err := store.ListDepartments(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError("directory department", err)
		}
		return &directoryDepartmentsOutput{Body: Page[directory.Department]{Items: rows, Count: count}}, nil
	})
}
