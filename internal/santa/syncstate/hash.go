package syncstate

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/woodleighschool/woodstar/internal/fault"
)

func validateRulesHash(rulesHash string) error {
	return validateHexDigest("rules_hash", rulesHash, 32)
}

func validatePolicyDigest(digest string) error {
	return validateHexDigest("policy digest", digest, 64)
}

func validateHexDigest(name string, value string, length int) error {
	if len(value) != length || strings.ToLower(value) != value {
		return fmt.Errorf("%w: %s must be %d lowercase hexadecimal characters", fault.ErrInvalidInput, name, length)
	}
	if _, err := hex.DecodeString(value); err != nil {
		return fmt.Errorf("%w: %s must be %d lowercase hexadecimal characters", fault.ErrInvalidInput, name, length)
	}
	return nil
}
