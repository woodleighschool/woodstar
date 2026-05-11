package users

import (
	"context"
	"errors"
	"testing"
)

// Self-mutation guards short-circuit before touching any store, so they exercise
// without a database.

func TestUpdateRejectsOwnRoleChange(t *testing.T) {
	svc := NewService(&Store{})
	actor := &User{ID: 5, Role: RoleAdmin}

	_, err := svc.Update(context.Background(), actor, actor.ID, UpdateParams{
		Name: "John Doe",
		Role: RoleViewer,
	})
	if !errors.Is(err, ErrCannotChangeOwnRole) {
		t.Fatalf("err = %v, want ErrCannotChangeOwnRole", err)
	}
}

func TestDeleteRejectsSelf(t *testing.T) {
	svc := NewService(&Store{})

	err := svc.Delete(context.Background(), 9, 9)
	if !errors.Is(err, ErrCannotDeleteSelf) {
		t.Fatalf("err = %v, want ErrCannotDeleteSelf", err)
	}
}
