package adminapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/adminctx"
	"github.com/woodleighschool/woodstar/internal/directory"
)

// Admin posture is decided from the user already in the request context, so
// these tests inject the user directly and assert how the group modifiers gate
// each method. No auth service, session, or database is involved: those resolve
// the user (and are tested in auth), while this pins what adminapi does with it.

type postureProbeInput struct{}

type postureProbeOutput struct {
	Body struct {
		OK bool `json:"ok"`
	}
}

func roleUser(role directory.Role) *directory.User {
	return &directory.User{ID: 1, Role: &role}
}

// postureHandler mounts an ordinary (admin-for-mutations) and a sensitive
// (admin-for-all) group carrying the real modifiers, with a middleware that
// stands in for RequireAuth by injecting user into the context.
func postureHandler(user *directory.User) http.Handler {
	router := chi.NewRouter()
	api := humachi.New(router, huma.DefaultConfig("test", "test"))

	inject := func(ctx huma.Context, next func(huma.Context)) {
		if user != nil {
			ctx = huma.WithContext(ctx, adminctx.WithUser(ctx.Context(), user))
		}
		next(ctx)
	}
	ok := func(context.Context, *postureProbeInput) (*postureProbeOutput, error) {
		return &postureProbeOutput{}, nil
	}

	ordinary := huma.NewGroup(api)
	ordinary.UseMiddleware(inject)
	ordinary.UseModifier(RequireAdminForMutations(api))
	huma.Register(ordinary, huma.Operation{
		OperationID: "posture-ordinary-read", Method: http.MethodGet, Path: "/probe",
	}, ok)
	huma.Register(ordinary, huma.Operation{
		OperationID:   "posture-ordinary-write",
		Method:        http.MethodPost,
		Path:          "/probe",
		DefaultStatus: http.StatusCreated,
	}, ok)

	sensitive := huma.NewGroup(api)
	sensitive.UseMiddleware(inject)
	sensitive.UseModifier(requireAdminForAll(api))
	huma.Register(sensitive, huma.Operation{
		OperationID: "posture-sensitive-read", Method: http.MethodGet, Path: "/sensitive",
	}, ok)

	return router
}

func TestAdminModifiersGateByRoleAndMethod(t *testing.T) {
	admin := roleUser(directory.RoleAdmin)
	viewer := roleUser(directory.RoleViewer)

	for _, tc := range []struct {
		name   string
		user   *directory.User
		method string
		path   string
		want   int
	}{
		// Ordinary group: only mutations require admin.
		{"admin mutates ordinary", admin, http.MethodPost, "/probe", http.StatusCreated},
		{"viewer reads ordinary", viewer, http.MethodGet, "/probe", http.StatusOK},
		{"viewer cannot mutate ordinary", viewer, http.MethodPost, "/probe", http.StatusForbidden},
		{"anonymous cannot mutate ordinary", nil, http.MethodPost, "/probe", http.StatusUnauthorized},
		// Sensitive group: every method requires admin.
		{"admin reads sensitive", admin, http.MethodGet, "/sensitive", http.StatusOK},
		{"viewer cannot read sensitive", viewer, http.MethodGet, "/sensitive", http.StatusForbidden},
		{"anonymous cannot read sensitive", nil, http.MethodGet, "/sensitive", http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(context.Background(), tc.method, tc.path, nil)
			postureHandler(tc.user).ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("%s %s = %d, want %d; body = %q", tc.method, tc.path, rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}
