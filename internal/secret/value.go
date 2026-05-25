// Package secret makes random shared secrets.
package secret

import (
	"crypto/rand"
	"encoding/base64"
)

// Generate makes a URL-safe random string.
func Generate(byteCount int) (string, error) {
	b := make([]byte, byteCount)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
