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
	Body directoryUsersBody
}

type directoryGroupsOutput struct {
	Body directoryGroupsBody
}

type directoryDepartmentsOutput struct {
	Body directoryDepartmentsBody
}

type directoryUsersBody struct {
	Items []directoryUserBody `json:"items"`
	Count int                 `json:"count"`
}

type directoryGroupsBody struct {
	Items []directoryGroupBody `json:"items"`
	Count int                  `json:"count"`
}

type directoryDepartmentsBody struct {
	Items []directory.Department `json:"items"`
	Count int                    `json:"count"`
}

type directoryUserBody struct {
	ID                int64  `json:"id"`
	ExternalID        string `json:"external_id"`
	UserPrincipalName string `json:"user_principal_name"`
	Mail              string `json:"mail,omitempty"`
	MailNickname      string `json:"mail_nickname,omitempty"`
	DisplayName       string `json:"display_name"`
	GivenName         string `json:"given_name,omitempty"`
	FamilyName        string `json:"family_name,omitempty"`
	Department        string `json:"department,omitempty"`
	Active            bool   `json:"active"`
}

type directoryGroupBody struct {
	ID           int64  `json:"id"`
	ExternalID   string `json:"external_id"`
	DisplayName  string `json:"display_name"`
	MailNickname string `json:"mail_nickname,omitempty"`
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
		return &directoryUsersOutput{Body: directoryUsersBody{Items: directoryUserBodies(rows), Count: count}}, nil
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
		return &directoryGroupsOutput{Body: directoryGroupsBody{Items: directoryGroupBodies(rows), Count: count}}, nil
	})
}

func directoryUserBodies(rows []directory.User) []directoryUserBody {
	out := make([]directoryUserBody, len(rows))
	for i, row := range rows {
		out[i] = directoryUserBody{
			ID:                row.ID,
			ExternalID:        row.ExternalID,
			UserPrincipalName: row.UserPrincipalName,
			Mail:              row.Mail,
			MailNickname:      row.MailNickname,
			DisplayName:       row.DisplayName,
			GivenName:         row.GivenName,
			FamilyName:        row.FamilyName,
			Department:        row.Department,
			Active:            row.Active,
		}
	}
	return out
}

func directoryGroupBodies(rows []directory.Group) []directoryGroupBody {
	out := make([]directoryGroupBody, len(rows))
	for i, row := range rows {
		out[i] = directoryGroupBody{
			ID:           row.ID,
			ExternalID:   row.ExternalID,
			DisplayName:  row.DisplayName,
			MailNickname: row.MailNickname,
		}
	}
	return out
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
		return &directoryDepartmentsOutput{Body: directoryDepartmentsBody{Items: rows, Count: count}}, nil
	})
}
