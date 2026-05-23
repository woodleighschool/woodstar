// Package enrollment contains the shared enrollment credentials used by Orbit and osquery.
package enrollment

import (
	"crypto/rand"
	"errors"
	"math/big"
)

// Enrollment failures shared by Orbit and osquery enrollment.
var (
	ErrInvalidEnrollSecret = errors.New("invalid enroll secret")
	ErrMissingHardwareUUID = errors.New("hardware_uuid is required")
)

// NodeKeyLength is the generated Orbit/osquery node-key length.
const NodeKeyLength = 24

// NodeKeyAlphabet keeps generated node keys URL-safe.
const NodeKeyAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateNodeKey returns a new random node key for enrollment.
func GenerateNodeKey() (string, error) {
	alphabetLen := big.NewInt(int64(len(NodeKeyAlphabet)))
	buf := make([]byte, NodeKeyLength)
	for i := range buf {
		n, err := rand.Int(rand.Reader, alphabetLen)
		if err != nil {
			return "", err
		}
		buf[i] = NodeKeyAlphabet[n.Int64()]
	}
	return string(buf), nil
}
