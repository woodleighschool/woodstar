package users

import (
	"context"
	"errors"
	"testing"
)

func TestDeleteRejectsInitialUser(t *testing.T) {
	svc := NewService(&Store{})

	err := svc.Delete(context.Background(), InitialUserID)
	if !errors.Is(err, ErrCannotDeleteInitialUser) {
		t.Fatalf("err = %v, want ErrCannotDeleteInitialUser", err)
	}
}
