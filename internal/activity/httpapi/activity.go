// Package httpapi exposes the activity timeline.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	authhuma "github.com/woodleighschool/goodies/auth/huma"

	"github.com/woodleighschool/woodstar/internal/activity"
	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/rbac"
)

type listInput struct {
	api.ListQueryInput

	Area        activity.Area      `query:"area,omitempty"`
	ActorKind   activity.ActorKind `query:"actor_kind,omitempty"`
	Action      activity.Action    `query:"action,omitempty"`
	Since       time.Time          `query:"since,omitempty"`
	Before      time.Time          `query:"before,omitempty"`
	SubjectType string             `query:"subject_type,omitempty"`
	SubjectID   int64              `query:"subject_id,omitempty" minimum:"1"`
}

type listOutput struct {
	Body api.Page[activity.ActivityEvent]
}

// RegisterAPI mounts the activity timeline endpoint.
func RegisterAPI(
	routes api.AppRoutes,
	store *activity.Store,
	authorizer authhuma.Authorizer,
	logger *slog.Logger,
) {
	registerAPI(routes, store, authorizer, logger)
}

// RegisterOpenAPI documents the activity timeline endpoint.
func RegisterOpenAPI(routes api.AppRoutes) {
	registerAPI(routes, nil, nil, nil)
}

func registerAPI(
	routes api.AppRoutes,
	store *activity.Store,
	authorizer authhuma.Authorizer,
	logger *slog.Logger,
) {
	humaAPI := authhuma.ResourceAPI(routes.Protected, authorizer, logger, rbac.ResourceActivity)
	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-activity",
		Method:      http.MethodGet,
		Path:        "/api/activity",
		Tags:        []string{api.TagActivity},
		Summary:     "List activity",
	}, func(ctx context.Context, input *listInput) (*listOutput, error) {
		items, count, err := store.List(ctx, activity.ListParams{
			ListParams:  input.Params(),
			Area:        input.Area,
			ActorKind:   input.ActorKind,
			Action:      input.Action,
			Since:       input.Since,
			Before:      input.Before,
			SubjectType: input.SubjectType,
			SubjectID:   input.SubjectID,
		})
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "list-activity", err)
		}
		return &listOutput{Body: api.Page[activity.ActivityEvent]{Items: items, Count: count}}, nil
	})
}
