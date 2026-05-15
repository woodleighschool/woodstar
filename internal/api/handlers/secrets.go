package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/secrets"
)

type secretDeleteInput struct {
	ID string `path:"id"`
}

type secretListOutput struct {
	Body []secrets.Secret
}

type secretCreateOutput struct {
	Body secrets.Secret
}

// RegisterSecrets registers shared credential endpoints for Orbit enroll secrets.
// Santa and Munki return when their modules ship.
func RegisterSecrets(api huma.API, secretStore *secrets.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-orbit-enroll-secrets",
		Method:      http.MethodGet,
		Path:        "/api/orbit/enroll-secrets",
		Tags:        []string{"Orbit"},
		Summary:     "List Orbit enroll secrets",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, _ *struct{}) (*secretListOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		secrets, err := secretStore.ListOrbitEnrollSecrets(ctx)
		if err != nil {
			return nil, err
		}
		return &secretListOutput{Body: secrets}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "create-orbit-enroll-secret",
		Method:        http.MethodPost,
		Path:          "/api/orbit/enroll-secrets",
		Tags:          []string{"Orbit"},
		Summary:       "Create Orbit enroll secret",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, _ *struct{}) (*secretCreateOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		secret, err := secretStore.CreateOrbitEnrollSecret(ctx)
		if err != nil {
			return nil, err
		}
		return &secretCreateOutput{Body: secret}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "delete-orbit-enroll-secret",
		Method:      http.MethodDelete,
		Path:        "/api/orbit/enroll-secrets/{id}",
		Tags:        []string{"Orbit"},
		Summary:     "Delete Orbit enroll secret",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *secretDeleteInput) (*struct{}, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		id, err := parseResourceID(input.ID, "secret")
		if err != nil {
			return nil, err
		}
		err = secretStore.DeleteOrbitEnrollSecret(ctx, id)
		if errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("secret not found")
		}
		if err != nil {
			return nil, err
		}
		return &struct{}{}, nil
	})
}
