package auth

import "testing"

func TestHashTokenUsesSessionSecret(t *testing.T) {
	token := "session-token"

	first := hashToken("secret-one", token)
	second := hashToken("secret-two", token)

	if first == second {
		t.Fatal("hashToken returned the same hash for different secrets")
	}
}
