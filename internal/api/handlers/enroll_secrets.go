package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/enrollment"
)

type secretDeleteInput struct {
	ID string `path:"id"`
}

type secretListOutput struct {
	Body []enrollment.EnrollSecret
}

type secretCreateOutput struct {
	Body enrollment.EnrollSecret
}

func RegisterEnrollSecrets(api huma.API, secretStore *enrollment.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-enroll-secrets",
		Method:      http.MethodGet,
		Path:        "/api/enroll-secrets",
		Tags:        []string{"Enrollment"},
		Summary:     "List enroll secrets",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, _ *struct{}) (*secretListOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		secrets, err := secretStore.List(ctx)
		if err != nil {
			return nil, err
		}
		return &secretListOutput{Body: secrets}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "create-enroll-secret",
		Method:        http.MethodPost,
		Path:          "/api/enroll-secrets",
		Tags:          []string{"Enrollment"},
		Summary:       "Create enroll secret",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, _ *struct{}) (*secretCreateOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		secret, err := secretStore.Create(ctx)
		if err != nil {
			return nil, err
		}
		return &secretCreateOutput{Body: secret}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "delete-enroll-secret",
		Method:      http.MethodDelete,
		Path:        "/api/enroll-secrets/{id}",
		Tags:        []string{"Enrollment"},
		Summary:     "Delete enroll secret",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *secretDeleteInput) (*struct{}, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		id, err := parseResourceID(input.ID, "secret")
		if err != nil {
			return nil, err
		}
		err = secretStore.Delete(ctx, id)
		if errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("secret not found")
		}
		if err != nil {
			return nil, err
		}
		return &struct{}{}, nil
	})
}
