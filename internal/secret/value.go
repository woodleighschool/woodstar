// Package secret creates opaque shared secrets for Woodstar credentials.
package secret

import (
	"crypto/rand"
	"encoding/base64"
)

// Generate returns a URL-safe base64 string backed by byteCount random bytes.
func Generate(byteCount int) (string, error) {
	b := make([]byte, byteCount)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
