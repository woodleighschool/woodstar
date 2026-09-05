//go:build postgres

package rbac

import (
	"testing"

	"github.com/woodleighschool/goodies/auth/authz"

	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestEffectivePermissionsMergeDirectAndGroupRoles(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database)
	service, err := authz.NewService(store, Resources())
	if err != nil {
		t.Fatal(err)
	}

	var userID int64
	if err := database.QueryRow(ctx, `
INSERT INTO users (email, name, source)
VALUES ('group-member@example.invalid', 'Group Member', 'local')
RETURNING id`).Scan(&userID); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if allowed, err := service.HasAccess(ctx, userID); err != nil || allowed {
		t.Fatalf("unassigned access = %v, error = %v", allowed, err)
	}

	var groupID int64
	if err := database.QueryRow(ctx, `
INSERT INTO directory_groups (source, external_id, display_name)
VALUES ('entra', 'group-external-id', 'Directory Group')
RETURNING id`).Scan(&groupID); err != nil {
		t.Fatalf("create group: %v", err)
	}
	if _, err := database.Exec(ctx, `
INSERT INTO directory_group_memberships (user_id, group_id)
VALUES ($1, $2)`, userID, groupID); err != nil {
		t.Fatalf("create group membership: %v", err)
	}
	if _, err := database.Exec(ctx, `
INSERT INTO authz_group_roles (group_id, role_id)
SELECT $1, id FROM authz_roles WHERE key = 'viewer'`, groupID); err != nil {
		t.Fatalf("assign group role: %v", err)
	}

	permissions, err := service.EffectivePermissions(ctx, userID)
	if err != nil {
		t.Fatalf("get permissions: %v", err)
	}
	if got := permissions[ResourceHosts]; got != authz.View {
		t.Fatalf("inherited hosts permission = %q, want %q", got, authz.View)
	}
	if got := permissions[ResourceAgentSecrets]; got != authz.None {
		t.Fatalf("inherited agent secrets permission = %q, want %q", got, authz.None)
	}

	if _, err := database.Exec(ctx, `
INSERT INTO authz_user_roles (user_id, role_id)
SELECT $1, id FROM authz_roles WHERE key = 'admin'`, userID); err != nil {
		t.Fatalf("assign direct role: %v", err)
	}
	grants, err := store.Grants(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	levels := map[authz.Access]int{}
	for _, grant := range grants {
		if grant.Resource == ResourceHosts {
			levels[grant.Access]++
		}
	}
	if levels[authz.View] != 1 || levels[authz.Edit] != 1 {
		t.Fatalf("store aggregated direct and inherited grants: %v", levels)
	}
	permissions, err = service.EffectivePermissions(ctx, userID)
	if err != nil {
		t.Fatalf("get merged permissions: %v", err)
	}
	for _, resource := range Resources() {
		if got := permissions[resource]; got != authz.Edit {
			t.Errorf("merged %s permission = %q, want %q", resource, got, authz.Edit)
		}
	}
}
