// Package capability signs and verifies short-lived storage transfer tokens.
package capability

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const (
	// OpGet allows a token to read one object.
	OpGet = "get"
	// OpPut allows a token to write one object.
	OpPut = "put"
)

var (
	// ErrInvalid reports a malformed token, a failed MAC, or invalid claims.
	ErrInvalid = errors.New("invalid capability")
	// ErrExpired reports a valid token whose expiry has passed.
	ErrExpired = errors.New("expired capability")
	// ErrWrongOp reports a valid token used for a different operation.
	ErrWrongOp = errors.New("wrong capability operation")
)

// Claims is the signed storage capability payload.
type Claims struct {
	Op          string `json:"op"`
	Key         string `json:"key"`
	Exp         int64  `json:"exp"`
	SHA256      string `json:"sha256,omitempty"`
	Size        *int64 `json:"size,omitempty"`
	ContentType string `json:"content_type,omitempty"`
}

// Sign returns a compact HMAC capability token for claims.
func Sign(key []byte, claims Claims) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmacSHA256(key, []byte(encodedPayload))
	encodedMAC := base64.RawURLEncoding.EncodeToString(mac)
	return encodedPayload + "." + encodedMAC, nil
}

// Verify checks token's MAC, expiry, operation, and required key claim.
func Verify(key []byte, token string, op string, now time.Time) (Claims, error) {
	payload, mac, ok := strings.Cut(token, ".")
	if !ok || payload == "" || mac == "" || strings.Contains(mac, ".") {
		return Claims{}, ErrInvalid
	}

	gotMAC, err := base64.RawURLEncoding.DecodeString(mac)
	if err != nil {
		return Claims{}, ErrInvalid
	}
	wantMAC := hmacSHA256(key, []byte(payload))
	if !hmac.Equal(gotMAC, wantMAC) {
		return Claims{}, ErrInvalid
	}

	rawClaims, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return Claims{}, ErrInvalid
	}
	var claims Claims
	if err := json.Unmarshal(rawClaims, &claims); err != nil {
		return Claims{}, ErrInvalid
	}
	if claims.Exp <= now.Unix() {
		return Claims{}, ErrExpired
	}
	if claims.Op != op {
		return Claims{}, ErrWrongOp
	}
	if claims.Key == "" {
		return Claims{}, ErrInvalid
	}
	return claims, nil
}

func hmacSHA256(key []byte, message []byte) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(message)
	return mac.Sum(nil)
}
