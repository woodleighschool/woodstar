package directory

import (
	"errors"
	"testing"
)

func TestPasswordMutationsRejectShortPasswords(t *testing.T) {
	users := NewUserService(nil)
	password := "elevenchars"
	for _, tt := range []struct {
		name string
		run  func() error
	}{
		{"create", func() error {
			_, err := users.Create(t.Context(), UserCreate{
				Email: "viewer@example.invalid", Password: password, Role: RoleViewer,
			})
			return err
		}},
		{"admin update", func() error {
			_, err := users.Update(t.Context(), 1, UserMutation{Password: &password})
			return err
		}},
		{"account update", func() error {
			_, err := users.UpdateAccount(t.Context(), 1, AccountMutation{Password: &password})
			return err
		}},
		{"password reset", func() error {
			_, err := users.SetPasswordByEmail(t.Context(), "viewer@example.invalid", password)
			return err
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); !errors.Is(err, ErrWeakPassword) {
				t.Fatalf("password mutation error = %v, want %v", err, ErrWeakPassword)
			}
		})
	}
}

func TestHashPasswordAcceptsMinimumLength(t *testing.T) {
	if _, err := hashPassword("twelvechars!"); err != nil {
		t.Fatalf("hash minimum-length password: %v", err)
	}
}
