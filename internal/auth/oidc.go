package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/randtoken"
	"github.com/woodleighschool/woodstar/internal/users"
)

const (
	ssoStateSessionKey = "sso_state"
	ssoNonceSessionKey = "sso_nonce"
)

// SSO errors describe expected callback failures.
var (
	ErrSSOStateMismatch   = errors.New("sso state mismatch")
	ErrSSONotConfigured   = errors.New("sso is not configured")
	ErrSSOUnknownUser     = errors.New("no woodstar account for this identity")
	ErrSSOInitialUser     = errors.New("the initial user must sign in with a password, not SSO")
	ErrSSOEmailClaimEmpty = errors.New("identity provider returned no email claim")
)

// OIDCConfig contains everything needed to talk to an OIDC issuer.
type OIDCConfig struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	EmailClaim   string
}

// oidcProvider holds the configured oauth2 client and ID-token verifier.
type oidcProvider struct {
	oauth2     *oauth2.Config
	verifier   *oidc.IDTokenVerifier
	emailClaim string
}

// configureOIDC discovers the OIDC issuer and returns a provider.
func configureOIDC(ctx context.Context, cfg OIDCConfig) (*oidcProvider, error) {
	if cfg.IssuerURL == "" {
		return nil, ErrSSONotConfigured
	}
	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("discover oidc issuer: %w", err)
	}
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "email", "profile"}
	}
	emailClaim := cfg.EmailClaim
	if emailClaim == "" {
		emailClaim = "email"
	}
	return &oidcProvider{
		oauth2: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
		},
		verifier:   provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}),
		emailClaim: emailClaim,
	}, nil
}

// SSOEnabled reports whether the service has a configured OIDC provider.
func (s *Service) SSOEnabled() bool {
	return s.oidc != nil
}

// BeginSSO generates a state and nonce, stores them on the session, and
// returns the provider authorization URL the caller should redirect to.
func (s *Service) BeginSSO(ctx context.Context) (string, error) {
	if s.oidc == nil {
		return "", ErrSSONotConfigured
	}
	state, err := randtoken.Generate(apiKeyByteLen)
	if err != nil {
		return "", err
	}
	nonce, err := randtoken.Generate(apiKeyByteLen)
	if err != nil {
		return "", err
	}
	s.sessions.Put(ctx, ssoStateSessionKey, state)
	s.sessions.Put(ctx, ssoNonceSessionKey, nonce)
	return s.oidc.oauth2.AuthCodeURL(state, oidc.Nonce(nonce)), nil
}

func (s *Service) CompleteSSO(ctx context.Context, state, code string) (*users.User, error) {
	if s.oidc == nil {
		return nil, ErrSSONotConfigured
	}
	expectedState := s.sessions.PopString(ctx, ssoStateSessionKey)
	expectedNonce := s.sessions.PopString(ctx, ssoNonceSessionKey)
	if expectedState == "" || state == "" || state != expectedState {
		return nil, ErrSSOStateMismatch
	}

	token, err := s.oidc.oauth2.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("oidc token exchange: %w", err)
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return nil, errors.New("oidc token response missing id_token")
	}
	idToken, err := s.oidc.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("verify id token: %w", err)
	}
	if expectedNonce != "" && idToken.Nonce != expectedNonce {
		return nil, errors.New("id token nonce mismatch")
	}

	claims := map[string]any{}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("decode id token claims: %w", err)
	}
	email, _ := claims[s.oidc.emailClaim].(string)
	if email == "" {
		return nil, ErrSSOEmailClaimEmpty
	}

	user, err := s.users.GetByEmail(ctx, email)
	if errors.Is(err, dbutil.ErrNotFound) {
		return nil, ErrSSOUnknownUser
	}
	if err != nil {
		return nil, fmt.Errorf("lookup sso user: %w", err)
	}
	if s.users.IsInitialUser(user) {
		return nil, ErrSSOInitialUser
	}

	if err := s.startSession(ctx, user.ID); err != nil {
		return nil, err
	}
	return user, nil
}
