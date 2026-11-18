//go:build postgres

package directory

import (
	"errors"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/listing"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestDeleteRemovesLocalUsers(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database, labels.NewStore(database))
	service := newTestUserService(store)
	if _, err := service.Create(ctx, UserCreate{
		Email:    "other-admin@example.test",
		Name:     "Other Admin",
		Role:     RoleAdmin,
		Password: "correct-password",
	}); err != nil {
		t.Fatalf("create other admin: %v", err)
	}

	user, err := service.Create(ctx, UserCreate{
		Email:    "local@example.test",
		Name:     "Local User",
		Role:     RoleAdmin,
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	if err := service.Delete(ctx, user.ID); err != nil {
		t.Fatalf("delete user: %v", err)
	}
	if _, err := store.GetUserByID(ctx, user.ID); !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("get deleted user err = %v, want %v", err, fault.ErrNotFound)
	}
}

func TestListFiltersUsers(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database, labels.NewStore(database))
	service := newTestUserService(store)

	if err := store.ApplyProviderSnapshot(ctx, SourceEntra, ProviderSnapshot{
		GeneratedAt: time.Now().UTC(),
		Groups: []ProviderGroup{
			{ExternalID: "all-users", DisplayName: "All Users", MailNickname: "all-users"},
			{ExternalID: "engineering", DisplayName: "Engineering", MailNickname: "engineering"},
		},
		Users: []ProviderUser{
			{
				ExternalID:        "u-alice",
				UserPrincipalName: "alice@example.edu",
				DisplayName:       "Alice Engineering",
				Department:        "Engineering",
				Enabled:           true,
				GroupExternalIDs:  []string{"all-users", "engineering"},
			},
			{
				ExternalID:        "u-bob",
				UserPrincipalName: "bob@example.edu",
				DisplayName:       "Bob Operations",
				Department:        "Operations",
				Enabled:           false,
				GroupExternalIDs:  []string{"all-users"},
			},
		},
	}); err != nil {
		t.Fatalf("apply entra snapshot: %v", err)
	}
	var engineeringGroupID int64
	if err := database.QueryRow(ctx, `
SELECT id
FROM directory_groups
WHERE external_id = 'engineering'`).Scan(&engineeringGroupID); err != nil {
		t.Fatalf("get engineering group id: %v", err)
	}
	local, err := service.Create(ctx, UserCreate{
		Email:    "local@example.edu",
		Name:     "Local Admin",
		Role:     RoleAdmin,
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("create local user: %v", err)
	}

	users, count, err := store.ListUsers(ctx, UserListParams{
		ListParams: listing.Params{Q: "engineering"},
		Source:     "entra",
	})
	if err != nil {
		t.Fatalf("list Entra engineering users: %v", err)
	}
	if count != 1 || len(users) != 1 || users[0].Email != "alice@example.edu" {
		t.Fatalf("users = %+v count=%d, want Alice only", users, count)
	}

	users, count, err = store.ListUsers(ctx, UserListParams{Source: "entra"})
	if err != nil {
		t.Fatalf("list current Entra users: %v", err)
	}
	if count != 2 || len(users) != 2 || !users[0].AccountEnabled || users[1].AccountEnabled {
		t.Fatalf("current Entra users = %+v count=%d, want enabled Alice and disabled Bob", users, count)
	}

	users, count, err = store.ListUsers(ctx, UserListParams{GroupID: engineeringGroupID})
	if err != nil {
		t.Fatalf("list engineering group users: %v", err)
	}
	if count != 1 || len(users) != 1 || users[0].Email != "alice@example.edu" {
		t.Fatalf("engineering group users = %+v count=%d, want Alice only", users, count)
	}

	users, count, err = store.ListUsers(ctx, UserListParams{
		Roles:  []string{"admin"},
		Source: "local",
	})
	if err != nil {
		t.Fatalf("list local admins: %v", err)
	}
	if count != 1 || len(users) != 1 || users[0].ID != local.ID {
		t.Fatalf("local admins = %+v count=%d, want local admin", users, count)
	}

	users, count, err = store.ListUsers(ctx, UserListParams{
		Roles: []string{"admin", "none"},
	})
	if err != nil {
		t.Fatalf("list multiple roles and sources: %v", err)
	}
	if count != 3 || len(users) != 3 {
		t.Fatalf("multi-filter users = %+v count=%d, want local admin and both Entra members", users, count)
	}
}

func TestListDepartmentsReturnsDirectoryDepartments(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database, labels.NewStore(database))

	if err := store.ApplyProviderSnapshot(ctx, SourceEntra, ProviderSnapshot{
		GeneratedAt: time.Now().UTC(),
		Users: []ProviderUser{
			{
				ExternalID:        "u-alice",
				UserPrincipalName: "alice@example.edu",
				DisplayName:       "Alice",
				Department:        "Engineering",
				Enabled:           true,
			},
			{
				ExternalID:        "u-bob",
				UserPrincipalName: "bob@example.edu",
				DisplayName:       "Bob",
				Department:        "Operations",
				Enabled:           true,
			},
		},
	}); err != nil {
		t.Fatalf("apply entra snapshot: %v", err)
	}

	departments, count, err := store.ListDepartments(ctx, UserListParams{ListParams: listing.Params{Q: "eng"}})
	if err != nil {
		t.Fatalf("list departments: %v", err)
	}
	if count != 1 || len(departments) != 1 || departments[0].Value != "Engineering" {
		t.Fatalf("departments = %+v count=%d, want Engineering only", departments, count)
	}
}
