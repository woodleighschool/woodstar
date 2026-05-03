package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func hashToken(secret string, value string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}
