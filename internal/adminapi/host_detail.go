package adminapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/osquery/checks"
	"github.com/woodleighschool/woodstar/internal/santa"
)

const hostsTag = "Hosts"

// HostDetail is the admin host detail response with optional capability fields.
type HostDetail struct {
	hosts.HostDetail

	Munki *munki.HostState `json:"munki,omitempty"`
	Santa *santa.HostState `json:"santa,omitempty"`
}

type HostRoutesOptions struct {
	Store        *hosts.Store
	PrimaryUsers *hosts.PrimaryUserStore
	CheckStore   *checks.Store
	MunkiState   munkiHostStateLoader
	SantaState   santaHostStateLoader
}

type hostDetailOutput struct {
	Body HostDetail
}

type hostGetInput struct {
	ID int64 `path:"id"`
}

type hostPrimaryUserPutBody struct {
	Email string `json:"email" format:"email" minLength:"3"`
}

type hostPrimaryUserPutInput struct {
	ID   int64 `path:"id"`
	Body hostPrimaryUserPutBody
}

// RegisterHostAdminRoutes registers the host routes whose response is composed
// across capabilities.
func RegisterHostAdminRoutes(api huma.API, opts HostRoutesOptions) {
	hosts.RegisterAdminRoutes(api, hosts.AdminRoutesOptions{
		Store:       opts.Store,
		CheckFilter: hostCheckStatusFilter(opts.CheckStore),
	})
	registerGetHost(api, opts)
	registerSetHostPrimaryUser(api, opts)
	registerClearHostPrimaryUser(api, opts)
}

func hostCheckStatusFilter(checkStore *checks.Store) hosts.CheckStatusFilter {
	if checkStore == nil {
		return nil
	}
	return osqueryCheckFilter{store: checkStore}
}

type osqueryCheckFilter struct {
	store *checks.Store
}

func (f osqueryCheckFilter) HostIDsByStatus(ctx context.Context, checkID int64, status string) ([]int64, error) {
	return f.store.HostIDsByStatus(ctx, checkID, checks.CheckStatus(status))
}

type hostDetailContributor interface {
	ContributeHostDetail(ctx context.Context, hostID int64, body *HostDetail) error
}

func hostDetailContributors(opts HostRoutesOptions) []hostDetailContributor {
	contributors := make([]hostDetailContributor, 0, 2)
	if opts.MunkiState != nil {
		contributors = append(contributors, munkiHostDetailContributor{loader: opts.MunkiState})
	}
	if opts.SantaState != nil {
		contributors = append(contributors, santaHostDetailContributor{loader: opts.SantaState})
	}
	return contributors
}

func loadHostDetailBody(ctx context.Context, opts HostRoutesOptions, hostID int64) (*HostDetail, error) {
	host, err := opts.Store.GetByID(ctx, hostID)
	if errors.Is(err, dbutil.ErrNotFound) {
		return nil, huma.Error404NotFound("host not found")
	}
	if err != nil {
		return nil, err
	}
	detail, err := opts.Store.LoadDetail(ctx, host)
	if err != nil {
		return nil, err
	}
	body := HostDetail{HostDetail: *detail}
	for _, contributor := range hostDetailContributors(opts) {
		if err := contributor.ContributeHostDetail(ctx, hostID, &body); err != nil {
			return nil, err
		}
	}
	return &body, nil
}

func registerGetHost(api huma.API, opts HostRoutesOptions) {
	huma.Register(api, huma.Operation{
		OperationID: "get-host",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}",
		Tags:        []string{hostsTag},
		Summary:     "Get an enrolled host",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*hostDetailOutput, error) {
		body, err := loadHostDetailBody(ctx, opts, input.ID)
		if err != nil {
			return nil, err
		}
		return &hostDetailOutput{Body: *body}, nil
	})
}

func registerSetHostPrimaryUser(api huma.API, opts HostRoutesOptions) {
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
		if _, err := opts.Store.GetByID(ctx, input.ID); errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		} else if err != nil {
			return nil, err
		}
		email := strings.TrimSpace(input.Body.Email)
		if email == "" {
			return nil, huma.Error400BadRequest("email is required")
		}
		if err := opts.PrimaryUsers.Upsert(ctx, input.ID, email, hosts.PrimaryUserSourceManual); err != nil {
			return nil, err
		}
		body, err := loadHostDetailBody(ctx, opts, input.ID)
		if err != nil {
			return nil, err
		}
		return &hostDetailOutput{Body: *body}, nil
	})
}

func registerClearHostPrimaryUser(api huma.API, opts HostRoutesOptions) {
	huma.Register(api, huma.Operation{
		OperationID: "clear-host-primary-user",
		Method:      http.MethodDelete,
		Path:        "/api/hosts/{id}/primary-user",
		Tags:        []string{hostsTag},
		Summary:     "Clear the host primary user",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*hostDetailOutput, error) {
		if _, err := opts.Store.GetByID(ctx, input.ID); errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		} else if err != nil {
			return nil, err
		}
		if err := opts.PrimaryUsers.Delete(ctx, input.ID, hosts.PrimaryUserSourceManual); err != nil {
			return nil, err
		}
		body, err := loadHostDetailBody(ctx, opts, input.ID)
		if err != nil {
			return nil, err
		}
		return &hostDetailOutput{Body: *body}, nil
	})
}

type munkiHostDetailContributor struct {
	loader munkiHostStateLoader
}

type munkiHostStateLoader interface {
	LoadHostState(context.Context, int64) (*munki.HostState, error)
}

func (c munkiHostDetailContributor) ContributeHostDetail(
	ctx context.Context,
	hostID int64,
	body *HostDetail,
) error {
	detail, err := c.loader.LoadHostState(ctx, hostID)
	if err != nil {
		return err
	}
	body.Munki = detail
	return nil
}

type santaHostDetailContributor struct {
	loader santaHostStateLoader
}

type santaHostStateLoader interface {
	LoadHostState(context.Context, int64) (*santa.HostState, error)
}

func (c santaHostDetailContributor) ContributeHostDetail(
	ctx context.Context,
	hostID int64,
	body *HostDetail,
) error {
	detail, err := c.loader.LoadHostState(ctx, hostID)
	if err != nil {
		return err
	}
	body.Santa = detail
	return nil
}
