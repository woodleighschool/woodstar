// Package crudtest is the shared conformance harness for Tier 1 resource stores.
//
// Every store that exposes the uniform CRUD+List contract wires its fixtures
// into [RunConformance]. A store that diverges from the contract or its expected
// behavior fails here, which is the enforcement mechanism for the store rewrite.
// The harness imports no concrete store: each resource supplies builders through
// [Fixtures].
//
// Assertions tolerate pre-existing rows so stores that seed builtin or migrated
// rows conform: the harness tracks the row it created by ID rather than assuming
// an empty table or a fixed total count.
package crudtest

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// Resource is the uniform CRUD+List contract every Tier 1 store implements. D is
// the domain type, C the create input, M the mutation, and P the list params.
type Resource[D, C, M, P any] interface {
	Create(ctx context.Context, in C) (*D, error)
	GetByID(ctx context.Context, id int64) (*D, error)
	Update(ctx context.Context, id int64, m M) (*D, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, params P) ([]D, int, error)
}

// Fixtures supplies the per-resource builders the harness needs. Closures keep
// the harness generic: it never names a concrete store, domain, or column.
type Fixtures[D, C, M, P any] struct {
	// Store is the resource under test.
	Store Resource[D, C, M, P]
	// NewValid creates any prerequisites (parent rows, FKs) and returns a valid
	// create input. It runs once per RunConformance.
	NewValid func(t *testing.T, ctx context.Context) C
	// Mutate derives a mutation from a created resource. The mutation must change
	// at least one field GetByID reflects so the harness can observe the update.
	Mutate func(D) M
	// ID extracts the primary key from a domain value.
	ID func(D) int64
	// ListParams builds the store's list params from the common knobs.
	ListParams func(q, sort string, pageIndex, pageSize int32) P
	// SortKeys lists every key the store declares in its OrderKeys map. The
	// harness asserts each one sorts both ascending and descending.
	SortKeys []string
	// SearchMatch returns a query string that matches the created resource. Nil
	// skips the search assertion.
	SearchMatch func(D) string
	// NewInvalid returns a create input the store must reject with
	// dbutil.ErrInvalidInput. ok=false skips the invalid-input assertion.
	NewInvalid func() (C, bool)
}

// RunConformance exercises the uniform CRUD+List contract against a live store.
// It uses subtests so a single divergence reports against the behavior that
// broke rather than failing the whole suite.
func RunConformance[D, C, M, P any](t *testing.T, ctx context.Context, f Fixtures[D, C, M, P]) {
	t.Helper()

	created := mustCreate(t, ctx, f, f.NewValid(t, ctx))
	id := f.ID(*created)

	t.Run("GetByID matches create", func(t *testing.T) { assertGetMatchesCreate(t, ctx, f, *created) })
	t.Run("Update reflects in GetByID", func(t *testing.T) { assertUpdateReflects(t, ctx, f, *created) })
	t.Run("List contains created", func(t *testing.T) { assertListContains(t, ctx, f, id) })
	t.Run("List paginates", func(t *testing.T) { assertListPaginates(t, ctx, f) })
	t.Run("List sorts by every declared key", func(t *testing.T) { assertListSorts(t, ctx, f, id) })
	t.Run("List finds created by search", func(t *testing.T) { assertListSearch(t, ctx, f, *created, id) })
	t.Run("Create rejects invalid input", func(t *testing.T) { assertRejectsInvalid(t, ctx, f) })
	t.Run("Delete then GetByID not found", func(t *testing.T) { assertDeleteNotFound(t, ctx, f, id) })
}

func assertGetMatchesCreate[D, C, M, P any](t *testing.T, ctx context.Context, f Fixtures[D, C, M, P], created D) {
	t.Helper()

	got, err := f.Store.GetByID(ctx, f.ID(created))
	if err != nil {
		t.Fatalf("GetByID(%d): %v", f.ID(created), err)
	}
	if f.ID(*got) != f.ID(created) {
		t.Fatalf("GetByID id = %d, want %d", f.ID(*got), f.ID(created))
	}
	if !reflect.DeepEqual(created, *got) {
		t.Fatalf("GetByID does not match create return:\n got  %+v\n want %+v", *got, created)
	}
}

func assertUpdateReflects[D, C, M, P any](t *testing.T, ctx context.Context, f Fixtures[D, C, M, P], created D) {
	t.Helper()

	id := f.ID(created)
	updated, err := f.Store.Update(ctx, id, f.Mutate(created))
	if err != nil {
		t.Fatalf("Update(%d): %v", id, err)
	}
	got, err := f.Store.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID(%d) after update: %v", id, err)
	}
	if !reflect.DeepEqual(*updated, *got) {
		t.Fatalf("GetByID does not match update return:\n got  %+v\n want %+v", *got, *updated)
	}
}

func assertListContains[D, C, M, P any](t *testing.T, ctx context.Context, f Fixtures[D, C, M, P], id int64) {
	t.Helper()

	rows, count, err := f.Store.List(ctx, f.ListParams("", "", 0, 50))
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if count < 1 {
		t.Fatalf("List count = %d, want >= 1", count)
	}
	if !containsID(f, rows, id) {
		t.Fatalf("List does not contain created id %d", id)
	}
}

func assertListPaginates[D, C, M, P any](t *testing.T, ctx context.Context, f Fixtures[D, C, M, P]) {
	t.Helper()

	const pageSize = 1
	first, total, err := f.Store.List(ctx, f.ListParams("", "", 0, pageSize))
	if err != nil {
		t.Fatalf("List page 0 size %d: %v", pageSize, err)
	}
	if len(first) > pageSize {
		t.Fatalf("List page 0 size %d returned %d items, want at most %d", pageSize, len(first), pageSize)
	}
	if total < 1 {
		t.Fatalf("List page 0 size %d count = %d, want full count >= 1", pageSize, total)
	}
	past, pastTotal, err := f.Store.List(ctx, f.ListParams("", "", int32(total), pageSize))
	if err != nil {
		t.Fatalf("List page past end: %v", err)
	}
	if len(past) != 0 {
		t.Fatalf("List page past end returned %d items, want 0", len(past))
	}
	if pastTotal != total {
		t.Fatalf("List page past end count = %d, want %d", pastTotal, total)
	}
}

func assertListSorts[D, C, M, P any](t *testing.T, ctx context.Context, f Fixtures[D, C, M, P], id int64) {
	t.Helper()

	for _, key := range f.SortKeys {
		for _, sort := range []string{key, key + ".desc"} {
			rows, count, err := f.Store.List(ctx, f.ListParams("", sort, 0, 50))
			if err != nil {
				t.Fatalf("List sort %q: %v", sort, err)
			}
			if count < 1 || !containsID(f, rows, id) {
				t.Fatalf("List sort %q dropped created id %d (count %d)", sort, id, count)
			}
		}
	}
}

func assertListSearch[D, C, M, P any](t *testing.T, ctx context.Context, f Fixtures[D, C, M, P], created D, id int64) {
	t.Helper()

	if f.SearchMatch == nil {
		t.Skip("no SearchMatch fixture")
	}
	query := f.SearchMatch(created)
	rows, _, err := f.Store.List(ctx, f.ListParams(query, "", 0, 50))
	if err != nil {
		t.Fatalf("List search %q: %v", query, err)
	}
	if !containsID(f, rows, id) {
		t.Fatalf("List search %q did not match created id %d", query, id)
	}
}

func assertRejectsInvalid[D, C, M, P any](t *testing.T, ctx context.Context, f Fixtures[D, C, M, P]) {
	t.Helper()

	if f.NewInvalid == nil {
		t.Skip("no NewInvalid fixture")
	}
	invalid, ok := f.NewInvalid()
	if !ok {
		t.Skip("NewInvalid declined")
	}
	if _, err := f.Store.Create(ctx, invalid); !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("Create(invalid) error = %v, want ErrInvalidInput", err)
	}
}

func assertDeleteNotFound[D, C, M, P any](t *testing.T, ctx context.Context, f Fixtures[D, C, M, P], id int64) {
	t.Helper()

	if err := f.Store.Delete(ctx, id); err != nil {
		t.Fatalf("Delete(%d): %v", id, err)
	}
	if _, err := f.Store.GetByID(ctx, id); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("GetByID(%d) after delete error = %v, want ErrNotFound", id, err)
	}
	if err := f.Store.Delete(ctx, id); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("second Delete(%d) error = %v, want ErrNotFound", id, err)
	}
}

func mustCreate[D, C, M, P any](t *testing.T, ctx context.Context, f Fixtures[D, C, M, P], in C) *D {
	t.Helper()
	created, err := f.Store.Create(ctx, in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created == nil {
		t.Fatal("Create returned nil resource")
	}
	return created
}

func containsID[D, C, M, P any](f Fixtures[D, C, M, P], rows []D, id int64) bool {
	for i := range rows {
		if f.ID(rows[i]) == id {
			return true
		}
	}
	return false
}
