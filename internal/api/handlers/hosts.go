package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

const hostResource = "host"

type hostListOutput struct {
	Body Page[hosts.Host]
}

type hostDetailOutput struct {
	Body hosts.HostDetail
}

type hostGetInput struct {
	ID int64 `path:"id"`
}

type hostListInput struct {
	ListQueryInput

	Status          string  `query:"status,omitempty"`
	LabelID         int64   `query:"label_id,omitempty"`
	SoftwareTitleID int64   `query:"software_title_id,omitempty"`
	SoftwareID      int64   `query:"software_id,omitempty"`
	IDs             []int64 `query:"ids,omitempty"`
}

func (i hostListInput) params() hosts.HostListParams {
	return hosts.HostListParams{
		ListParams:      i.ListQueryInput.Params(),
		Status:          i.Status,
		LabelID:         i.LabelID,
		SoftwareTitleID: i.SoftwareTitleID,
		SoftwareID:      i.SoftwareID,
		IDs:             i.IDs,
	}
}

type hostBulkDeleteInput struct {
	Body BulkIDsBody
}

type hostPrimaryUserPutBody struct {
	Email string `json:"email" format:"email" minLength:"3"`
}

type hostPrimaryUserPutInput struct {
	ID   int64 `path:"id"`
	Body hostPrimaryUserPutBody
}

func registerHosts(g Groups, deps Dependencies) {
	registerListHosts(g.Ordinary, deps.Hosts)
	registerGetHost(g.Ordinary, deps.Hosts)
	registerDeleteHost(g.Ordinary, deps.Hosts)
	registerBulkDeleteHosts(g.Ordinary, deps.Hosts)
	registerSetHostPrimaryUser(g.Ordinary, deps.Hosts, deps.PrimaryUser)
	registerClearHostPrimaryUser(g.Ordinary, deps.Hosts, deps.PrimaryUser)
}

func registerListHosts(api huma.API, hostStore *hosts.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-hosts",
		Method:      http.MethodGet,
		Path:        "/api/hosts",
		Tags:        []string{hostsTag},
		Summary:     "List enrolled hosts",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *hostListInput) (*hostListOutput, error) {
		params := input.params()
		rows, count, err := hostStore.List(ctx, params)
		if err != nil {
			return nil, ResourceMutationError(hostResource, err)
		}
		return &hostListOutput{Body: Page[hosts.Host]{Items: rows, Count: int32(count)}}, nil
	})
}

func registerGetHost(api huma.API, hostStore *hosts.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-host",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}",
		Tags:        []string{hostsTag},
		Summary:     "Get an enrolled host",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*hostDetailOutput, error) {
		body, err := loadHostDetailBody(ctx, hostStore, input.ID)
		if err != nil {
			return nil, err
		}
		return &hostDetailOutput{Body: *body}, nil
	})
}

func registerSetHostPrimaryUser(api huma.API, hostStore *hosts.Store, primaryUsers *hosts.PrimaryUserStore) {
	huma.Register(api, huma.Operation{
		OperationID: "set-host-primary-user",
		Method:      http.MethodPut,
		Path:        "/api/hosts/{id}/primary-user",
		Tags:        []string{hostsTag},
		Summary:     "Set the host primary user",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
		},
	}, func(ctx context.Context, input *hostPrimaryUserPutInput) (*hostDetailOutput, error) {
		if _, err := hostStore.GetByID(ctx, input.ID); errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		} else if err != nil {
			return nil, err
		}
		email := strings.TrimSpace(input.Body.Email)
		if email == "" {
			return nil, huma.Error400BadRequest("email is required")
		}
		if err := primaryUsers.Upsert(ctx, input.ID, email, hosts.PrimaryUserSourceManual); err != nil {
			return nil, err
		}
		body, err := loadHostDetailBody(ctx, hostStore, input.ID)
		if err != nil {
			return nil, err
		}
		return &hostDetailOutput{Body: *body}, nil
	})
}

func registerClearHostPrimaryUser(api huma.API, hostStore *hosts.Store, primaryUsers *hosts.PrimaryUserStore) {
	huma.Register(api, huma.Operation{
		OperationID: "clear-host-primary-user",
		Method:      http.MethodDelete,
		Path:        "/api/hosts/{id}/primary-user",
		Tags:        []string{hostsTag},
		Summary:     "Clear the host primary user",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*hostDetailOutput, error) {
		if _, err := hostStore.GetByID(ctx, input.ID); errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		} else if err != nil {
			return nil, err
		}
		if err := primaryUsers.Delete(ctx, input.ID, hosts.PrimaryUserSourceManual); err != nil {
			return nil, err
		}
		body, err := loadHostDetailBody(ctx, hostStore, input.ID)
		if err != nil {
			return nil, err
		}
		return &hostDetailOutput{Body: *body}, nil
	})
}

func loadHostDetailBody(ctx context.Context, hostStore *hosts.Store, hostID int64) (*hosts.HostDetail, error) {
	host, err := hostStore.GetByID(ctx, hostID)
	if errors.Is(err, dbutil.ErrNotFound) {
		return nil, huma.Error404NotFound("host not found")
	}
	if err != nil {
		return nil, err
	}
	detail, err := hostStore.LoadDetail(ctx, host)
	if err != nil {
		return nil, err
	}
	return detail, nil
}

func registerDeleteHost(api huma.API, hostStore *hosts.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-host",
		Method:      http.MethodDelete,
		Path:        "/api/hosts/{id}",
		Tags:        []string{hostsTag},
		Summary:     "Delete an enrolled host",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*struct{}, error) {
		if err := hostStore.Delete(ctx, input.ID); err != nil {
			return nil, ResourceMutationError(hostResource, err)
		}
		return &struct{}{}, nil
	})
}

func registerBulkDeleteHosts(api huma.API, hostStore *hosts.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "bulk-delete-hosts",
		Method:      http.MethodPost,
		Path:        "/api/hosts/bulk-delete",
		Tags:        []string{hostsTag},
		Summary:     "Delete enrolled hosts",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *hostBulkDeleteInput) (*struct{}, error) {
		if _, err := hostStore.DeleteMany(ctx, input.Body.IDs); err != nil {
			return nil, err
		}
		return &struct{}{}, nil
	})
}
