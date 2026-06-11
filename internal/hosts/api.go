package hosts

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/adminapi/apitypes"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

const (
	hostsTag     = "Hosts"
	hostResource = "host"
)

// CheckStatusFilter returns host IDs matching an external check status.
type CheckStatusFilter interface {
	HostIDsByStatus(context.Context, int64, string) ([]int64, error)
}

type DetailBuilder[T any] func(HostDetail) T

// AdminRoutesOptions groups dependencies for host admin routes.
type AdminRoutesOptions[T any] struct {
	Store          *Store
	UserAffinities *UserAffinityStore
	RequireAdmin   func(context.Context) error
	CheckFilter    CheckStatusFilter
	DetailBuilder  DetailBuilder[T]
	Contributors   []DetailContributor[T]
}

type hostListOutput struct {
	Body apitypes.Page[Host]
}

type hostDetailOutput[T any] struct {
	Body T
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

type hostUserAffinityPutBody struct {
	Email string `json:"email" format:"email" minLength:"3"`
}

type hostUserAffinityPutInput struct {
	ID   int64 `path:"id"`
	Body hostUserAffinityPutBody
}

// RegisterAdminRoutes registers admin host inventory endpoints.
// Reading hosts is open to admins and viewers. Deleting hosts is admin-only.
func RegisterAdminRoutes[T any](api huma.API, opts AdminRoutesOptions[T]) {
	registerListHosts(api, opts.Store, opts.CheckFilter)
	registerGetHost(api, opts)
	registerPutHostUserAffinity(api, opts)
	registerDeleteHostUserAffinity(api, opts)
	registerDeleteHost(api, opts.Store, opts.RequireAdmin)
	registerBulkDeleteHosts(api, opts.Store, opts.RequireAdmin)
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
				return nil, apitypes.ResourceMutationError("check", err)
			}
			params.IDs = intersectHostIDs(params.IDs, checkHostIDs)
			if len(params.IDs) == 0 {
				return &hostListOutput{Body: apitypes.Page[Host]{Items: []Host{}, Count: 0}}, nil
			}
		}

		rows, count, err := hostStore.List(ctx, params)
		if err != nil {
			return nil, apitypes.ResourceMutationError("host", err)
		}
		return &hostListOutput{Body: apitypes.Page[Host]{Items: rows, Count: count}}, nil
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

func loadHostDetailBody[T any](
	ctx context.Context,
	hostStore *Store,
	hostID int64,
	build DetailBuilder[T],
	contributors []DetailContributor[T],
) (*T, error) {
	if build == nil {
		return nil, errors.New("host detail builder is not configured")
	}
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
	body := build(*detail)
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

func registerGetHost[T any](api huma.API, opts AdminRoutesOptions[T]) {
	huma.Register(api, huma.Operation{
		OperationID: "get-host",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}",
		Tags:        []string{hostsTag},
		Summary:     "Get an enrolled host",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*hostDetailOutput[T], error) {
		body, err := loadHostDetailBody(ctx, opts.Store, input.ID, opts.DetailBuilder, opts.Contributors)
		if err != nil {
			return nil, err
		}
		return &hostDetailOutput[T]{Body: *body}, nil
	})
}

func registerPutHostUserAffinity[T any](api huma.API, opts AdminRoutesOptions[T]) {
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
	}, func(ctx context.Context, input *hostUserAffinityPutInput) (*hostDetailOutput[T], error) {
		if err := requireAdmin(ctx, opts.RequireAdmin); err != nil {
			return nil, err
		}
		if _, err := opts.Store.GetByID(ctx, input.ID); errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		} else if err != nil {
			return nil, err
		}
		email := strings.TrimSpace(input.Body.Email)
		if email == "" {
			return nil, huma.Error400BadRequest("email is required")
		}
		if err := opts.UserAffinities.Upsert(ctx, input.ID, email, UserAffinitySourceManual); err != nil {
			return nil, err
		}
		body, err := loadHostDetailBody(ctx, opts.Store, input.ID, opts.DetailBuilder, opts.Contributors)
		if err != nil {
			return nil, err
		}
		return &hostDetailOutput[T]{Body: *body}, nil
	})
}

func registerDeleteHostUserAffinity[T any](api huma.API, opts AdminRoutesOptions[T]) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-host-user-affinity",
		Method:      http.MethodDelete,
		Path:        "/api/hosts/{id}/user-affinity",
		Tags:        []string{hostsTag},
		Summary:     "Clear the host user affinity",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*hostDetailOutput[T], error) {
		if err := requireAdmin(ctx, opts.RequireAdmin); err != nil {
			return nil, err
		}
		if _, err := opts.Store.GetByID(ctx, input.ID); errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		} else if err != nil {
			return nil, err
		}
		if err := opts.UserAffinities.Delete(ctx, input.ID, UserAffinitySourceManual); err != nil {
			return nil, err
		}
		body, err := loadHostDetailBody(ctx, opts.Store, input.ID, opts.DetailBuilder, opts.Contributors)
		if err != nil {
			return nil, err
		}
		return &hostDetailOutput[T]{Body: *body}, nil
	})
}

func registerDeleteHost(api huma.API, hostStore *Store, requireAdminFunc func(context.Context) error) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-host",
		Method:      http.MethodDelete,
		Path:        "/api/hosts/{id}",
		Tags:        []string{hostsTag},
		Summary:     "Delete an enrolled host",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*struct{}, error) {
		if err := requireAdmin(ctx, requireAdminFunc); err != nil {
			return nil, err
		}
		if err := hostStore.Delete(ctx, input.ID); err != nil {
			return nil, apitypes.ResourceMutationError("host", err)
		}
		return &struct{}{}, nil
	})
}

func registerBulkDeleteHosts(api huma.API, hostStore *Store, requireAdminFunc func(context.Context) error) {
	huma.Register(api, huma.Operation{
		OperationID: "bulk-delete-hosts",
		Method:      http.MethodPost,
		Path:        "/api/hosts/bulk-delete",
		Tags:        []string{hostsTag},
		Summary:     "Delete enrolled hosts",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *hostBulkDeleteInput) (*struct{}, error) {
		if err := requireAdmin(ctx, requireAdminFunc); err != nil {
			return nil, err
		}
		if _, err := hostStore.DeleteMany(ctx, input.Body.IDs); err != nil {
			return nil, err
		}
		return &struct{}{}, nil
	})
}

func requireAdmin(ctx context.Context, requireAdminFunc func(context.Context) error) error {
	if requireAdminFunc == nil {
		return errors.New("admin authorizer is not configured")
	}
	return requireAdminFunc(ctx)
}
