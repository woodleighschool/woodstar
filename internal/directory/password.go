package directory

import (
	"errors"

	"github.com/alexedwards/argon2id"
)

// ErrWeakPassword is returned when a local password is too short.
var ErrWeakPassword = errors.New("password must be at least 12 characters")

// passwordParams pins Argon2id cost so changes are explicit. Library defaults
// are documented as dev-only.
var passwordParams = &argon2id.Params{
	Memory:      64 * 1024,
	Iterations:  3,
	Parallelism: 1,
	SaltLength:  16,
	KeyLength:   32,
}

// HashPassword returns an encoded Argon2id hash for password.
func HashPassword(password string) (string, error) {
	if len(password) < 12 {
		return "", ErrWeakPassword
	}
	return argon2id.CreateHash(password, passwordParams)
}

// VerifyPassword reports whether password matches an encoded Argon2id hash.
func VerifyPassword(password string, encoded string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, encoded)
}
