package capability

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSignVerify(t *testing.T) {
	t.Parallel()
	now := time.Unix(1_700_000_000, 0)
	size := int64(42)
	claims := Claims{
		Op:          OpGet,
		Key:         "munki/packages/1/Installer.pkg",
		Exp:         now.Add(time.Minute).Unix(),
		SHA256:      strings.Repeat("a", 64),
		Size:        &size,
		ContentType: "application/octet-stream",
	}

	token, err := Sign([]byte("secret"), claims)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	got, err := Verify([]byte("secret"), token, OpGet, now)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got.Key != claims.Key ||
		got.Op != claims.Op ||
		got.Exp != claims.Exp ||
		got.SHA256 != claims.SHA256 ||
		got.ContentType != claims.ContentType ||
		got.Size == nil ||
		*got.Size != size {
		t.Fatalf("claims = %+v, want %+v", got, claims)
	}
}

func TestVerifyRejectsInvalidTokens(t *testing.T) {
	t.Parallel()
	now := time.Unix(1_700_000_000, 0)
	key := []byte("secret")
	valid, err := Sign(key, Claims{
		Op:  OpGet,
		Key: "munki/icons/1/icon.png",
		Exp: now.Add(time.Minute).Unix(),
	})
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	cases := []struct {
		name  string
		token string
		key   []byte
		want  error
	}{
		{
			name:  "malformed",
			token: "not-a-token",
			key:   key,
			want:  ErrInvalid,
		},
		{
			name: "tampered claims",
			token: tamperClaims(t, valid, func(claims Claims) Claims {
				claims.Key = "munki/icons/2/icon.png"
				return claims
			}),
			key:  key,
			want: ErrInvalid,
		},
		{
			name:  "tampered mac",
			token: tamperLastByte(valid),
			key:   key,
			want:  ErrInvalid,
		},
		{
			name:  "wrong key",
			token: valid,
			key:   []byte("other-secret"),
			want:  ErrInvalid,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Verify(tc.key, tc.token, OpGet, now); !errors.Is(err, tc.want) {
				t.Fatalf("Verify error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestVerifyRejectsExpiredAndWrongOp(t *testing.T) {
	t.Parallel()
	now := time.Unix(1_700_000_000, 0)
	key := []byte("secret")

	expired, err := Sign(key, Claims{
		Op:  OpGet,
		Key: "munki/packages/1/Installer.pkg",
		Exp: now.Add(-time.Second).Unix(),
	})
	if err != nil {
		t.Fatalf("Sign expired: %v", err)
	}
	if _, err := Verify(key, expired, OpGet, now); !errors.Is(err, ErrExpired) {
		t.Fatalf("expired Verify error = %v, want ErrExpired", err)
	}

	put, err := Sign(key, Claims{
		Op:  OpPut,
		Key: "munki/packages/1/Installer.pkg",
		Exp: now.Add(time.Minute).Unix(),
	})
	if err != nil {
		t.Fatalf("Sign put: %v", err)
	}
	if _, err := Verify(key, put, OpGet, now); !errors.Is(err, ErrWrongOp) {
		t.Fatalf("wrong op Verify error = %v, want ErrWrongOp", err)
	}
}

func tamperLastByte(value string) string {
	replacement := byte('x')
	if value[len(value)-1] == replacement {
		replacement = 'y'
	}
	return value[:len(value)-1] + string(replacement)
}

func tamperClaims(t *testing.T, token string, edit func(Claims) Claims) string {
	t.Helper()
	payload, mac, ok := strings.Cut(token, ".")
	if !ok {
		t.Fatalf("token has no mac: %q", token)
	}
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	var claims Claims
	if err := json.Unmarshal(raw, &claims); err != nil {
		t.Fatalf("unmarshal claims: %v", err)
	}
	raw, err = json.Marshal(edit(claims))
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw) + "." + mac
}
