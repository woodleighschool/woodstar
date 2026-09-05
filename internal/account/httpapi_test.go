package account

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

func TestAccountRejectsMissingPrincipal(t *testing.T) {
	router := http.NewServeMux()
	api := humago.New(router, huma.DefaultConfig("test", "test"))
	registerAccount(api, Dependencies{})
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodGet, "/api/account", ""},
		{http.MethodPut, "/api/account", `{"name":"Synthetic User"}`},
		{http.MethodPost, "/api/account/api-key", ""},
		{http.MethodDelete, "/api/account/api-key", ""},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
			}
		})
	}
}
