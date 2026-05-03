package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/models"
)

type secretDeleteInput struct {
	ID string `path:"id"`
}

type secretListOutput struct {
	Body []models.Secret
}

type secretCreateOutput struct {
	Body models.Secret
}

type secretRoute struct {
	listOperationID   string
	createOperationID string
	deleteOperationID string
	path              string
	tag               string
	name              string
}

// RegisterSecrets registers shared credential endpoints for Orbit, Santa, and Munki.
func RegisterSecrets(api huma.API, store *models.SecretStore) {
	registerSecretRoutes(api, store, models.SecretOrbit, secretRoute{
		listOperationID:   "list-orbit-enroll-secrets",
		createOperationID: "create-orbit-enroll-secret",
		deleteOperationID: "delete-orbit-enroll-secret",
		path:              "/api/orbit/enroll-secrets",
		tag:               "Orbit",
		name:              "Orbit enroll secret",
	})

	registerSecretRoutes(api, store, models.SecretSanta, secretRoute{
		listOperationID:   "list-santa-tokens",
		createOperationID: "create-santa-token",
		deleteOperationID: "delete-santa-token",
		path:              "/api/santa/tokens",
		tag:               "Santa",
		name:              "Santa token",
	})

	registerSecretRoutes(api, store, models.SecretMunki, secretRoute{
		listOperationID:   "list-munki-tokens",
		createOperationID: "create-munki-token",
		deleteOperationID: "delete-munki-token",
		path:              "/api/munki/tokens",
		tag:               "Munki",
		name:              "Munki token",
	})
}

func registerSecretRoutes(api huma.API, store *models.SecretStore, kind models.SecretKind, route secretRoute) {
	huma.Register(api, huma.Operation{
		OperationID: route.listOperationID,
		Method:      http.MethodGet,
		Path:        route.path,
		Tags:        []string{route.tag},
		Summary:     "List " + route.name + "s",
	}, func(ctx context.Context, _ *struct{}) (*secretListOutput, error) {
		secrets, err := store.List(ctx, kind)
		if err != nil {
			return nil, err
		}
		return &secretListOutput{Body: secrets}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   route.createOperationID,
		Method:        http.MethodPost,
		Path:          route.path,
		Tags:          []string{route.tag},
		Summary:       "Create " + route.name,
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, _ *struct{}) (*secretCreateOutput, error) {
		secret, err := store.Create(ctx, kind)
		if err != nil {
			return nil, err
		}
		return &secretCreateOutput{Body: *secret}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: route.deleteOperationID,
		Method:      http.MethodDelete,
		Path:        route.path + "/{id}",
		Tags:        []string{route.tag},
		Summary:     "Delete " + route.name,
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *secretDeleteInput) (*struct{}, error) {
		err := store.Delete(ctx, kind, input.ID)
		if errors.Is(err, models.ErrNotFound) {
			return nil, huma.Error404NotFound("secret not found")
		}
		return nil, err
	})
}
