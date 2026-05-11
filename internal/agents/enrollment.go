// Package agents contains shared authentication primitives for agent protocols.
package agents

import (
	"crypto/rand"
	"errors"
)

// Enrollment failures shared by agent protocols that exchange enroll secrets for node keys.
var (
	ErrInvalidEnrollSecret = errors.New("invalid enroll secret")
	ErrMissingHardwareUUID = errors.New("hardware_uuid is required")
)

// NodeKeyLength is the generated Orbit/osquery node-key length.
const NodeKeyLength = 24

// NodeKeyAlphabet keeps generated node keys URL-safe.
const NodeKeyAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateNodeKey returns a new random node key for enrolling agents.
func GenerateNodeKey() (string, error) {
	buf := make([]byte, NodeKeyLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	for i, b := range buf {
		buf[i] = NodeKeyAlphabet[int(b)%len(NodeKeyAlphabet)]
	}
	return string(buf), nil
}
