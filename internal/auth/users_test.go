package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/models"
)

// Self-mutation guards short-circuit before touching any store, so they exercise
// without a database.

func TestUpdateUserRejectsOwnRoleChange(t *testing.T) {
	svc := NewService(&models.UserStore{}, nil)
	role := models.RoleViewer

	_, err := svc.UpdateUser(context.Background(), 5, 5, UpdateUserParams{Role: &role})
	if !errors.Is(err, ErrCannotChangeOwnRole) {
		t.Fatalf("err = %v, want ErrCannotChangeOwnRole", err)
	}
}

func TestDeleteUserRejectsSelf(t *testing.T) {
	svc := NewService(&models.UserStore{}, nil)

	err := svc.DeleteUser(context.Background(), 9, 9)
	if !errors.Is(err, ErrCannotDeleteSelf) {
		t.Fatalf("err = %v, want ErrCannotDeleteSelf", err)
	}
}

func TestDeleteUserRequiresStores(t *testing.T) {
	svc := NewService(nil, nil)

	err := svc.DeleteUser(context.Background(), 1, 2)
	if !errors.Is(err, ErrNotSetup) {
		t.Fatalf("err = %v, want ErrNotSetup", err)
	}
}
