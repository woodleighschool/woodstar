package labels

import (
	"context"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func TestLabelMutationValidate(t *testing.T) {
	t.Parallel()
	query := "select 1;"
	tests := []struct {
		name    string
		in      LabelMutation
		wantErr string
	}{
		{
			name: "dynamic label with query is valid",
			in: LabelMutation{
				Name:                "Macs",
				Query:               &query,
				LabelMembershipType: LabelMembershipTypeDynamic,
			},
		},
		{
			name: "dynamic label without query is invalid",
			in: LabelMutation{
				Name:                "No query",
				LabelMembershipType: LabelMembershipTypeDynamic,
			},
			wantErr: "query is required for dynamic labels",
		},
		{
			name: "manual label with query is invalid",
			in: LabelMutation{
				Name:                "Manual",
				Query:               &query,
				LabelMembershipType: LabelMembershipTypeManual,
			},
			wantErr: "query is only allowed for dynamic labels",
		},
		{
			name: "derived label with criteria is valid",
			in: LabelMutation{
				Name: "Department",
				Criteria: &Criteria{
					Attribute: DerivedAttributeDirectoryDepartment,
					Values:    []string{"Engineering"},
				},
				LabelMembershipType: LabelMembershipTypeDerived,
			},
		},
		{
			name: "derived label without criteria is invalid",
			in: LabelMutation{
				Name:                "Department",
				LabelMembershipType: LabelMembershipTypeDerived,
			},
			wantErr: "criteria is required for derived labels",
		},
		{
			name: "dynamic label with hosts is invalid",
			in: LabelMutation{
				Name:                "Dynamic hosts",
				Query:               &query,
				HostIDs:             []int64{1},
				LabelMembershipType: LabelMembershipTypeDynamic,
			},
			wantErr: "hosts are only allowed for manual labels",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.in.Validate()
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Validate error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Validate returned error: %v", err)
			}
		})
	}
}

func TestCreateManualLabelStoresHostIDs(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := NewStore(db)
	hostA := insertHost(t, db, "manual-a")
	hostB := insertHost(t, db, "manual-b")

	label, err := store.Create(ctx, LabelMutation{
		Name:                "Manual",
		LabelMembershipType: LabelMembershipTypeManual,
		HostIDs:             []int64{hostB, hostA, hostA},
	})
	if err != nil {
		t.Fatalf("create label: %v", err)
	}

	if label.HostsCount != 2 {
		t.Fatalf("HostsCount = %d, want 2", label.HostsCount)
	}
	wantHostIDs := []int64{hostA, hostB}
	if !equalInt64s(label.HostIDs, wantHostIDs) {
		t.Fatalf("HostIDs = %v, want %v", label.HostIDs, wantHostIDs)
	}
}

func TestListIncludesDerivedCriteria(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := NewStore(db)

	label, err := store.Create(ctx, LabelMutation{
		Name:                "Engineering",
		LabelMembershipType: LabelMembershipTypeDerived,
		Criteria:            &Criteria{Attribute: DerivedAttributeDirectoryDepartment, Values: []string{"Engineering"}},
	})
	if err != nil {
		t.Fatalf("create label: %v", err)
	}

	labels, count, err := store.List(ctx, ListParams{})
	if err != nil {
		t.Fatalf("list labels: %v", err)
	}
	if count == 0 {
		t.Fatal("count = 0, want at least one label")
	}
	var got *Label
	for i := range labels {
		if labels[i].ID == label.ID {
			got = &labels[i]
			break
		}
	}
	if got == nil {
		t.Fatalf("created label %d was not returned", label.ID)
	}
	if got.Criteria == nil || got.Criteria.Attribute != DerivedAttributeDirectoryDepartment ||
		len(got.Criteria.Values) != 1 || got.Criteria.Values[0] != "Engineering" {
		t.Fatalf("Criteria = %#v, want department Engineering", got.Criteria)
	}
}

func TestDerivedLabelsMatchDirectoryAttributes(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := NewStore(db)
	hostA := insertHost(t, db, "derived-a")
	hostB := insertHost(t, db, "derived-b")
	aliceID := insertDirectoryUser(t, db, "alice", "alice@example.com", "Engineering")
	bobID := insertDirectoryUser(t, db, "bob", "bob@example.com", "Operations")
	linkHostDirectoryUser(t, db, hostA, aliceID)
	linkHostDirectoryUser(t, db, hostB, bobID)
	staffID := insertDirectoryGroup(t, db, "staff", "Staff")
	linkDirectoryUserGroup(t, db, aliceID, staffID)

	tests := []struct {
		name       string
		criteria   Criteria
		wantHostID int64
	}{
		{
			name:       "department",
			criteria:   Criteria{Attribute: DerivedAttributeDirectoryDepartment, Values: []string{"Engineering"}},
			wantHostID: hostA,
		},
		{
			name:       "group",
			criteria:   Criteria{Attribute: DerivedAttributeDirectoryGroup, Values: []string{"staff"}},
			wantHostID: hostA,
		},
		{
			name:       "user",
			criteria:   Criteria{Attribute: DerivedAttributeDirectoryUser, Values: []string{"alice"}},
			wantHostID: hostA,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			label, err := store.Create(ctx, LabelMutation{
				Name:                "Derived " + tt.name,
				LabelMembershipType: LabelMembershipTypeDerived,
				Criteria:            &tt.criteria,
			})
			if err != nil {
				t.Fatalf("create label: %v", err)
			}
			if label.HostsCount != 1 {
				t.Fatalf("HostsCount = %d, want 1", label.HostsCount)
			}
			matched, err := hostHasLabel(ctx, db, tt.wantHostID, label.ID)
			if err != nil {
				t.Fatalf("lookup membership: %v", err)
			}
			if !matched {
				t.Fatalf("host %d did not receive label %d", tt.wantHostID, label.ID)
			}
		})
	}
}

func TestRefreshDerivedLabelsRecomputesMembership(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := NewStore(db)
	hostID := insertHost(t, db, "derived-refresh")
	userID := insertDirectoryUser(t, db, "refresh-user", "refresh@example.com", "Engineering")
	linkHostDirectoryUser(t, db, hostID, userID)

	label, err := store.Create(ctx, LabelMutation{
		Name:                "Refresh derived",
		LabelMembershipType: LabelMembershipTypeDerived,
		Criteria:            &Criteria{Attribute: DerivedAttributeDirectoryDepartment, Values: []string{"Engineering"}},
	})
	if err != nil {
		t.Fatalf("create label: %v", err)
	}

	if _, err := db.Pool().
		Exec(ctx, `UPDATE directory_users SET department = 'Operations' WHERE id = $1`, userID); err != nil {
		t.Fatalf("update directory user: %v", err)
	}
	if err := store.RefreshDerived(ctx); err != nil {
		t.Fatalf("refresh derived labels: %v", err)
	}
	matched, err := hostHasLabel(ctx, db, hostID, label.ID)
	if err != nil {
		t.Fatalf("lookup membership: %v", err)
	}
	if matched {
		t.Fatalf("host %d still has label %d after criteria no longer matches", hostID, label.ID)
	}
}

func insertHost(t *testing.T, db *database.DB, hardwareUUID string) int64 {
	t.Helper()
	var id int64
	if err := db.Pool().QueryRow(context.Background(), `
INSERT INTO hosts (hardware_uuid)
VALUES ($1)
RETURNING id`, hardwareUUID).Scan(&id); err != nil {
		t.Fatalf("insert host: %v", err)
	}
	return id
}

func insertDirectoryUser(t *testing.T, db *database.DB, externalID string, upn string, department string) int64 {
	t.Helper()
	var id int64
	if err := db.Pool().QueryRow(context.Background(), `
INSERT INTO directory_users (external_id, user_principal_name, mail, display_name, department, last_synced_at)
VALUES ($1, $2, $2, $2, $3, now())
RETURNING id`, externalID, upn, dbutil.NullString(department)).Scan(&id); err != nil {
		t.Fatalf("insert directory user: %v", err)
	}
	return id
}

func linkHostDirectoryUser(t *testing.T, db *database.DB, hostID int64, userID int64) {
	t.Helper()
	if _, err := db.Pool().Exec(context.Background(), `
INSERT INTO host_directory_user (host_id, directory_user_id, source)
VALUES ($1, $2, 'manual')`, hostID, userID); err != nil {
		t.Fatalf("link host directory user: %v", err)
	}
}

func insertDirectoryGroup(t *testing.T, db *database.DB, externalID string, displayName string) int64 {
	t.Helper()
	var id int64
	if err := db.Pool().QueryRow(context.Background(), `
INSERT INTO directory_groups (external_id, display_name, last_synced_at)
VALUES ($1, $2, now())
RETURNING id`, externalID, displayName).Scan(&id); err != nil {
		t.Fatalf("insert directory group: %v", err)
	}
	return id
}

func linkDirectoryUserGroup(t *testing.T, db *database.DB, userID int64, groupID int64) {
	t.Helper()
	if _, err := db.Pool().Exec(context.Background(), `
INSERT INTO directory_user_groups (directory_user_id, directory_group_id)
VALUES ($1, $2)`, userID, groupID); err != nil {
		t.Fatalf("link directory user group: %v", err)
	}
}

func hostHasLabel(ctx context.Context, db *database.DB, hostID int64, labelID int64) (bool, error) {
	var exists bool
	err := db.Pool().QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM label_membership
    WHERE host_id = $1 AND label_id = $2
)`, hostID, labelID).Scan(&exists)
	return exists, err
}

func equalInt64s(a []int64, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
