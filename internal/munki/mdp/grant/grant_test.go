package grant

import (
	"errors"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/storage/capability"
)

func TestSignVerifyCarriesGrantClaims(t *testing.T) {
	t.Parallel()
	now := time.Unix(1_700_000_000, 0)
	key := []byte("distribution-point-key")
	claims := Claims{
		Exp:                 now.Add(time.Minute).Unix(),
		PackageID:           12,
		SHA256:              "abc123",
		SizeBytes:           4096,
		HostID:              7,
		Serial:              "C02ABC",
		DistributionPointID: 3,
	}

	token, err := Sign(key, claims)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	got, err := Verify(key, token, now)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got.Op != capability.OpGet {
		t.Fatalf("op = %q, want %q", got.Op, capability.OpGet)
	}
	claims.Op = capability.OpGet
	if got != claims {
		t.Fatalf("claims = %+v, want %+v", got, claims)
	}
}

func TestVerifyRejectsWrongKeyExpiryAndTampering(t *testing.T) {
	t.Parallel()
	now := time.Unix(1_700_000_000, 0)
	key := []byte("distribution-point-key")
	valid, err := Sign(key, Claims{Exp: now.Add(time.Minute).Unix(), PackageID: 12})
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	if _, err := Verify([]byte("other-key"), valid, now); !errors.Is(err, capability.ErrInvalid) {
		t.Fatalf("wrong key error = %v, want ErrInvalid", err)
	}

	expired, err := Sign(key, Claims{Exp: now.Add(-time.Second).Unix(), PackageID: 12})
	if err != nil {
		t.Fatalf("Sign expired: %v", err)
	}
	if _, err := Verify(key, expired, now); !errors.Is(err, capability.ErrExpired) {
		t.Fatalf("expired error = %v, want ErrExpired", err)
	}

	if _, err := Verify(key, valid+"x", now); !errors.Is(err, capability.ErrInvalid) {
		t.Fatalf("tampered token error = %v, want ErrInvalid", err)
	}
}
