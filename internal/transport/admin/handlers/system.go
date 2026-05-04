package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/database"
)

type healthOutput struct {
	Body struct {
		Status string `json:"status"`
	}
}

type versionOutput struct {
	Body struct {
		Version   string `json:"version"`
		StartedAt string `json:"started_at"`
	}
}

const systemTag = "System"

// RegisterSystem registers health, readiness, and version endpoints.
func RegisterSystem(api huma.API, db *database.DB, version string, started time.Time) {
	huma.Register(api, huma.Operation{
		OperationID: "health",
		Method:      http.MethodGet,
		Path:        "/api/healthz",
		Tags:        []string{systemTag},
		Summary:     "Liveness check",
	}, func(_ context.Context, _ *struct{}) (*healthOutput, error) {
		out := &healthOutput{}
		out.Body.Status = "alive"
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "ready",
		Method:      http.MethodGet,
		Path:        "/api/readyz",
		Tags:        []string{systemTag},
		Summary:     "Readiness check",
		Errors:      []int{http.StatusServiceUnavailable},
	}, func(ctx context.Context, _ *struct{}) (*healthOutput, error) {
		if err := db.Ping(ctx); err != nil {
			return nil, huma.Error503ServiceUnavailable("not ready")
		}
		out := &healthOutput{}
		out.Body.Status = "ready"
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "version",
		Method:      http.MethodGet,
		Path:        "/api/version",
		Tags:        []string{systemTag},
		Summary:     "Build version",
	}, func(_ context.Context, _ *struct{}) (*versionOutput, error) {
		out := &versionOutput{}
		out.Body.Version = version
		out.Body.StartedAt = started.Format(time.RFC3339)
		return out, nil
	})
}
