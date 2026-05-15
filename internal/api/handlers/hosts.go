package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api/apihelpers"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/software"
)

type hostListOutput struct {
	Body hostListBody
}

type hostDetailOutput struct {
	Body hosts.HostDetail
}

type hostSoftwareOutput struct {
	Body hostSoftwareListBody
}

type hostListBody struct {
	Items []hosts.Host `json:"items"`
	Count int          `json:"count"`
}

type hostSoftwareListBody struct {
	Items []software.HostSoftwareRow `json:"items"`
	Count int                        `json:"count"`
}

type hostGetInput struct {
	ID string `path:"id"`
}

type hostListInput struct {
	Q               string `query:"q,omitempty"`
	Page            int    `query:"page,omitempty"`
	PerPage         int    `query:"per_page,omitempty"`
	OrderKey        string `query:"order_key,omitempty"`
	OrderDirection  string `query:"order_direction,omitempty"`
	Status          string `query:"status,omitempty"`
	Platform        string `query:"platform,omitempty"`
	LabelID         string `query:"label_id,omitempty"`
	SoftwareTitleID string `query:"software_title_id,omitempty"`
	SoftwareID      string `query:"software_id,omitempty"`
}

func (i hostListInput) params() (hosts.HostListParams, error) {
	titleID, err := parseOptionalPositiveID(i.SoftwareTitleID, "software_title_id")
	if err != nil {
		return hosts.HostListParams{}, err
	}
	softwareID, err := parseOptionalPositiveID(i.SoftwareID, "software_id")
	if err != nil {
		return hosts.HostListParams{}, err
	}
	labelID, err := parseOptionalPositiveID(i.LabelID, "label_id")
	if err != nil {
		return hosts.HostListParams{}, err
	}
	listParams := dbutil.CleanListParams(dbutil.ListParams{
		Q:              i.Q,
		Page:           i.Page,
		PerPage:        i.PerPage,
		OrderKey:       i.OrderKey,
		OrderDirection: i.OrderDirection,
	})
	return hosts.HostListParams{
		ListParams:      listParams,
		Status:          strings.TrimSpace(i.Status),
		Platform:        strings.TrimSpace(i.Platform),
		LabelID:         labelID,
		SoftwareTitleID: titleID,
		SoftwareID:      softwareID,
	}, nil
}

type hostSoftwareInput struct {
	ID             string   `path:"id"`
	Q              string   `query:"q,omitempty"`
	Page           int      `query:"page,omitempty"`
	PerPage        int      `query:"per_page,omitempty"`
	OrderKey       string   `query:"order_key,omitempty"`
	OrderDirection string   `query:"order_direction,omitempty"`
	Source         []string `query:"source,omitempty"`
}

func (i hostSoftwareInput) params() (int64, software.HostSoftwareListParams, error) {
	id, err := apihelpers.ParseHostID(i.ID)
	if err != nil {
		return 0, software.HostSoftwareListParams{}, err
	}
	listParams := dbutil.CleanListParams(dbutil.ListParams{
		Q:              i.Q,
		Page:           i.Page,
		PerPage:        i.PerPage,
		OrderKey:       i.OrderKey,
		OrderDirection: i.OrderDirection,
	})
	return id, software.HostSoftwareListParams{
		ListParams:      listParams,
		SoftwareSources: dbutil.SplitListValues(i.Source),
	}, nil
}

type hostBulkIDsBody struct {
	IDs []int64 `json:"ids"`
}

type hostBulkDeleteInput struct {
	Body hostBulkIDsBody
}

func (i hostBulkDeleteInput) ids() ([]int64, error) {
	return apihelpers.CleanBulkIDs(i.Body.IDs, "host IDs")
}

// RegisterHosts registers admin host inventory endpoints.
// Reading hosts is open to admins and viewers. Deleting hosts is admin-only.
func RegisterHosts(
	api huma.API,
	hostStore *hosts.Store,
	deviceMappings *hosts.DeviceMappingStore,
	softwareStore *software.Store,
	labelStore *labels.Store,
) {
	registerListHosts(api, hostStore, deviceMappings)
	registerGetHost(api, hostStore, deviceMappings, labelStore)
	registerDeleteHost(api, hostStore)
	registerBulkDeleteHosts(api, hostStore)
	registerHostSoftware(api, hostStore, softwareStore)
}

func registerListHosts(api huma.API, hostStore *hosts.Store, deviceMappings *hosts.DeviceMappingStore) {
	huma.Register(api, huma.Operation{
		OperationID: "list-hosts",
		Method:      http.MethodGet,
		Path:        "/api/hosts",
		Tags:        []string{apihelpers.HostsTag},
		Summary:     "List enrolled hosts",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *hostListInput) (*hostListOutput, error) {
		params, err := input.params()
		if err != nil {
			return nil, err
		}
		rows, count, err := hostStore.List(ctx, params)
		if err != nil {
			return nil, apihelpers.ResourceMutationError("host", err)
		}
		if err := attachDeviceMappings(ctx, deviceMappings, rows); err != nil {
			return nil, err
		}
		return &hostListOutput{Body: hostListBody{Items: rows, Count: count}}, nil
	})
}

// attachDeviceMappings fills DeviceMappings on each host via one bulk query.
func attachDeviceMappings(ctx context.Context, store *hosts.DeviceMappingStore, rows []hosts.Host) error {
	if len(rows) == 0 {
		return nil
	}
	ids := make([]int64, len(rows))
	for i := range rows {
		ids[i] = rows[i].ID
	}
	grouped, err := store.ListForHosts(ctx, ids)
	if err != nil {
		return err
	}
	for i := range rows {
		rows[i].DeviceMappings = grouped[rows[i].ID]
	}
	return nil
}

func registerGetHost(
	api huma.API,
	hostStore *hosts.Store,
	deviceMappings *hosts.DeviceMappingStore,
	labelStore *labels.Store,
) {
	huma.Register(api, huma.Operation{
		OperationID: "get-host",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}",
		Tags:        []string{apihelpers.HostsTag},
		Summary:     "Get an enrolled host",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*hostDetailOutput, error) {
		id, err := apihelpers.ParseHostID(input.ID)
		if err != nil {
			return nil, err
		}
		host, err := hostStore.GetByID(ctx, id)
		if errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		}
		if err != nil {
			return nil, err
		}
		detail, err := loadHostDetail(ctx, hostStore, labelStore, deviceMappings, host)
		if err != nil {
			return nil, err
		}
		return &hostDetailOutput{Body: *detail}, nil
	})
}

func registerDeleteHost(api huma.API, hostStore *hosts.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-host",
		Method:      http.MethodDelete,
		Path:        "/api/hosts/{id}",
		Tags:        []string{apihelpers.HostsTag},
		Summary:     "Delete an enrolled host",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*struct{}, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		id, err := apihelpers.ParseHostID(input.ID)
		if err != nil {
			return nil, err
		}
		if err := hostStore.Delete(ctx, id); err != nil {
			return nil, apihelpers.ResourceMutationError("host", err)
		}
		return &struct{}{}, nil
	})
}

func registerBulkDeleteHosts(api huma.API, hostStore *hosts.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "bulk-delete-hosts",
		Method:      http.MethodPost,
		Path:        "/api/hosts/bulk-delete",
		Tags:        []string{apihelpers.HostsTag},
		Summary:     "Delete enrolled hosts",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *hostBulkDeleteInput) (*struct{}, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		ids, err := input.ids()
		if err != nil {
			return nil, err
		}
		if _, err := hostStore.DeleteMany(ctx, ids); err != nil {
			return nil, err
		}
		return &struct{}{}, nil
	})
}

func loadHostDetail(
	ctx context.Context,
	hostStore *hosts.Store,
	labelStore *labels.Store,
	deviceMappings *hosts.DeviceMappingStore,
	host *hosts.Host,
) (*hosts.HostDetail, error) {
	hostLabels, err := labelStore.ListForHost(ctx, host.ID)
	if err != nil {
		return nil, err
	}
	hostUsers, err := hostStore.ListUsers(ctx, host.ID)
	if err != nil {
		return nil, err
	}
	batteries, err := hostStore.ListBatteries(ctx, host.ID)
	if err != nil {
		return nil, err
	}
	certificates, err := hostStore.ListCertificates(ctx, host.ID)
	if err != nil {
		return nil, err
	}
	mappings, err := deviceMappings.ListForHost(ctx, host.ID)
	if err != nil {
		return nil, err
	}
	host.DeviceMappings = mappings
	return &hosts.HostDetail{
		Host:         *host,
		Labels:       hostLabels,
		Users:        hostUsers,
		Batteries:    batteries,
		Certificates: certificates,
	}, nil
}

func registerHostSoftware(api huma.API, hostStore *hosts.Store, softwareStore *software.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-software",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/software",
		Tags:        []string{apihelpers.HostsTag},
		Summary:     "List software installed on a host",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostSoftwareInput) (*hostSoftwareOutput, error) {
		id, params, err := input.params()
		if err != nil {
			return nil, err
		}
		if _, err := hostStore.GetByID(ctx, id); errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		} else if err != nil {
			return nil, err
		}
		rows, count, err := softwareStore.ListForHost(ctx, id, params)
		if err != nil {
			return nil, apihelpers.ResourceMutationError("software", err)
		}
		return &hostSoftwareOutput{Body: hostSoftwareListBody{Items: rows, Count: count}}, nil
	})
}
