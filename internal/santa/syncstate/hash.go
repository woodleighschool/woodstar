package syncstate

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/woodleighschool/woodstar/internal/fault"
)

func validateRulesHash(rulesHash string) error {
	if len(rulesHash) != 32 || strings.ToLower(rulesHash) != rulesHash {
		return fmt.Errorf("%w: rules_hash must be 32 lowercase hexadecimal characters", fault.ErrInvalidInput)
	}
	if _, err := hex.DecodeString(rulesHash); err != nil {
		return fmt.Errorf("%w: rules_hash must be 32 lowercase hexadecimal characters", fault.ErrInvalidInput)
	}
	return nil
}
