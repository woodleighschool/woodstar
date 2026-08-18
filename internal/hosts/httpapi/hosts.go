package httpapi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/activity"
	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/munki/mdp"
)

const hostResource = "host"

type hostListOutput struct {
	Body api.Page[hosts.Host]
}

type hostDetailOutput struct {
	Body hosts.HostDetail
}

type hostGetInput struct {
	ID int64 `path:"id"`
}

type hostListInput struct {
	api.ListQueryInput

	Status          hosts.HostStatus `query:"status,omitempty"`
	LabelID         int64            `query:"label_id,omitempty"`
	SoftwareTitleID int64            `query:"software_title_id,omitempty"`
	SoftwareID      int64            `query:"software_id,omitempty"`
	IDs             []int64          `query:"ids,omitempty"`
}

func (i hostListInput) params() hosts.HostListParams {
	return hosts.HostListParams{
		ListParams:      i.Params(),
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

// RegisterAPI mounts host inventory and host ownership endpoints.
func RegisterAPI(
	routes api.AppRoutes,
	hostStore *hosts.Store,
	primaryUsers *hosts.PrimaryUserStore,
	munkiVersions agentVersionLoader,
	santaVersions agentVersionLoader,
	distribution *mdp.Store,
	geo geoIPLookup,
	activityRecorder activity.Recorder,
	logger *slog.Logger,
) {
	registerAPI(
		routes,
		hostStore,
		primaryUsers,
		munkiVersions,
		santaVersions,
		distribution,
		geo,
		activityRecorder,
		logger,
	)
}

// RegisterOpenAPI documents host endpoints without runtime services.
func RegisterOpenAPI(routes api.AppRoutes) {
	registerAPI(routes, nil, nil, nil, nil, nil, nil, nil, nil)
}

func registerAPI(
	routes api.AppRoutes,
	hostStore *hosts.Store,
	primaryUsers *hosts.PrimaryUserStore,
	munkiVersions agentVersionLoader,
	santaVersions agentVersionLoader,
	distribution *mdp.Store,
	geo geoIPLookup,
	activityRecorder activity.Recorder,
	logger *slog.Logger,
) {
	humaAPI := routes.Ordinary
	registerListHosts(humaAPI, hostStore, munkiVersions, santaVersions, distribution, geo, logger)
	registerGetHost(humaAPI, hostStore, munkiVersions, santaVersions, distribution, geo, logger)
	registerRequestHostInventoryRefresh(humaAPI, hostStore, activityRecorder, logger)
	registerDeleteHost(humaAPI, hostStore, activityRecorder, logger)
	registerBulkDeleteHosts(humaAPI, hostStore, activityRecorder, logger)
	registerSetHostPrimaryUser(
		humaAPI, hostStore, primaryUsers, munkiVersions, santaVersions, distribution, geo, activityRecorder, logger,
	)
	registerClearHostPrimaryUser(
		humaAPI, hostStore, primaryUsers, munkiVersions, santaVersions, distribution, geo, activityRecorder, logger,
	)
}

func registerRequestHostInventoryRefresh(
	humaAPI huma.API,
	hostStore *hosts.Store,
	activityRecorder activity.Recorder,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "request-host-inventory-refresh",
		Method:        http.MethodPost,
		Path:          "/api/hosts/{id}/inventory-refresh",
		Tags:          []string{api.TagHosts},
		Summary:       "Request host inventory refresh",
		DefaultStatus: http.StatusAccepted,
		Errors:        []int{http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*struct{}, error) {
		host, err := hostStore.GetByID(ctx, input.ID)
		if err != nil {
			return nil, api.ResourceError(
				ctx, logger, "request-host-inventory-refresh", hostResource, err, "host_id", input.ID,
			)
		}
		if err := hostStore.RequestInventoryRefresh(ctx, input.ID); err != nil {
			return nil, api.ResourceError(
				ctx,
				logger,
				"request-host-inventory-refresh",
				hostResource,
				err,
				"host_id",
				input.ID,
			)
		}
		activity.RecordUser(ctx, activityRecorder, logger, activity.AreaHosts, activity.ActionHostInventoryRequested,
			activity.Resource(hostResource, host.ID, host.DisplayName))
		return &struct{}{}, nil
	})
}

func registerListHosts(
	humaAPI huma.API,
	hostStore *hosts.Store,
	munkiVersions agentVersionLoader,
	santaVersions agentVersionLoader,
	distribution *mdp.Store,
	geo geoIPLookup,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-hosts",
		Method:      http.MethodGet,
		Path:        "/api/hosts",
		Tags:        []string{api.TagHosts},
		Summary:     "List hosts",
	}, func(ctx context.Context, input *hostListInput) (*hostListOutput, error) {
		params := input.params()
		rows, count, err := hostStore.List(ctx, params)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "list-hosts", hostResource, err)
		}
		if err := enrichHostAgents(ctx, rows, munkiVersions, santaVersions); err != nil {
			return nil, api.HandlerError(ctx, logger, "list-hosts", err)
		}
		enrichHostPublicIPs(ctx, rows, distribution, geo, logger)
		return &hostListOutput{Body: api.Page[hosts.Host]{Items: rows, Count: count}}, nil
	})
}

func registerGetHost(
	humaAPI huma.API,
	hostStore *hosts.Store,
	munkiVersions agentVersionLoader,
	santaVersions agentVersionLoader,
	distribution *mdp.Store,
	geo geoIPLookup,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "get-host",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}",
		Tags:        []string{api.TagHosts},
		Summary:     "Get a host",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*hostDetailOutput, error) {
		body, err := loadHostDetailBody(
			ctx,
			hostStore,
			input.ID,
			munkiVersions,
			santaVersions,
			distribution,
			geo,
			logger,
			"get-host",
		)
		if err != nil {
			return nil, err
		}
		return &hostDetailOutput{Body: *body}, nil
	})
}

func registerSetHostPrimaryUser(
	humaAPI huma.API,
	hostStore *hosts.Store,
	primaryUsers *hosts.PrimaryUserStore,
	munkiVersions agentVersionLoader,
	santaVersions agentVersionLoader,
	distribution *mdp.Store,
	geo geoIPLookup,
	activityRecorder activity.Recorder,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "set-host-primary-user",
		Method:      http.MethodPut,
		Path:        "/api/hosts/{id}/primary-user",
		Tags:        []string{api.TagHosts},
		Summary:     "Set primary user for a host",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusNotFound,
		},
	}, func(ctx context.Context, input *hostPrimaryUserPutInput) (*hostDetailOutput, error) {
		if err := primaryUsers.Upsert(ctx, input.ID, input.Body.Email, hosts.PrimaryUserSourceManual); err != nil {
			return nil, api.ResourceError(ctx, logger, "set-host-primary-user", hostResource, err, "host_id", input.ID)
		}
		body, err := loadHostDetailBody(
			ctx,
			hostStore,
			input.ID,
			munkiVersions,
			santaVersions,
			distribution,
			geo,
			logger,
			"set-host-primary-user",
		)
		if err != nil {
			return nil, err
		}
		activity.RecordUser(ctx, activityRecorder, logger, activity.AreaHosts, activity.ActionHostPrimaryUserSet,
			activity.Resource(hostResource, body.ID, body.DisplayName))
		return &hostDetailOutput{Body: *body}, nil
	})
}

func registerClearHostPrimaryUser(
	humaAPI huma.API,
	hostStore *hosts.Store,
	primaryUsers *hosts.PrimaryUserStore,
	munkiVersions agentVersionLoader,
	santaVersions agentVersionLoader,
	distribution *mdp.Store,
	geo geoIPLookup,
	activityRecorder activity.Recorder,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "clear-host-primary-user",
		Method:      http.MethodDelete,
		Path:        "/api/hosts/{id}/primary-user",
		Tags:        []string{api.TagHosts},
		Summary:     "Clear primary user for a host",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*hostDetailOutput, error) {
		if err := primaryUsers.Delete(ctx, input.ID, hosts.PrimaryUserSourceManual); err != nil {
			return nil, api.ResourceError(ctx, logger, "clear-host-primary-user", hostResource, err, "host_id", input.ID)
		}
		body, err := loadHostDetailBody(
			ctx,
			hostStore,
			input.ID,
			munkiVersions,
			santaVersions,
			distribution,
			geo,
			logger,
			"clear-host-primary-user",
		)
		if err != nil {
			return nil, err
		}
		activity.RecordUser(ctx, activityRecorder, logger, activity.AreaHosts, activity.ActionHostPrimaryUserCleared,
			activity.Resource(hostResource, body.ID, body.DisplayName))
		return &hostDetailOutput{Body: *body}, nil
	})
}

func loadHostDetailBody(
	ctx context.Context,
	hostStore *hosts.Store,
	hostID int64,
	munkiVersions agentVersionLoader,
	santaVersions agentVersionLoader,
	distribution *mdp.Store,
	geo geoIPLookup,
	logger *slog.Logger,
	operation string,
) (*hosts.HostDetail, error) {
	host, err := hostStore.GetByID(ctx, hostID)
	if err != nil {
		return nil, api.ResourceError(ctx, logger, operation, hostResource, err, "host_id", hostID)
	}
	detail, err := hostStore.LoadDetail(ctx, host)
	if err != nil {
		return nil, api.HandlerError(ctx, logger, operation, err, "host_id", hostID)
	}
	rows := []hosts.Host{detail.Host}
	if err := enrichHostAgents(ctx, rows, munkiVersions, santaVersions); err != nil {
		return nil, api.HandlerError(ctx, logger, operation, err, "host_id", hostID)
	}
	enrichHostPublicIPs(ctx, rows, distribution, geo, logger)
	detail.Host = rows[0]
	return detail, nil
}

func registerDeleteHost(
	humaAPI huma.API,
	hostStore *hosts.Store,
	activityRecorder activity.Recorder,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "delete-host",
		Method:      http.MethodDelete,
		Path:        "/api/hosts/{id}",
		Tags:        []string{api.TagHosts},
		Summary:     "Delete a host",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*struct{}, error) {
		host, err := hostStore.GetByID(ctx, input.ID)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "delete-host", hostResource, err, "host_id", input.ID)
		}
		if err := hostStore.Delete(ctx, input.ID); err != nil {
			return nil, api.ResourceError(ctx, logger, "delete-host", hostResource, err, "host_id", input.ID)
		}
		activity.RecordUser(ctx, activityRecorder, logger, activity.AreaHosts, activity.ActionHostDeleted,
			activity.Resource(hostResource, host.ID, host.DisplayName))
		return &struct{}{}, nil
	})
}

func registerBulkDeleteHosts(
	humaAPI huma.API,
	hostStore *hosts.Store,
	activityRecorder activity.Recorder,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "bulk-delete-hosts",
		Method:        http.MethodDelete,
		Path:          "/api/hosts",
		Tags:          []string{api.TagHosts},
		Summary:       "Delete hosts",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *api.DeleteManyInput) (*struct{}, error) {
		deleted, err := hostStore.DeleteMany(ctx, input.IDs)
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "bulk-delete-hosts", err)
		}
		if deleted > 0 {
			activity.RecordUser(ctx, activityRecorder, logger, activity.AreaHosts, activity.ActionHostsDeleted,
				activity.Collection(hostResource, fmt.Sprintf("%d hosts", deleted)))
		}
		return &struct{}{}, nil
	})
}
