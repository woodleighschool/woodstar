package mdp

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/apitypes"
	"github.com/woodleighschool/woodstar/internal/randtoken"
)

const (
	munkiTag                  = "Munki"
	distributionPointResource = "distribution point"
	distributionPointIDPath   = "/api/munki/distribution-points/{id}"

	// keyByteLen is the entropy of a per-DP key. The key is a bearer credential
	// and an HMAC signing key, so it is sized like other agent secrets.
	keyByteLen = 32
)

type distributionPointListInput struct {
	apitypes.ListQueryInput
}

type distributionPointGetInput struct {
	ID int64 `path:"id"`
}

type distributionPointCreateInput struct {
	Body DistributionPointMutation
}

type distributionPointUpdateInput struct {
	ID   int64 `path:"id"`
	Body DistributionPointMutation
}

type distributionPointDeleteInput struct {
	ID int64 `path:"id"`
}

type distributionPointRotateInput struct {
	ID int64 `path:"id"`
}

type distributionPointReorderInput struct {
	Body distributionPointReorderBody
}

type distributionPointReorderBody struct {
	OrderedIDs []int64 `json:"ordered_ids"`
}

type distributionPointListOutput struct {
	Body apitypes.Page[DistributionPoint]
}

type distributionPointDetailOutput struct {
	Body DistributionPointDetail
}

// revealedDistributionPoint carries the key once, on create. The admin model
// never serializes it otherwise.
type revealedDistributionPoint struct {
	DistributionPoint

	Key string `json:"key"`
}

type distributionPointCreateOutput struct {
	Body revealedDistributionPoint
}

type distributionPointKeyOutput struct {
	Body distributionPointKeyBody
}

type distributionPointKeyBody struct {
	Key string `json:"key"`
}

func (input distributionPointListInput) params() DistributionPointListParams {
	return DistributionPointListParams{ListParams: input.ListQueryInput.Params()}
}

// RegisterAdminRoutes mounts the distribution point admin API on api.
func RegisterAdminRoutes(api huma.API, store *Store) {
	registerListDistributionPoints(api, store)
	registerCreateDistributionPoint(api, store)
	registerGetDistributionPoint(api, store)
	registerUpdateDistributionPoint(api, store)
	registerDeleteDistributionPoint(api, store)
	registerReorderDistributionPoints(api, store)
	registerRotateDistributionPointKey(api, store)
}

func registerListDistributionPoints(api huma.API, store *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-munki-distribution-points",
		Method:      http.MethodGet,
		Path:        "/api/munki/distribution-points",
		Tags:        []string{munkiTag},
		Summary:     "List Munki distribution points",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *distributionPointListInput) (*distributionPointListOutput, error) {
		rows, count, err := store.List(ctx, input.params())
		if err != nil {
			return nil, distributionPointMutationError(err)
		}
		return &distributionPointListOutput{
			Body: apitypes.Page[DistributionPoint]{Items: rows, Count: int32(count)},
		}, nil
	})
}

func registerCreateDistributionPoint(api huma.API, store *Store) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-munki-distribution-point",
		Method:        http.MethodPost,
		Path:          "/api/munki/distribution-points",
		Tags:          []string{munkiTag},
		Summary:       "Create a Munki distribution point",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *distributionPointCreateInput) (*distributionPointCreateOutput, error) {
		key, err := randtoken.Generate(keyByteLen)
		if err != nil {
			return nil, err
		}
		point, err := store.Create(ctx, input.Body, key)
		if err != nil {
			return nil, distributionPointMutationError(err)
		}
		return &distributionPointCreateOutput{
			Body: revealedDistributionPoint{DistributionPoint: *point, Key: key},
		}, nil
	})
}

func registerGetDistributionPoint(api huma.API, store *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-munki-distribution-point",
		Method:      http.MethodGet,
		Path:        distributionPointIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Get a Munki distribution point",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *distributionPointGetInput) (*distributionPointDetailOutput, error) {
		detail, err := store.GetByID(ctx, input.ID)
		if err != nil {
			return nil, distributionPointMutationError(err)
		}
		return &distributionPointDetailOutput{Body: *detail}, nil
	})
}

func registerUpdateDistributionPoint(api huma.API, store *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "update-munki-distribution-point",
		Method:      http.MethodPut,
		Path:        distributionPointIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Update a Munki distribution point",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *distributionPointUpdateInput) (*distributionPointDetailOutput, error) {
		if _, err := store.Update(ctx, input.ID, input.Body); err != nil {
			return nil, distributionPointMutationError(err)
		}
		detail, err := store.GetByID(ctx, input.ID)
		if err != nil {
			return nil, distributionPointMutationError(err)
		}
		return &distributionPointDetailOutput{Body: *detail}, nil
	})
}

func registerDeleteDistributionPoint(api huma.API, store *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-munki-distribution-point",
		Method:      http.MethodDelete,
		Path:        distributionPointIDPath,
		Tags:        []string{munkiTag},
		Summary:     "Delete a Munki distribution point",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *distributionPointDeleteInput) (*struct{}, error) {
		if err := store.Delete(ctx, input.ID); err != nil {
			return nil, distributionPointMutationError(err)
		}
		return &struct{}{}, nil
	})
}

func registerReorderDistributionPoints(api huma.API, store *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "reorder-munki-distribution-points",
		Method:      http.MethodPut,
		Path:        "/api/munki/distribution-points/order",
		Tags:        []string{munkiTag},
		Summary:     "Reorder Munki distribution points",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *distributionPointReorderInput) (*struct{}, error) {
		if err := store.Reorder(ctx, input.Body.OrderedIDs); err != nil {
			return nil, distributionPointMutationError(err)
		}
		return &struct{}{}, nil
	})
}

func registerRotateDistributionPointKey(api huma.API, store *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "rotate-munki-distribution-point-key",
		Method:      http.MethodPost,
		Path:        "/api/munki/distribution-points/{id}/key",
		Tags:        []string{munkiTag},
		Summary:     "Rotate a Munki distribution point key",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *distributionPointRotateInput) (*distributionPointKeyOutput, error) {
		key, err := randtoken.Generate(keyByteLen)
		if err != nil {
			return nil, err
		}
		if err := store.RotateKey(ctx, input.ID, key); err != nil {
			return nil, distributionPointMutationError(err)
		}
		return &distributionPointKeyOutput{Body: distributionPointKeyBody{Key: key}}, nil
	})
}

func distributionPointMutationError(err error) error {
	return apitypes.ResourceMutationError(distributionPointResource, err)
}
