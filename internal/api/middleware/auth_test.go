package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/api/ctxkeys"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/directory"
)

// Admin posture is decided from the user already in the request context, so
// these tests inject the user directly and assert how the group modifiers gate
// each method. No auth service, session, or database is involved: those resolve
// the user (and are tested in auth), while this pins what api does with it.

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
			ctx = huma.WithContext(ctx, ctxkeys.WithUser(ctx.Context(), user))
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
	sensitive.UseModifier(RequireAdminForAll(api))
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

type fakeAuthenticator struct {
	user *directory.User
	err  error
	got  string
}

func (f *fakeAuthenticator) Authenticate(_ context.Context, authHeader string) (*directory.User, error) {
	f.got = authHeader
	if f.err != nil {
		return nil, f.err
	}
	return f.user, nil
}

func TestRequireHTTPAuthAttachesUser(t *testing.T) {
	role := directory.RoleAdmin
	authenticator := &fakeAuthenticator{user: &directory.User{ID: 42, Role: &role}}
	handler := RequireHTTPAuth(authenticator)(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		user, ok := ctxkeys.User(req.Context())
		if !ok {
			t.Fatal("missing user in context")
		}
		if user.ID != 42 {
			t.Fatalf("user id = %d, want 42", user.ID)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if authenticator.got != "Bearer secret" {
		t.Fatalf("auth header = %q, want Bearer secret", authenticator.got)
	}
}

func TestRequireHTTPAuthRejectsMissingCredentials(t *testing.T) {
	authenticator := &fakeAuthenticator{err: auth.ErrNotAuthenticated}
	handler := RequireHTTPAuth(authenticator)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler should not run")
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/protected", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAPIKeyFromQueryPromotesBearerAuthorization(t *testing.T) {
	var got string
	handler := APIKeyFromQuery("apikey")(http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
		got = req.Header.Get("Authorization")
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/hook?apikey=query-key", nil))

	if got != "Bearer query-key" {
		t.Fatalf("authorization = %q, want Bearer query-key", got)
	}
}

func TestAPIKeyFromQueryDoesNotOverwriteAuthorization(t *testing.T) {
	var got string
	handler := APIKeyFromQuery("apikey")(http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
		got = req.Header.Get("Authorization")
	}))

	req := httptest.NewRequest(http.MethodGet, "/hook?apikey=query-key", nil)
	req.Header.Set("Authorization", "Bearer header-key")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if got != "Bearer header-key" {
		t.Fatalf("authorization = %q, want existing header", got)
	}
}

func TestOptionalHumaAuthAllowsAnonymousAndRejectsBrokenLookup(t *testing.T) {
	type output struct {
		Body struct {
			UserID int64 `json:"user_id"`
		}
	}

	register := func(authenticator *fakeAuthenticator) http.Handler {
		r := chi.NewRouter()
		humaAPI := humachi.New(r, huma.DefaultConfig("test", "test"))
		group := huma.NewGroup(humaAPI)
		group.UseMiddleware(OptionalHumaAuth(humaAPI, authenticator))
		huma.Register(group, huma.Operation{
			OperationID: "optional-auth", Method: http.MethodGet, Path: "/session",
		}, func(ctx context.Context, _ *struct{}) (*output, error) {
			out := &output{}
			if user, ok := ctxkeys.User(ctx); ok {
				out.Body.UserID = user.ID
			}
			return out, nil
		})
		return r
	}

	for _, tc := range []struct {
		name          string
		authenticator *fakeAuthenticator
		wantStatus    int
		wantUserID    int64
	}{
		{
			name:          "anonymous allowed",
			authenticator: &fakeAuthenticator{err: auth.ErrNotAuthenticated},
			wantStatus:    http.StatusOK,
		},
		{
			name:          "user attached",
			authenticator: &fakeAuthenticator{user: &directory.User{ID: 7}},
			wantStatus:    http.StatusOK,
			wantUserID:    7,
		},
		{
			name:          "broken auth lookup fails",
			authenticator: &fakeAuthenticator{err: errors.New("db down")},
			wantStatus:    http.StatusInternalServerError,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			register(tc.authenticator).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/session", nil))
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body = %q", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus == http.StatusOK {
				var body struct {
					UserID int64 `json:"user_id"`
				}
				if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
					t.Fatalf("decode body: %v", err)
				}
				if body.UserID != tc.wantUserID {
					t.Fatalf("user_id = %d, want %d", body.UserID, tc.wantUserID)
				}
			}
		})
	}
}
