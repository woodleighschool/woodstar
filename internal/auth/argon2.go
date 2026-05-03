package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonMemory      uint32 = 64 * 1024
	argonIterations  uint32 = 3
	argonParallelism uint8  = 1
	argonSaltLength         = 16
	argonKeyLength   uint32 = 32
)

// HashPassword returns an encoded Argon2id hash for password.
func HashPassword(password string) (string, error) {
	if len(password) < 12 {
		return "", ErrWeakPassword
	}

	salt := make([]byte, argonSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	key := argon2.IDKey([]byte(password), salt, argonIterations, argonMemory, argonParallelism, argonKeyLength)
	return fmt.Sprintf(
		"argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory,
		argonIterations,
		argonParallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// VerifyPassword reports whether password matches an encoded Argon2id hash.
func VerifyPassword(password string, encoded string) (bool, error) {
	params, salt, expected, err := parsePasswordHash(encoded)
	if err != nil {
		return false, err
	}

	actual := argon2.IDKey(
		[]byte(password),
		salt,
		params.iterations,
		params.memory,
		params.parallelism,
		uint32(len(expected)),
	)
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}

type passwordParams struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
}

func parsePasswordHash(encoded string) (passwordParams, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 5 || parts[0] != "argon2id" || parts[1] != "v=19" {
		return passwordParams{}, nil, nil, errors.New("unsupported password hash")
	}

	params, err := parseParams(parts[2])
	if err != nil {
		return passwordParams{}, nil, nil, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return passwordParams{}, nil, nil, fmt.Errorf("decode password salt: %w", err)
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return passwordParams{}, nil, nil, fmt.Errorf("decode password hash: %w", err)
	}
	return params, salt, key, nil
}

func parseParams(encoded string) (passwordParams, error) {
	values := map[string]string{}
	for part := range strings.SplitSeq(encoded, ",") {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			return passwordParams{}, errors.New("invalid password hash params")
		}
		values[key] = value
	}

	memory, err := parseUint32(values["m"])
	if err != nil {
		return passwordParams{}, err
	}
	iterations, err := parseUint32(values["t"])
	if err != nil {
		return passwordParams{}, err
	}
	parallelism, err := parseUint8(values["p"])
	if err != nil {
		return passwordParams{}, err
	}

	return passwordParams{
		memory:      memory,
		iterations:  iterations,
		parallelism: parallelism,
	}, nil
}

func parseUint32(value string) (uint32, error) {
	parsed, err := strconv.ParseUint(value, 10, 32)
	return uint32(parsed), err
}

func parseUint8(value string) (uint8, error) {
	parsed, err := strconv.ParseUint(value, 10, 8)
	return uint8(parsed), err
}
