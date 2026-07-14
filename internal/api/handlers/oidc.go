package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/auth"
)

// registerOIDC mounts the OIDC browser redirect endpoints. These routes bypass
// Huma because their contract is HTTP redirects, not API response bodies.
func registerOIDC(r chi.Router, authService *auth.Service, logger *slog.Logger) {
	r.Get("/api/auth/sso/start", oidcStart(authService, logger))
	r.Get("/api/auth/sso/callback", oidcCallback(authService, logger))
}

func oidcStart(authService *auth.Service, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !authService.SSOEnabled() {
			http.Error(w, "sso not configured", http.StatusNotFound)
			return
		}
		authURL, err := authService.BeginSSO(r.Context())
		if err != nil {
			logger.ErrorContext(r.Context(), "oidc start failed",
				"operation", "oidc-start",
				"err", err,
			)
			http.Error(w, "sso sign-in failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, authURL, http.StatusFound)
	}
}

func oidcCallback(authService *auth.Service, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if providerErr := query.Get("error"); providerErr != "" {
			redirectWithOIDCError(w, r, providerErr)
			return
		}
		state := query.Get("state")
		code := query.Get("code")
		if state == "" || code == "" {
			redirectWithOIDCError(w, r, "missing state or code")
			return
		}
		if _, err := authService.CompleteSSO(r.Context(), state, code); err != nil {
			if !expectedOIDCError(err) {
				logger.ErrorContext(r.Context(), "oidc callback failed",
					"operation", "oidc-callback",
					"err", err,
				)
			}
			redirectWithOIDCError(w, r, oidcUserMessage(err))
			return
		}
		http.Redirect(w, r, "/hosts", http.StatusFound)
	}
}

func expectedOIDCError(err error) bool {
	return errors.Is(err, auth.ErrSSOStateMismatch) ||
		errors.Is(err, auth.ErrSSONonceMismatch) ||
		errors.Is(err, auth.ErrSSOUnknownUser) ||
		errors.Is(err, auth.ErrSSOEmailClaimEmpty)
}

// oidcUserMessage maps known sentinel errors to short user-facing strings and
// leaves everything else as a generic message so SQL/IDP detail does not leak
// through the redirect URL.
func oidcUserMessage(err error) string {
	switch {
	case errors.Is(err, auth.ErrSSOStateMismatch):
		return "sso state mismatch; try again"
	case errors.Is(err, auth.ErrSSONonceMismatch):
		return "sso nonce mismatch; try again"
	case errors.Is(err, auth.ErrSSOUnknownUser):
		return "no woodstar account for this identity"
	case errors.Is(err, auth.ErrSSOEmailClaimEmpty):
		return "identity provider returned no email"
	default:
		return "sso sign-in failed"
	}
}

func redirectWithOIDCError(w http.ResponseWriter, r *http.Request, message string) {
	target := "/login?sso_error=" + url.QueryEscape(message)
	http.Redirect(w, r, target, http.StatusFound)
}
