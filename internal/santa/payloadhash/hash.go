package payloadhash

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
)

// Hash returns the hex-encoded SHA-256 of v1-framed fields.
func Hash(fields ...string) string {
	var b strings.Builder
	b.WriteString("v1")
	for _, f := range fields {
		b.WriteByte(0)
		b.WriteString(strconv.Itoa(len(f)))
		b.WriteByte(':')
		b.WriteString(f)
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}
