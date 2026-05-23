package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/santa"
)

type santaSyncTokenDeleteInput struct {
	ID string `path:"id"`
}

type santaSyncTokenListOutput struct {
	Body []santa.SyncToken
}

type santaSyncTokenCreateOutput struct {
	Body santa.CreatedSyncToken
}

func RegisterSantaSyncTokens(api huma.API, store *santa.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-santa-sync-tokens",
		Method:      http.MethodGet,
		Path:        "/api/santa/sync-tokens",
		Tags:        []string{"Santa"},
		Summary:     "List Santa sync tokens",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, _ *struct{}) (*santaSyncTokenListOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		tokens, err := store.ListSyncTokens(ctx)
		if err != nil {
			return nil, err
		}
		return &santaSyncTokenListOutput{Body: tokens}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "create-santa-sync-token",
		Method:        http.MethodPost,
		Path:          "/api/santa/sync-tokens",
		Tags:          []string{"Santa"},
		Summary:       "Create Santa sync token",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, _ *struct{}) (*santaSyncTokenCreateOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		token, err := store.CreateSyncToken(ctx)
		if err != nil {
			return nil, err
		}
		return &santaSyncTokenCreateOutput{Body: token}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "delete-santa-sync-token",
		Method:      http.MethodDelete,
		Path:        "/api/santa/sync-tokens/{id}",
		Tags:        []string{"Santa"},
		Summary:     "Delete Santa sync token",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *santaSyncTokenDeleteInput) (*struct{}, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		id, err := parseResourceID(input.ID, "sync token")
		if err != nil {
			return nil, err
		}
		err = store.DeleteSyncToken(ctx, id)
		if errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("sync token not found")
		}
		if err != nil {
			return nil, err
		}
		return &struct{}{}, nil
	})
}
