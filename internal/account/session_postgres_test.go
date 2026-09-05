//go:build postgres

package account

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/woodleighschool/goodies/auth/authn"
	"github.com/woodleighschool/goodies/auth/authz"

	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/rbac"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestAccountCredentialsAndAdmission(t *testing.T) { //nolint:funlen // Keep credential replacement and access revocation in one ordered HTTP lifecycle.
	database, ctx := testdb.Open(t)
	store := directory.NewStore(database, labels.NewStore(database))
	users := directory.NewUserService(store)

	user, err := users.Create(ctx, directory.UserCreate{
		Email: "account@example.invalid", Name: "Account User", Password: "configured-password",
		Role: directory.RoleAdmin,
	})
	if err != nil {
		t.Fatal(err)
	}
	sessions := testSessionManager()
	authorization := testAuthorizer(t, database)
	authentication, err := authn.New(ctx, directory.NewAuthnStore(store), sessions, authn.Config{Admit: authorization.HasAccess, Logger: discardLogger()})
	if err != nil {
		t.Fatal(err)
	}
	router := authTestRouter(Dependencies{
		Users: users, Authn: authentication, Authz: authorization, Logger: discardLogger(),
	}, sessions)
	login := authTestLogin(router, user.Email, "configured-password")
	if login.Code != http.StatusOK {
		t.Fatalf("login = %d %s", login.Code, login.Body.String())
	}
	cookies := login.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("login cookies = %d, want one session cookie", len(cookies))
	}
	request := func(method, path, body, bearer string, cookie bool, want int) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequestWithContext(ctx, method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		if cookie {
			req.AddCookie(cookies[0])
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Fatalf("%s %s = %d %s, want %d", method, path, rec.Code, rec.Body.String(), want)
		}
		return rec
	}

	accountResponse(t, user.ID, request(http.MethodGet, "/api/account", "", "", true, http.StatusOK))
	updated := accountResponse(t, user.ID, request(http.MethodPut, "/api/account", `{"name":"Renamed User"}`, "", true, http.StatusOK))
	if updated.User.Name != "Renamed User" {
		t.Fatalf("updated name = %q", updated.User.Name)
	}
	first := accountResponse(t, user.ID, request(http.MethodPost, "/api/account/api-key", "", "", true, http.StatusCreated))
	if first.APIKey == "" || first.APIKeyCreatedAt == nil {
		t.Fatal("rotation did not return the stored credential")
	}
	accountResponse(t, user.ID, request(http.MethodGet, "/api/account", "", first.APIKey, false, http.StatusOK))
	request(http.MethodGet, "/content", "", first.APIKey, false, http.StatusNoContent)
	second := accountResponse(t, user.ID, request(http.MethodPost, "/api/account/api-key", "", "", true, http.StatusCreated))
	if second.APIKey == "" || second.APIKey == first.APIKey {
		t.Fatal("rotation did not replace the credential")
	}
	request(http.MethodGet, "/api/account", "", first.APIKey, false, http.StatusUnauthorized)
	revoked := accountResponse(t, user.ID, request(http.MethodDelete, "/api/account/api-key", "", "", true, http.StatusOK))
	if revoked.APIKey != "" || revoked.APIKeyCreatedAt != nil {
		t.Fatal("revocation left a credential in the account response")
	}
	request(http.MethodGet, "/api/account", "", second.APIKey, false, http.StatusUnauthorized)
	active := accountResponse(t, user.ID, request(http.MethodPost, "/api/account/api-key", "", "", true, http.StatusCreated))
	if _, err := database.Exec(ctx, `DELETE FROM authz_user_roles WHERE user_id = $1`, user.ID); err != nil {
		t.Fatal(err)
	}
	for _, bearer := range []string{"", active.APIKey} {
		request(http.MethodGet, "/api/account", "", bearer, bearer == "", http.StatusUnauthorized)
		request(http.MethodGet, "/content", "", bearer, bearer == "", http.StatusUnauthorized)
	}
	session := request(http.MethodGet, "/api/session", "", "", true, http.StatusOK)
	var state struct {
		User *authn.Principal `json:"user"`
	}
	if err := json.Unmarshal(session.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	if state.User != nil {
		t.Fatal("session exposed a principal without application access")
	}
	denied := authTestLogin(router, user.Email, "configured-password")
	if denied.Code != http.StatusUnauthorized || len(denied.Result().Cookies()) != 0 {
		t.Fatalf("unadmitted login = %d, cookies = %d", denied.Code, len(denied.Result().Cookies()))
	}
	if _, err := database.Exec(ctx, `ALTER TABLE authz_role_permissions RENAME TO unavailable_permissions`); err != nil {
		t.Fatal(err)
	}
	request(http.MethodGet, "/api/account", "", "", true, http.StatusInternalServerError)
	request(http.MethodGet, "/content", "", active.APIKey, false, http.StatusInternalServerError)
	unavailable := authTestLogin(router, user.Email, "configured-password")
	if unavailable.Code != http.StatusInternalServerError || len(unavailable.Result().Cookies()) != 0 {
		t.Fatalf("unavailable grants login = %d, cookies = %d", unavailable.Code, len(unavailable.Result().Cookies()))
	}
	request(http.MethodDelete, "/api/session", "", "", true, http.StatusNoContent)
	if _, err := database.Exec(ctx, `ALTER TABLE unavailable_permissions RENAME TO authz_role_permissions`); err != nil {
		t.Fatal(err)
	}
	if _, err := users.SetRoleByEmail(ctx, user.Email, directory.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	request(http.MethodGet, "/api/account", "", "", true, http.StatusUnauthorized)
	accountResponse(t, user.ID, request(http.MethodGet, "/api/account", "", active.APIKey, false, http.StatusOK))
}

func testAuthorizer(t *testing.T, database *pgxpool.Pool) *authz.Service {
	t.Helper()
	service, err := authz.NewService(rbac.NewStore(database), rbac.Resources())
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func accountResponse(t *testing.T, principalID int64, rec *httptest.ResponseRecorder) Account {
	t.Helper()
	var account Account
	if err := json.Unmarshal(rec.Body.Bytes(), &account); err != nil {
		t.Fatal(err)
	}
	if account.User.ID != principalID || account.EffectivePermissions[rbac.ResourceUsers] != authz.Edit {
		t.Fatalf("account identity or permissions missing: %+v", account)
	}
	return account
}
