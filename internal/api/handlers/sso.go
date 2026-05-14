package handlers

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/auth"
)

// RegisterSSO mounts the OIDC sign-in endpoints. They are public (no auth
// required) and live under /api/auth/sso. Mount under a router that has
// the scs session middleware attached so state and nonce survive the
// redirect to the IdP and back.
func RegisterSSO(r chi.Router, authService *auth.Service) {
	r.Get("/api/auth/sso/start", ssoStart(authService))
	r.Get("/api/auth/sso/callback", ssoCallback(authService))
}

func ssoStart(authService *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !authService.SSOEnabled() {
			http.Error(w, "sso not configured", http.StatusNotFound)
			return
		}
		authURL, err := authService.BeginSSO(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, authURL, http.StatusFound)
	}
}

func ssoCallback(authService *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if providerErr := query.Get("error"); providerErr != "" {
			redirectWithSSOError(w, r, providerErr)
			return
		}
		state := query.Get("state")
		code := query.Get("code")
		if state == "" || code == "" {
			redirectWithSSOError(w, r, "missing state or code")
			return
		}
		if _, err := authService.CompleteSSO(r.Context(), state, code); err != nil {
			redirectWithSSOError(w, r, ssoUserMessage(err))
			return
		}
		http.Redirect(w, r, "/hosts", http.StatusFound)
	}
}

// ssoUserMessage maps known sentinel errors to short user-facing strings and
// leaves everything else as a generic message so SQL/IDP detail does not
// leak through the redirect URL.
func ssoUserMessage(err error) string {
	switch {
	case errors.Is(err, auth.ErrSSOStateMismatch):
		return "sso state mismatch; try again"
	case errors.Is(err, auth.ErrSSOUnknownUser):
		return "no woodstar account for this identity"
	case errors.Is(err, auth.ErrSSOInitialUser):
		return "the initial user must sign in with a password"
	case errors.Is(err, auth.ErrSSOEmailClaimEmpty):
		return "identity provider returned no email"
	default:
		return "sso sign-in failed"
	}
}

func redirectWithSSOError(w http.ResponseWriter, r *http.Request, message string) {
	target := "/login?sso_error=" + url.QueryEscape(message)
	http.Redirect(w, r, target, http.StatusFound)
}
