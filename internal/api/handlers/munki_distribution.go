package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

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
	ListQueryInput
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
	Body Page[mdp.DistributionPoint]
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
	return mdp.DistributionPointListParams{ListParams: input.ListQueryInput.params()}
}

func registerMunkiDistributionPoints(
	api huma.API,
	store *mdp.Store,
	connections distributionPointConnections,
	logger *slog.Logger,
) {
	registerListDistributionPoints(api, store, logger)
	registerCreateDistributionPoint(api, store, logger)
	registerGetDistributionPoint(api, store, logger)
	registerUpdateDistributionPoint(api, store, connections, logger)
	registerDeleteDistributionPoint(api, store, connections, logger)
	registerReorderDistributionPoints(api, store, logger)
	registerRotateDistributionPointKey(api, store, connections, logger)
}

func registerListDistributionPoints(api huma.API, store *mdp.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "list-munki-distribution-points",
		Method:      http.MethodGet,
		Path:        "/api/munki/distribution-points",
		Tags:        []string{munkiDistributionPointsTag},
		Summary:     "List distribution points",
	}, func(ctx context.Context, input *distributionPointListInput) (*distributionPointListOutput, error) {
		rows, count, err := store.List(ctx, input.params())
		if err != nil {
			return nil, handlerError(
				ctx,
				logger,
				"list-munki-distribution-points",
				resourceMutationError(distributionPointResource, err),
			)
		}
		return &distributionPointListOutput{
			Body: Page[mdp.DistributionPoint]{Items: rows, Count: count},
		}, nil
	})
}

func registerCreateDistributionPoint(api huma.API, store *mdp.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-munki-distribution-point",
		Method:        http.MethodPost,
		Path:          "/api/munki/distribution-points",
		Tags:          []string{munkiDistributionPointsTag},
		Summary:       "Create a distribution point",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *distributionPointCreateInput) (*distributionPointCreateOutput, error) {
		key, err := randtoken.Generate(keyByteLen)
		if err != nil {
			return nil, handlerError(ctx, logger, "create-munki-distribution-point", err)
		}
		point, err := store.Create(ctx, input.Body, key)
		if err != nil {
			return nil, handlerError(
				ctx,
				logger,
				"create-munki-distribution-point",
				resourceMutationError(distributionPointResource, err),
			)
		}
		return &distributionPointCreateOutput{
			Body: MunkiRevealedDistributionPoint{DistributionPoint: *point, Key: key},
		}, nil
	})
}

func registerGetDistributionPoint(
	api huma.API,
	store *mdp.Store,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID: "get-munki-distribution-point",
		Method:      http.MethodGet,
		Path:        distributionPointIDPath,
		Tags:        []string{munkiDistributionPointsTag},
		Summary:     "Get a distribution point",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *distributionPointGetInput) (*distributionPointDetailOutput, error) {
		detail, err := store.GetByID(ctx, input.ID)
		if err != nil {
			return nil, handlerError(
				ctx,
				logger,
				"get-munki-distribution-point",
				resourceMutationError(distributionPointResource, err),
				"distribution_point_id", input.ID,
			)
		}
		return &distributionPointDetailOutput{Body: *detail}, nil
	})
}

func registerUpdateDistributionPoint(
	api huma.API,
	store *mdp.Store,
	connections distributionPointConnections,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID: "update-munki-distribution-point",
		Method:      http.MethodPut,
		Path:        distributionPointIDPath,
		Tags:        []string{munkiDistributionPointsTag},
		Summary:     "Update a distribution point",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *distributionPointUpdateInput) (*distributionPointDetailOutput, error) {
		point, err := store.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, handlerError(
				ctx,
				logger,
				"update-munki-distribution-point",
				resourceMutationError(distributionPointResource, err),
				"distribution_point_id", input.ID,
			)
		}
		if !point.Enabled {
			connections.Disconnect(input.ID)
		}
		detail, err := store.GetByID(ctx, input.ID)
		if err != nil {
			return nil, handlerError(
				ctx,
				logger,
				"update-munki-distribution-point",
				resourceMutationError(distributionPointResource, err),
				"distribution_point_id", input.ID,
			)
		}
		return &distributionPointDetailOutput{Body: *detail}, nil
	})
}

func registerDeleteDistributionPoint(
	api huma.API,
	store *mdp.Store,
	connections distributionPointConnections,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-munki-distribution-point",
		Method:      http.MethodDelete,
		Path:        distributionPointIDPath,
		Tags:        []string{munkiDistributionPointsTag},
		Summary:     "Delete a distribution point",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *distributionPointDeleteInput) (*struct{}, error) {
		if err := store.Delete(ctx, input.ID); err != nil {
			return nil, handlerError(
				ctx,
				logger,
				"delete-munki-distribution-point",
				resourceMutationError(distributionPointResource, err),
				"distribution_point_id", input.ID,
			)
		}
		connections.Disconnect(input.ID)
		return &struct{}{}, nil
	})
}

func registerReorderDistributionPoints(api huma.API, store *mdp.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "reorder-munki-distribution-points",
		Method:      http.MethodPut,
		Path:        "/api/munki/distribution-points/order",
		Tags:        []string{munkiDistributionPointsTag},
		Summary:     "Reorder distribution points",
		Errors:      []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *distributionPointReorderInput) (*struct{}, error) {
		if err := store.Reorder(ctx, input.Body.OrderedIDs); err != nil {
			return nil, handlerError(
				ctx,
				logger,
				"reorder-munki-distribution-points",
				resourceMutationError(distributionPointResource, err),
			)
		}
		return &struct{}{}, nil
	})
}

func registerRotateDistributionPointKey(
	api huma.API,
	store *mdp.Store,
	connections distributionPointConnections,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID: "rotate-munki-distribution-point-key",
		Method:      http.MethodPost,
		Path:        "/api/munki/distribution-points/{id}/key",
		Tags:        []string{munkiDistributionPointsTag},
		Summary:     "Rotate a distribution point key",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *distributionPointRotateInput) (*distributionPointKeyOutput, error) {
		key, err := randtoken.Generate(keyByteLen)
		if err != nil {
			return nil, handlerError(
				ctx,
				logger,
				"rotate-munki-distribution-point-key",
				err,
				"distribution_point_id", input.ID,
			)
		}
		if err := store.RotateKey(ctx, input.ID, key); err != nil {
			return nil, handlerError(
				ctx,
				logger,
				"rotate-munki-distribution-point-key",
				resourceMutationError(distributionPointResource, err),
				"distribution_point_id", input.ID,
			)
		}
		connections.Disconnect(input.ID)
		return &distributionPointKeyOutput{Body: MunkiDistributionPointKeyBody{Key: key}}, nil
	})
}
