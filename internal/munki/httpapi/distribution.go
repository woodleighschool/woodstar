package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/munki/mdp"
	"github.com/woodleighschool/woodstar/internal/randtoken"
)

const (
	distributionPointResource = "distribution point"
	distributionPointIDPath   = "/api/munki/distribution-points/{id}"

	// keyByteLen is the entropy of a per-DP key. The key is a bearer credential
	// and an HMAC signing key, so it is sized like other agent secrets.
	keyByteLen = 32
)

type distributionPointListInput struct {
	api.ListQueryInput
}

type distributionPointGetInput struct {
	ID int64 `path:"id"`
}

type distributionPointCreateInput struct {
	Body mdp.DistributionPointMutation
}

type distributionPointUpdateInput struct {
	ID   int64 `path:"id"`
	Body mdp.DistributionPointMutation
}

type distributionPointDeleteInput struct {
	ID int64 `path:"id"`
}

type distributionPointRotateInput struct {
	ID int64 `path:"id"`
}

type distributionPointReorderInput struct {
	Body MunkiDistributionPointReorderBody
}

type MunkiDistributionPointReorderBody struct {
	OrderedIDs []int64 `json:"ordered_ids"`
}

type distributionPointListOutput struct {
	Body api.Page[mdp.DistributionPoint]
}

type distributionPointDetailOutput struct {
	Body mdp.DistributionPointDetail
}

// MunkiRevealedDistributionPoint carries the key once, on create. The admin model
// never serializes it otherwise.
type MunkiRevealedDistributionPoint struct {
	mdp.DistributionPoint

	Key string `json:"key"`
}

type distributionPointCreateOutput struct {
	Body MunkiRevealedDistributionPoint
}

type distributionPointKeyOutput struct {
	Body MunkiDistributionPointKeyBody
}

type MunkiDistributionPointKeyBody struct {
	Key string `json:"key"`
}

func (input distributionPointListInput) params() mdp.DistributionPointListParams {
	return mdp.DistributionPointListParams{ListParams: input.Params()}
}

func registerMunkiDistributionPoints(
	humaAPI huma.API,
	store *mdp.Store,
	connections distributionPointConnections,
	logger *slog.Logger,
) {
	registerListDistributionPoints(humaAPI, store, logger)
	registerCreateDistributionPoint(humaAPI, store, logger)
	registerGetDistributionPoint(humaAPI, store, logger)
	registerUpdateDistributionPoint(humaAPI, store, connections, logger)
	registerDeleteDistributionPoint(humaAPI, store, connections, logger)
	registerReorderDistributionPoints(humaAPI, store, logger)
	registerRotateDistributionPointKey(humaAPI, store, connections, logger)
}

func registerListDistributionPoints(humaAPI huma.API, store *mdp.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-munki-distribution-points",
		Method:      http.MethodGet,
		Path:        "/api/munki/distribution-points",
		Tags:        []string{api.TagMunkiDistributionPoints},
		Summary:     "List distribution points",
	}, func(ctx context.Context, input *distributionPointListInput) (*distributionPointListOutput, error) {
		rows, count, err := store.List(ctx, input.params())
		if err != nil {
			return nil, api.HandlerError(
				ctx,
				logger,
				"list-munki-distribution-points",
				api.ResourceMutationError(distributionPointResource, err),
			)
		}
		return &distributionPointListOutput{
			Body: api.Page[mdp.DistributionPoint]{Items: rows, Count: count},
		}, nil
	})
}

func registerCreateDistributionPoint(humaAPI huma.API, store *mdp.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "create-munki-distribution-point",
		Method:        http.MethodPost,
		Path:          "/api/munki/distribution-points",
		Tags:          []string{api.TagMunkiDistributionPoints},
		Summary:       "Create a distribution point",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *distributionPointCreateInput) (*distributionPointCreateOutput, error) {
		key, err := randtoken.Generate(keyByteLen)
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "create-munki-distribution-point", err)
		}
		point, err := store.Create(ctx, input.Body, key)
		if err != nil {
			return nil, api.HandlerError(
				ctx,
				logger,
				"create-munki-distribution-point",
				api.ResourceMutationError(distributionPointResource, err),
			)
		}
		return &distributionPointCreateOutput{
			Body: MunkiRevealedDistributionPoint{DistributionPoint: *point, Key: key},
		}, nil
	})
}

func registerGetDistributionPoint(
	humaAPI huma.API,
	store *mdp.Store,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "get-munki-distribution-point",
		Method:      http.MethodGet,
		Path:        distributionPointIDPath,
		Tags:        []string{api.TagMunkiDistributionPoints},
		Summary:     "Get a distribution point",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *distributionPointGetInput) (*distributionPointDetailOutput, error) {
		detail, err := store.GetByID(ctx, input.ID)
		if err != nil {
			return nil, api.HandlerError(
				ctx,
				logger,
				"get-munki-distribution-point",
				api.ResourceMutationError(distributionPointResource, err),
				"distribution_point_id", input.ID,
			)
		}
		return &distributionPointDetailOutput{Body: *detail}, nil
	})
}

func registerUpdateDistributionPoint(
	humaAPI huma.API,
	store *mdp.Store,
	connections distributionPointConnections,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "update-munki-distribution-point",
		Method:      http.MethodPut,
		Path:        distributionPointIDPath,
		Tags:        []string{api.TagMunkiDistributionPoints},
		Summary:     "Update a distribution point",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *distributionPointUpdateInput) (*distributionPointDetailOutput, error) {
		point, err := store.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, api.HandlerError(
				ctx,
				logger,
				"update-munki-distribution-point",
				api.ResourceMutationError(distributionPointResource, err),
				"distribution_point_id", input.ID,
			)
		}
		if !point.Enabled {
			connections.Disconnect(input.ID)
		}
		detail, err := store.GetByID(ctx, input.ID)
		if err != nil {
			return nil, api.HandlerError(
				ctx,
				logger,
				"update-munki-distribution-point",
				api.ResourceMutationError(distributionPointResource, err),
				"distribution_point_id", input.ID,
			)
		}
		return &distributionPointDetailOutput{Body: *detail}, nil
	})
}

func registerDeleteDistributionPoint(
	humaAPI huma.API,
	store *mdp.Store,
	connections distributionPointConnections,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "delete-munki-distribution-point",
		Method:      http.MethodDelete,
		Path:        distributionPointIDPath,
		Tags:        []string{api.TagMunkiDistributionPoints},
		Summary:     "Delete a distribution point",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *distributionPointDeleteInput) (*struct{}, error) {
		if err := store.Delete(ctx, input.ID); err != nil {
			return nil, api.HandlerError(
				ctx,
				logger,
				"delete-munki-distribution-point",
				api.ResourceMutationError(distributionPointResource, err),
				"distribution_point_id", input.ID,
			)
		}
		connections.Disconnect(input.ID)
		return &struct{}{}, nil
	})
}

func registerReorderDistributionPoints(humaAPI huma.API, store *mdp.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "reorder-munki-distribution-points",
		Method:      http.MethodPut,
		Path:        "/api/munki/distribution-points/order",
		Tags:        []string{api.TagMunkiDistributionPoints},
		Summary:     "Reorder distribution points",
		Errors:      []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *distributionPointReorderInput) (*struct{}, error) {
		if err := store.Reorder(ctx, input.Body.OrderedIDs); err != nil {
			return nil, api.HandlerError(
				ctx,
				logger,
				"reorder-munki-distribution-points",
				api.ResourceMutationError(distributionPointResource, err),
			)
		}
		return &struct{}{}, nil
	})
}

func registerRotateDistributionPointKey(
	humaAPI huma.API,
	store *mdp.Store,
	connections distributionPointConnections,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "rotate-munki-distribution-point-key",
		Method:      http.MethodPost,
		Path:        "/api/munki/distribution-points/{id}/key",
		Tags:        []string{api.TagMunkiDistributionPoints},
		Summary:     "Rotate a distribution point key",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *distributionPointRotateInput) (*distributionPointKeyOutput, error) {
		key, err := randtoken.Generate(keyByteLen)
		if err != nil {
			return nil, api.HandlerError(
				ctx,
				logger,
				"rotate-munki-distribution-point-key",
				err,
				"distribution_point_id", input.ID,
			)
		}
		if err := store.RotateKey(ctx, input.ID, key); err != nil {
			return nil, api.HandlerError(
				ctx,
				logger,
				"rotate-munki-distribution-point-key",
				api.ResourceMutationError(distributionPointResource, err),
				"distribution_point_id", input.ID,
			)
		}
		connections.Disconnect(input.ID)
		return &distributionPointKeyOutput{Body: MunkiDistributionPointKeyBody{Key: key}}, nil
	})
}
