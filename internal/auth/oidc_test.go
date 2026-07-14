package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"
)

func TestCompleteSSORejectsMissingSessionNonce(t *testing.T) {
	sessions := scs.New()
	sessions.Store = memstore.New()
	ctx, err := sessions.Load(context.Background(), "")
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	sessions.Put(ctx, ssoStateSessionKey, "expected-state")

	service := &Service{
		sessions: sessions,
		oidc:     &oidcProvider{},
	}
	if _, err := service.CompleteSSO(ctx, "expected-state", "code"); !errors.Is(err, ErrSSONonceMismatch) {
		t.Fatalf("CompleteSSO error = %v, want %v", err, ErrSSONonceMismatch)
	}
}
