package syncstate

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
)

// PayloadHash returns the hex-encoded SHA-256 of a v1-prefixed framing of
// fields. Each field is preceded by NUL + len + ":" so concatenations cannot
// collide across boundaries (e.g. "a" + "bc" vs "ab" + "c"). Used by rule
// targets and signing-chain hashing.
func PayloadHash(fields ...string) string {
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
