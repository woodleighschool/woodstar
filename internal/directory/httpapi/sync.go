package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/directory/entra"
)

type directorySyncOutput struct {
	Body entra.SyncStatus
}

func registerDirectorySync(humaAPI huma.API, jobs *entra.SyncJobs, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "get-directory-sync",
		Method:      http.MethodGet,
		Path:        "/api/directory/sync",
		Tags:        []string{api.TagDirectorySync},
		Summary:     "Get directory sync status",
	}, func(ctx context.Context, _ *struct{}) (*directorySyncOutput, error) {
		status, err := jobs.Status(ctx)
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "get-directory-sync", err)
		}
		return &directorySyncOutput{Body: status}, nil
	})

	huma.Register(humaAPI, huma.Operation{
		OperationID:   "trigger-directory-sync",
		Method:        http.MethodPost,
		Path:          "/api/directory/sync",
		Tags:          []string{api.TagDirectorySync},
		Summary:       "Trigger a directory sync",
		DefaultStatus: http.StatusAccepted,
		Errors:        []int{http.StatusConflict},
	}, func(ctx context.Context, _ *struct{}) (*directorySyncOutput, error) {
		status, err := jobs.Trigger(ctx)
		if errors.Is(err, entra.ErrSyncDisabled) {
			return nil, huma.Error409Conflict(err.Error())
		}
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "trigger-directory-sync", err)
		}
		return &directorySyncOutput{Body: status}, nil
	})
}
