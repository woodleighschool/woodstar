package osquery

import "crypto/rand"

const nodeKeyLength = 24

const nodeKeyAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateNodeKey() (string, error) {
	buf := make([]byte, nodeKeyLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	for i, b := range buf {
		buf[i] = nodeKeyAlphabet[int(b)%len(nodeKeyAlphabet)]
	}
	return string(buf), nil
}
