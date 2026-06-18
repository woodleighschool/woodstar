package hosts

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/apitypes"
)

const (
	hostsTag      = "Hosts"
	hostResource  = "host"
	checkResource = "check"
)

// CheckStatusFilter returns host IDs matching an external check status.
type CheckStatusFilter interface {
	HostIDsByStatus(context.Context, int64, string) ([]int64, error)
}

// AdminRoutesOptions groups dependencies for host admin routes.
type AdminRoutesOptions struct {
	Store       *Store
	CheckFilter CheckStatusFilter
}

type hostListOutput struct {
	Body apitypes.Page[Host]
}

type hostGetInput struct {
	ID int64 `path:"id"`
}

type hostListInput struct {
	apitypes.ListQueryInput

	Status          string  `query:"status,omitempty"`
	LabelID         int64   `query:"label_id,omitempty"`
	SoftwareTitleID int64   `query:"software_title_id,omitempty"`
	SoftwareID      int64   `query:"software_id,omitempty"`
	IDs             []int64 `query:"ids,omitempty"`
	CheckID         int64   `query:"check_id,omitempty"          minimum:"1"`
	CheckResponse   string  `query:"check_response,omitempty"                enum:"pass,fail"`
}

func (i hostListInput) params() HostListParams {
	return HostListParams{
		ListParams:      i.ListQueryInput.Params(),
		Status:          i.Status,
		LabelID:         i.LabelID,
		SoftwareTitleID: i.SoftwareTitleID,
		SoftwareID:      i.SoftwareID,
		IDs:             i.IDs,
	}
}

func (i hostListInput) checkStatusFilter() (string, bool, error) {
	hasCheckID := i.CheckID != 0
	hasResponse := i.CheckResponse != ""
	if hasCheckID != hasResponse {
		return "", false, huma.Error400BadRequest("check_id and check_response must be provided together")
	}
	if !hasCheckID {
		return "", false, nil
	}

	switch i.CheckResponse {
	case "pass", "fail":
		return i.CheckResponse, true, nil
	default:
		return "", false, huma.Error400BadRequest("unknown check_response")
	}
}

type hostBulkDeleteInput struct {
	Body apitypes.BulkIDsBody
}

// RegisterAdminRoutes registers admin host inventory endpoints.
func RegisterAdminRoutes(api huma.API, opts AdminRoutesOptions) {
	registerListHosts(api, opts.Store, opts.CheckFilter)
	registerDeleteHost(api, opts.Store)
	registerBulkDeleteHosts(api, opts.Store)
}

func registerListHosts(api huma.API, hostStore *Store, checkFilter CheckStatusFilter) {
	huma.Register(api, huma.Operation{
		OperationID: "list-hosts",
		Method:      http.MethodGet,
		Path:        "/api/hosts",
		Tags:        []string{hostsTag},
		Summary:     "List enrolled hosts",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *hostListInput) (*hostListOutput, error) {
		params := input.params()
		status, hasCheckFilter, err := input.checkStatusFilter()
		if err != nil {
			return nil, err
		}
		if hasCheckFilter {
			if checkFilter == nil {
				return nil, errors.New("check store is not configured")
			}
			checkHostIDs, err := checkFilter.HostIDsByStatus(ctx, input.CheckID, status)
			if err != nil {
				return nil, apitypes.ResourceMutationError(checkResource, err)
			}
			params.IDs = intersectHostIDs(params.IDs, checkHostIDs)
			if len(params.IDs) == 0 {
				return &hostListOutput{Body: apitypes.Page[Host]{Items: []Host{}, Count: 0}}, nil
			}
		}

		rows, count, err := hostStore.List(ctx, params)
		if err != nil {
			return nil, apitypes.ResourceMutationError(hostResource, err)
		}
		return &hostListOutput{Body: apitypes.Page[Host]{Items: rows, Count: int32(count)}}, nil
	})
}

func intersectHostIDs(requestedIDs []int64, checkHostIDs []int64) []int64 {
	if len(requestedIDs) == 0 {
		return checkHostIDs
	}
	if len(checkHostIDs) == 0 {
		return nil
	}

	checkIDs := make(map[int64]struct{}, len(checkHostIDs))
	for _, id := range checkHostIDs {
		checkIDs[id] = struct{}{}
	}
	ids := make([]int64, 0, len(requestedIDs))
	for _, id := range requestedIDs {
		if _, ok := checkIDs[id]; ok {
			ids = append(ids, id)
		}
	}
	return ids
}

func registerDeleteHost(api huma.API, hostStore *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-host",
		Method:      http.MethodDelete,
		Path:        "/api/hosts/{id}",
		Tags:        []string{hostsTag},
		Summary:     "Delete an enrolled host",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*struct{}, error) {
		if err := hostStore.Delete(ctx, input.ID); err != nil {
			return nil, apitypes.ResourceMutationError(hostResource, err)
		}
		return &struct{}{}, nil
	})
}

func registerBulkDeleteHosts(api huma.API, hostStore *Store) {
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
