package munki

import (
	"context"
	"errors"
	"testing"
)

func TestSoftwareDeletionServiceSignalsDesiredPackagesAfterDeletion(t *testing.T) {
	t.Parallel()

	store := &softwareDeletionTestStore{deleted: 2}
	var changes int
	service := NewSoftwareDeletionService(store, func() { changes++ })

	if err := service.Delete(t.Context(), 1); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := service.DeleteMany(t.Context(), []int64{2, 3}); err != nil {
		t.Fatalf("delete many: %v", err)
	}
	if changes != 2 {
		t.Fatalf("desired package changes = %d, want 2", changes)
	}
}

func TestSoftwareDeletionServiceDoesNotSignalWithoutDeletion(t *testing.T) {
	t.Parallel()

	store := &softwareDeletionTestStore{}
	var changes int
	service := NewSoftwareDeletionService(store, func() { changes++ })

	if _, err := service.DeleteMany(t.Context(), []int64{1}); err != nil {
		t.Fatalf("delete many: %v", err)
	}
	store.err = errors.New("delete failed")
	if err := service.Delete(t.Context(), 1); err == nil {
		t.Fatal("delete succeeded, want error")
	}
	if changes != 0 {
		t.Fatalf("desired package changes = %d, want 0", changes)
	}
}

type softwareDeletionTestStore struct {
	deleted int
	err     error
}

func (s *softwareDeletionTestStore) Delete(context.Context, int64) error {
	return s.err
}

func (s *softwareDeletionTestStore) DeleteMany(context.Context, []int64) (int, error) {
	return s.deleted, s.err
}
