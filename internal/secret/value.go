// Package secret creates opaque shared secrets for Woodstar credentials.
package secret

import (
	"crypto/rand"
	"encoding/base64"
)

// Generate returns a 32-byte random URL-safe secret string.
func Generate() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}
