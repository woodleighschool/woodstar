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

// RegisterSecrets registers shared credential endpoints for Orbit enroll secrets.
// Santa and Munki return when their modules ship.
func RegisterSecrets(api huma.API, store *models.SecretStore) {
	registerSecretRoutes(api, store, models.SecretOrbit, secretRoute{
		listOperationID:   "list-orbit-enroll-secrets",
		createOperationID: "create-orbit-enroll-secret",
		deleteOperationID: "delete-orbit-enroll-secret",
		path:              "/api/orbit/enroll-secrets",
		tag:               "Orbit",
		name:              "Orbit enroll secret",
	})
}

func registerSecretRoutes(api huma.API, store *models.SecretStore, kind models.SecretKind, route secretRoute) {
	huma.Register(api, huma.Operation{
		OperationID: route.listOperationID,
		Method:      http.MethodGet,
		Path:        route.path,
		Tags:        []string{route.tag},
		Summary:     "List " + route.name + "s",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, _ *struct{}) (*secretListOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
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
		Errors:        []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, _ *struct{}) (*secretCreateOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
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
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *secretDeleteInput) (*struct{}, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		id, err := parseResourceID(input.ID, "secret")
		if err != nil {
			return nil, err
		}
		err = store.Delete(ctx, kind, id)
		if errors.Is(err, models.ErrNotFound) {
			return nil, huma.Error404NotFound("secret not found")
		}
		return nil, err
	})
}
