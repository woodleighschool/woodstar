package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/software"
)

const (
	hostsTag     = "Hosts"
	hostResource = "host"
)

// hostDetailBody is the host detail response with capability-enriched fields.
// Capability packages contribute their slice through registered enrichers.
type hostDetailBody struct {
	hosts.HostDetail
	Santa *santa.HostState `json:"santa,omitempty"`
}

type hostListOutput struct {
	Body paginatedBody[hosts.Host]
}

type hostDetailOutput struct {
	Body hostDetailBody
}

type hostSoftwareOutput struct {
	Body paginatedBody[software.HostSoftwareRow]
}

type hostGetInput struct {
	ID string `path:"id"`
}

type hostListInput struct {
	ListQueryInput
	Status          string `query:"status,omitempty"`
	LabelID         string `query:"label_id,omitempty"`
	SoftwareTitleID string `query:"software_title_id,omitempty"`
	SoftwareID      string `query:"software_id,omitempty"`
}

func (i hostListInput) params() (hosts.ListParams, error) {
	titleID, err := parseOptionalPositiveID(i.SoftwareTitleID, "software_title_id")
	if err != nil {
		return hosts.ListParams{}, err
	}
	softwareID, err := parseOptionalPositiveID(i.SoftwareID, "software_id")
	if err != nil {
		return hosts.ListParams{}, err
	}
	labelID, err := parseOptionalPositiveID(i.LabelID, "label_id")
	if err != nil {
		return hosts.ListParams{}, err
	}
	return hosts.ListParams{
		ListParams:      i.ListQueryInput.params(),
		Status:          i.Status,
		LabelID:         labelID,
		SoftwareTitleID: titleID,
		SoftwareID:      softwareID,
	}, nil
}

type hostSoftwareInput struct {
	ID string `path:"id"`
	ListQueryInput
	Source []string `          query:"source,omitempty"`
}

func (i hostSoftwareInput) params() (int64, software.HostSoftwareListParams, error) {
	id, err := parseResourceID(i.ID, hostResource)
	if err != nil {
		return 0, software.HostSoftwareListParams{}, err
	}
	return id, software.HostSoftwareListParams{
		ListParams:      i.ListQueryInput.params(),
		SoftwareSources: i.Source,
	}, nil
}

type hostBulkDeleteInput struct {
	Body bulkIDsBody
}

// HostDetailContributor adds capability-specific sections to a host detail response.
type HostDetailContributor = hosts.DetailContributor[hostDetailBody]

// RegisterHosts registers admin host inventory endpoints.
// Reading hosts is open to admins and viewers. Deleting hosts is admin-only.
func RegisterHosts(
	api huma.API,
	hostStore *hosts.Store,
	softwareStore *software.Store,
	contributors ...HostDetailContributor,
) {
	registerListHosts(api, hostStore)
	registerGetHost(api, hostStore, contributors)
	registerDeleteHost(api, hostStore)
	registerBulkDeleteHosts(api, hostStore)
	registerHostSoftware(api, hostStore, softwareStore)
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
		params, err := input.params()
		if err != nil {
			return nil, err
		}
		rows, count, err := hostStore.List(ctx, params)
		if err != nil {
			return nil, resourceMutationError("host", err)
		}
		return &hostListOutput{Body: paginatedBody[hosts.Host]{Items: rows, Count: count}}, nil
	})
}

func registerGetHost(api huma.API, hostStore *hosts.Store, contributors []HostDetailContributor) {
	huma.Register(api, huma.Operation{
		OperationID: "get-host",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}",
		Tags:        []string{hostsTag},
		Summary:     "Get an enrolled host",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*hostDetailOutput, error) {
		id, err := parseResourceID(input.ID, hostResource)
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
		detail, err := hostStore.LoadDetail(ctx, host)
		if err != nil {
			return nil, err
		}
		body := hostDetailBody{HostDetail: *detail}
		for _, contributor := range contributors {
			if contributor == nil {
				continue
			}
			if err := contributor.ContributeHostDetail(ctx, id, &body); err != nil {
				return nil, err
			}
		}
		return &hostDetailOutput{Body: body}, nil
	})
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
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		id, err := parseResourceID(input.ID, hostResource)
		if err != nil {
			return nil, err
		}
		if err := hostStore.Delete(ctx, id); err != nil {
			return nil, resourceMutationError("host", err)
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
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		ids, err := input.Body.ids("host IDs")
		if err != nil {
			return nil, err
		}
		if _, err := hostStore.DeleteMany(ctx, ids); err != nil {
			return nil, err
		}
		return &struct{}{}, nil
	})
}

func registerHostSoftware(api huma.API, hostStore *hosts.Store, softwareStore *software.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-software",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/software",
		Tags:        []string{hostsTag},
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
			return nil, resourceMutationError("software", err)
		}
		return &hostSoftwareOutput{Body: paginatedBody[software.HostSoftwareRow]{Items: rows, Count: count}}, nil
	})
}
