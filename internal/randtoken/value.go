// Package randtoken creates URL-safe random tokens.
package randtoken

import (
	"crypto/rand"
	"encoding/base64"
)

// Generate returns a URL-safe random string with byteCount bytes of entropy.
func Generate(byteCount int) (string, error) {
	b := make([]byte, byteCount)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
