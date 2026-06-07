package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/adminapi/adminctx"
	"github.com/woodleighschool/woodstar/internal/adminapi/apitypes"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/inventory"
	"github.com/woodleighschool/woodstar/internal/munki/hoststate"
	"github.com/woodleighschool/woodstar/internal/osquery/checks"
	"github.com/woodleighschool/woodstar/internal/santa"
)

const (
	hostsTag     = "Hosts"
	hostResource = "host"
)

// HostDetail is the host detail response with capability-enriched fields.
// Capability packages contribute their slice through registered enrichers.
type HostDetail struct {
	hosts.HostDetail
	Munki *hoststate.State `json:"munki,omitempty"`
	Santa *santa.HostState `json:"santa,omitempty"`
}

type hostListOutput struct {
	Body apitypes.Page[hosts.Host]
}

type hostDetailOutput struct {
	Body HostDetail
}

type hostSoftwareOutput struct {
	Body apitypes.Page[inventory.HostSoftwareRow]
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

func (i hostListInput) params() hosts.ListParams {
	return hosts.ListParams{
		ListParams:      i.ListQueryInput.Params(),
		Status:          i.Status,
		LabelID:         i.LabelID,
		SoftwareTitleID: i.SoftwareTitleID,
		SoftwareID:      i.SoftwareID,
		IDs:             i.IDs,
	}
}

func (i hostListInput) checkStatusFilter() (checks.CheckStatus, bool, error) {
	hasCheckID := i.CheckID != 0
	hasResponse := i.CheckResponse != ""
	if hasCheckID != hasResponse {
		return "", false, huma.Error400BadRequest("check_id and check_response must be provided together")
	}
	if !hasCheckID {
		return "", false, nil
	}

	switch i.CheckResponse {
	case string(checks.CheckStatusPass):
		return checks.CheckStatusPass, true, nil
	case string(checks.CheckStatusFail):
		return checks.CheckStatusFail, true, nil
	default:
		return "", false, huma.Error400BadRequest("unknown check_response")
	}
}

type hostSoftwareInput struct {
	ID int64 `path:"id"`
	apitypes.ListQueryInput
	Source []string `          query:"source,omitempty"`
}

func (i hostSoftwareInput) params() (int64, inventory.HostSoftwareListParams) {
	return i.ID, inventory.HostSoftwareListParams{
		ListParams:      i.ListQueryInput.Params(),
		SoftwareSources: i.Source,
	}
}

type hostBulkDeleteInput struct {
	Body apitypes.BulkIDsBody
}

type hostUserAffinityPutBody struct {
	Email string `json:"email" format:"email" minLength:"3"`
}

type hostUserAffinityPutInput struct {
	ID   int64 `path:"id"`
	Body hostUserAffinityPutBody
}

// HostDetailContributor adds capability-specific sections to a host detail response.
type HostDetailContributor = hosts.DetailContributor[HostDetail]

// RegisterHosts registers admin host inventory endpoints.
// Reading hosts is open to admins and viewers. Deleting hosts is admin-only.
func RegisterHosts(
	api huma.API,
	hostStore *hosts.Store,
	userAffinities *hosts.UserAffinityStore,
	softwareStore *inventory.Store,
	checkStore *checks.Store,
	contributors ...HostDetailContributor,
) {
	registerListHosts(api, hostStore, checkStore)
	registerGetHost(api, hostStore, contributors)
	registerPutHostUserAffinity(api, hostStore, userAffinities, contributors)
	registerDeleteHostUserAffinity(api, hostStore, userAffinities, contributors)
	registerDeleteHost(api, hostStore)
	registerBulkDeleteHosts(api, hostStore)
	registerHostSoftware(api, hostStore, softwareStore)
}

func registerListHosts(api huma.API, hostStore *hosts.Store, checkStore *checks.Store) {
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
			if checkStore == nil {
				return nil, errors.New("check store is not configured")
			}
			checkHostIDs, err := checkStore.HostIDsByStatus(ctx, input.CheckID, status)
			if err != nil {
				return nil, apitypes.ResourceMutationError("check", err)
			}
			params.IDs = intersectHostIDs(params.IDs, checkHostIDs)
			if len(params.IDs) == 0 {
				return &hostListOutput{Body: apitypes.Page[hosts.Host]{Items: []hosts.Host{}, Count: 0}}, nil
			}
		}

		rows, count, err := hostStore.List(ctx, params)
		if err != nil {
			return nil, apitypes.ResourceMutationError("host", err)
		}
		return &hostListOutput{Body: apitypes.Page[hosts.Host]{Items: rows, Count: count}}, nil
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

func loadHostDetailBody(
	ctx context.Context,
	hostStore *hosts.Store,
	hostID int64,
	contributors []HostDetailContributor,
) (*HostDetail, error) {
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
	body := HostDetail{HostDetail: *detail}
	for _, contributor := range contributors {
		if contributor == nil {
			continue
		}
		if err := contributor.ContributeHostDetail(ctx, hostID, &body); err != nil {
			return nil, err
		}
	}
	return &body, nil
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
		body, err := loadHostDetailBody(ctx, hostStore, input.ID, contributors)
		if err != nil {
			return nil, err
		}
		return &hostDetailOutput{Body: *body}, nil
	})
}

func registerPutHostUserAffinity(
	api huma.API,
	hostStore *hosts.Store,
	userAffinities *hosts.UserAffinityStore,
	contributors []HostDetailContributor,
) {
	huma.Register(api, huma.Operation{
		OperationID: "put-host-user-affinity",
		Method:      http.MethodPut,
		Path:        "/api/hosts/{id}/user-affinity",
		Tags:        []string{hostsTag},
		Summary:     "Set the host user affinity",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
		},
	}, func(ctx context.Context, input *hostUserAffinityPutInput) (*hostDetailOutput, error) {
		if _, err := adminctx.RequireAdmin(ctx); err != nil {
			return nil, err
		}
		if _, err := hostStore.GetByID(ctx, input.ID); errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		} else if err != nil {
			return nil, err
		}
		email := strings.TrimSpace(input.Body.Email)
		if email == "" {
			return nil, huma.Error400BadRequest("email is required")
		}
		if err := userAffinities.Upsert(ctx, input.ID, email, hosts.UserAffinitySourceManual); err != nil {
			return nil, err
		}
		body, err := loadHostDetailBody(ctx, hostStore, input.ID, contributors)
		if err != nil {
			return nil, err
		}
		return &hostDetailOutput{Body: *body}, nil
	})
}

func registerDeleteHostUserAffinity(
	api huma.API,
	hostStore *hosts.Store,
	userAffinities *hosts.UserAffinityStore,
	contributors []HostDetailContributor,
) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-host-user-affinity",
		Method:      http.MethodDelete,
		Path:        "/api/hosts/{id}/user-affinity",
		Tags:        []string{hostsTag},
		Summary:     "Clear the host user affinity",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*hostDetailOutput, error) {
		if _, err := adminctx.RequireAdmin(ctx); err != nil {
			return nil, err
		}
		if _, err := hostStore.GetByID(ctx, input.ID); errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		} else if err != nil {
			return nil, err
		}
		if err := userAffinities.Delete(ctx, input.ID, hosts.UserAffinitySourceManual); err != nil {
			return nil, err
		}
		body, err := loadHostDetailBody(ctx, hostStore, input.ID, contributors)
		if err != nil {
			return nil, err
		}
		return &hostDetailOutput{Body: *body}, nil
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
		if _, err := adminctx.RequireAdmin(ctx); err != nil {
			return nil, err
		}
		if err := hostStore.Delete(ctx, input.ID); err != nil {
			return nil, apitypes.ResourceMutationError("host", err)
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
		if _, err := adminctx.RequireAdmin(ctx); err != nil {
			return nil, err
		}
		if _, err := hostStore.DeleteMany(ctx, input.Body.IDs); err != nil {
			return nil, err
		}
		return &struct{}{}, nil
	})
}

func registerHostSoftware(api huma.API, hostStore *hosts.Store, softwareStore *inventory.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-software",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/software",
		Tags:        []string{hostsTag},
		Summary:     "List software installed on a host",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostSoftwareInput) (*hostSoftwareOutput, error) {
		id, params := input.params()
		if _, err := hostStore.GetByID(ctx, id); errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		} else if err != nil {
			return nil, err
		}
		rows, count, err := softwareStore.ListForHost(ctx, id, params)
		if err != nil {
			return nil, apitypes.ResourceMutationError("software", err)
		}
		return &hostSoftwareOutput{Body: apitypes.Page[inventory.HostSoftwareRow]{Items: rows, Count: count}}, nil
	})
}
