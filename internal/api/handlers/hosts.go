package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

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

	Status          hosts.HostStatus `query:"status,omitempty"`
	LabelID         int64            `query:"label_id,omitempty"`
	SoftwareTitleID int64            `query:"software_title_id,omitempty"`
	SoftwareID      int64            `query:"software_id,omitempty"`
	IDs             []int64          `query:"ids,omitempty"`
}

func (i hostListInput) params() hosts.HostListParams {
	return hosts.HostListParams{
		ListParams:      i.ListQueryInput.params(),
		Status:          i.Status,
		LabelID:         i.LabelID,
		SoftwareTitleID: i.SoftwareTitleID,
		SoftwareID:      i.SoftwareID,
		IDs:             i.IDs,
	}
}

type hostPrimaryUserPutBody struct {
	Email string `json:"email" format:"email" minLength:"3"`
}

type hostPrimaryUserPutInput struct {
	ID   int64 `path:"id"`
	Body hostPrimaryUserPutBody
}

// RegisterHosts mounts host inventory and host ownership endpoints.
func RegisterHosts(
	api huma.API,
	hostStore *hosts.Store,
	primaryUsers *hosts.PrimaryUserStore,
	logger *slog.Logger,
) {
	registerListHosts(api, hostStore, logger)
	registerGetHost(api, hostStore, logger)
	registerDeleteHost(api, hostStore, logger)
	registerBulkDeleteHosts(api, hostStore, logger)
	registerSetHostPrimaryUser(api, hostStore, primaryUsers, logger)
	registerClearHostPrimaryUser(api, hostStore, primaryUsers, logger)
}

func registerListHosts(api huma.API, hostStore *hosts.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "list-hosts",
		Method:      http.MethodGet,
		Path:        "/api/hosts",
		Tags:        []string{hostsTag},
		Summary:     "List enrolled hosts",
	}, func(ctx context.Context, input *hostListInput) (*hostListOutput, error) {
		params := input.params()
		rows, count, err := hostStore.List(ctx, params)
		if err != nil {
			return nil, resourceError(ctx, logger, "list-hosts", hostResource, err)
		}
		return &hostListOutput{Body: Page[hosts.Host]{Items: rows, Count: count}}, nil
	})
}

func registerGetHost(api huma.API, hostStore *hosts.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "get-host",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}",
		Tags:        []string{hostsTag},
		Summary:     "Get an enrolled host",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*hostDetailOutput, error) {
		body, err := loadHostDetailBody(ctx, hostStore, input.ID, logger, "get-host")
		if err != nil {
			return nil, err
		}
		return &hostDetailOutput{Body: *body}, nil
	})
}

func registerSetHostPrimaryUser(
	api huma.API,
	hostStore *hosts.Store,
	primaryUsers *hosts.PrimaryUserStore,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID: "set-host-primary-user",
		Method:      http.MethodPut,
		Path:        "/api/hosts/{id}/primary-user",
		Tags:        []string{hostsTag},
		Summary:     "Set the host primary user",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusNotFound,
		},
	}, func(ctx context.Context, input *hostPrimaryUserPutInput) (*hostDetailOutput, error) {
		if err := primaryUsers.Upsert(ctx, input.ID, input.Body.Email, hosts.PrimaryUserSourceManual); err != nil {
			return nil, resourceError(ctx, logger, "set-host-primary-user", hostResource, err, "host_id", input.ID)
		}
		body, err := loadHostDetailBody(ctx, hostStore, input.ID, logger, "set-host-primary-user")
		if err != nil {
			return nil, err
		}
		return &hostDetailOutput{Body: *body}, nil
	})
}

func registerClearHostPrimaryUser(
	api huma.API,
	hostStore *hosts.Store,
	primaryUsers *hosts.PrimaryUserStore,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID: "clear-host-primary-user",
		Method:      http.MethodDelete,
		Path:        "/api/hosts/{id}/primary-user",
		Tags:        []string{hostsTag},
		Summary:     "Clear the host primary user",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*hostDetailOutput, error) {
		if err := primaryUsers.Delete(ctx, input.ID, hosts.PrimaryUserSourceManual); err != nil {
			return nil, resourceError(ctx, logger, "clear-host-primary-user", hostResource, err, "host_id", input.ID)
		}
		body, err := loadHostDetailBody(ctx, hostStore, input.ID, logger, "clear-host-primary-user")
		if err != nil {
			return nil, err
		}
		return &hostDetailOutput{Body: *body}, nil
	})
}

func loadHostDetailBody(
	ctx context.Context,
	hostStore *hosts.Store,
	hostID int64,
	logger *slog.Logger,
	operation string,
) (*hosts.HostDetail, error) {
	host, err := hostStore.GetByID(ctx, hostID)
	if err != nil {
		return nil, resourceError(ctx, logger, operation, hostResource, err, "host_id", hostID)
	}
	detail, err := hostStore.LoadDetail(ctx, host)
	if err != nil {
		return nil, handlerError(ctx, logger, operation, err, "host_id", hostID)
	}
	return detail, nil
}

func registerDeleteHost(api huma.API, hostStore *hosts.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-host",
		Method:      http.MethodDelete,
		Path:        "/api/hosts/{id}",
		Tags:        []string{hostsTag},
		Summary:     "Delete an enrolled host",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*struct{}, error) {
		if err := hostStore.Delete(ctx, input.ID); err != nil {
			return nil, resourceError(ctx, logger, "delete-host", hostResource, err, "host_id", input.ID)
		}
		return &struct{}{}, nil
	})
}

func registerBulkDeleteHosts(api huma.API, hostStore *hosts.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID:   "bulk-delete-hosts",
		Method:        http.MethodDelete,
		Path:          "/api/hosts",
		Tags:          []string{hostsTag},
		Summary:       "Delete enrolled hosts",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *deleteManyInput) (*struct{}, error) {
		if _, err := hostStore.DeleteMany(ctx, input.IDs); err != nil {
			return nil, handlerError(ctx, logger, "bulk-delete-hosts", err)
		}
		return &struct{}{}, nil
	})
}
