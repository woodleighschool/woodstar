package labels

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func TestLabelMutationValidate(t *testing.T) {
	t.Parallel()
	query := "select 1;"
	unknownTableQuery := "select * from osquery_info;"
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
			name: "dynamic label with unknown osquery table is valid",
			in: LabelMutation{
				Name:                "Osquery info",
				Query:               &unknownTableQuery,
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
			name: "dynamic label without name is invalid",
			in: LabelMutation{
				Query:               &query,
				LabelMembershipType: LabelMembershipTypeDynamic,
			},
			wantErr: "name is required",
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
					Attribute: DerivedAttributeUserDepartment,
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
		HostIDs:             []int64{hostB, hostA},
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
	if label.BuiltinKey != nil {
		t.Fatalf("BuiltinKey = %q, want nil", *label.BuiltinKey)
	}
}

func TestBuiltinLabelUsesStableKey(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := NewStore(db)

	rows, _, err := store.List(ctx, LabelListParams{LabelType: LabelTypeBuiltin})
	if err != nil {
		t.Fatalf("list builtin labels: %v", err)
	}

	for _, row := range rows {
		if row.BuiltinKey != nil && *row.BuiltinKey == BuiltinKeyAllHosts {
			if row.Name != "All Hosts" {
				t.Fatalf("All Hosts name = %q, want display name", row.Name)
			}
			if row.LabelMembershipType != LabelMembershipTypeManual {
				t.Fatalf("All Hosts membership type = %q, want manual", row.LabelMembershipType)
			}
			return
		}
	}
	t.Fatalf("builtin key %q not found in labels: %+v", BuiltinKeyAllHosts, rows)
}

func TestBuiltinKeyConstraints(t *testing.T) {
	db, ctx := dbtest.Open(t)

	_, err := db.Pool().Exec(ctx, `
		INSERT INTO labels (name, builtin_key, label_type, label_membership_type)
		VALUES ('Bad Regular Key', 'bad-regular', 'regular', 'manual')
	`)
	expectLabelSQLState(t, err, pgerrcode.CheckViolation)

	_, err = db.Pool().Exec(ctx, `
		INSERT INTO labels (name, label_type, label_membership_type)
		VALUES ('Missing Builtin Key', 'builtin', 'manual')
	`)
	expectLabelSQLState(t, err, pgerrcode.CheckViolation)

	_, err = db.Pool().Exec(ctx, `
		INSERT INTO labels (name, builtin_key, label_type, label_membership_type)
		VALUES ('Duplicate All Hosts', 'all-hosts', 'builtin', 'manual')
	`)
	expectLabelSQLState(t, err, pgerrcode.UniqueViolation)

	_, err = db.Pool().Exec(ctx, `
		INSERT INTO labels (name, builtin_key, label_type, label_membership_type)
		VALUES ('Unknown Builtin', 'unknown-builtin', 'builtin', 'manual')
	`)
	expectLabelSQLState(t, err, pgerrcode.CheckViolation)
}

func TestCreateManualLabelWithMissingHostReturnsNotFound(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := NewStore(db)

	_, err := store.Create(ctx, LabelMutation{
		Name:                "Missing host",
		LabelMembershipType: LabelMembershipTypeManual,
		HostIDs:             []int64{0},
	})
	if !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("Create error = %v, want ErrNotFound", err)
	}
}

func TestListIncludesDerivedCriteria(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := NewStore(db)

	label, err := store.Create(ctx, LabelMutation{
		Name:                "Engineering",
		LabelMembershipType: LabelMembershipTypeDerived,
		Criteria:            &Criteria{Attribute: DerivedAttributeUserDepartment, Values: []string{"Engineering"}},
	})
	if err != nil {
		t.Fatalf("create label: %v", err)
	}

	labels, count, err := store.List(ctx, LabelListParams{})
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
	if got.Criteria == nil || got.Criteria.Attribute != DerivedAttributeUserDepartment ||
		len(got.Criteria.Values) != 1 || got.Criteria.Values[0] != "Engineering" {
		t.Fatalf("Criteria = %#v, want department Engineering", got.Criteria)
	}
}

func TestDerivedLabelsMatchUserAndEntraAttributes(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := NewStore(db)
	hostA := insertHost(t, db, "derived-a")
	hostB := insertHost(t, db, "derived-b")
	aliceID := insertUser(t, db, "alice", "alice@example.com", "Engineering")
	insertUser(t, db, "bob", "bob@example.com", "Operations")
	linkHostPrimaryUser(t, db, hostA, "alice@example.com")
	linkHostPrimaryUser(t, db, hostB, "bob@example.com")
	staffID := insertDirectoryGroup(t, db, "staff", "Staff")
	linkDirectoryGroupMembership(t, db, aliceID, staffID)

	tests := []struct {
		name       string
		criteria   Criteria
		wantHostID int64
	}{
		{
			name:       "department",
			criteria:   Criteria{Attribute: DerivedAttributeUserDepartment, Values: []string{"Engineering"}},
			wantHostID: hostA,
		},
		{
			name:       "group",
			criteria:   Criteria{Attribute: DerivedAttributeDirectoryGroup, Values: []string{"staff"}},
			wantHostID: hostA,
		},
		{
			name:       "user",
			criteria:   Criteria{Attribute: DerivedAttributeUser, Values: []string{strconv.FormatInt(aliceID, 10)}},
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
	userID := insertUser(t, db, "refresh-user", "refresh@example.com", "Engineering")
	linkHostPrimaryUser(t, db, hostID, "refresh@example.com")

	label, err := store.Create(ctx, LabelMutation{
		Name:                "Refresh derived",
		LabelMembershipType: LabelMembershipTypeDerived,
		Criteria:            &Criteria{Attribute: DerivedAttributeUserDepartment, Values: []string{"Engineering"}},
	})
	if err != nil {
		t.Fatalf("create label: %v", err)
	}

	if _, err := db.Pool().
		Exec(ctx, `UPDATE users SET department = 'Operations' WHERE id = $1`, userID); err != nil {
		t.Fatalf("update user: %v", err)
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

func insertUser(t *testing.T, db *database.DB, externalID string, email string, department string) int64 {
	t.Helper()
	var id int64
	if err := db.Pool().QueryRow(context.Background(), `
INSERT INTO users (email, name, source, external_id, user_principal_name, department)
VALUES ($1, $1, 'entra', $2, $1, $3)
RETURNING id`, email, externalID, dbutil.NullString(department)).Scan(&id); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return id
}

func linkHostPrimaryUser(t *testing.T, db *database.DB, hostID int64, email string) {
	t.Helper()
	if _, err := db.Pool().Exec(context.Background(), `
INSERT INTO host_primary_user_sources (host_id, email, source)
VALUES ($1, $2, 'manual')`, hostID, email); err != nil {
		t.Fatalf("link host primary user: %v", err)
	}
}

func insertDirectoryGroup(t *testing.T, db *database.DB, externalID string, displayName string) int64 {
	t.Helper()
	var id int64
	if err := db.Pool().QueryRow(context.Background(), `
INSERT INTO directory_groups (source, external_id, display_name)
VALUES ('entra', $1, $2)
RETURNING id`, externalID, displayName).Scan(&id); err != nil {
		t.Fatalf("insert directory group: %v", err)
	}
	return id
}

func linkDirectoryGroupMembership(t *testing.T, db *database.DB, userID int64, groupID int64) {
	t.Helper()
	if _, err := db.Pool().Exec(context.Background(), `
INSERT INTO directory_group_memberships (user_id, group_id)
VALUES ($1, $2)`, userID, groupID); err != nil {
		t.Fatalf("link directory group membership: %v", err)
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

func expectLabelSQLState(t testing.TB, err error, code string) {
	t.Helper()
	if got := labelSQLState(err); got != code {
		t.Fatalf("SQL state = %q, want %q from err %v", got, code, err)
	}
}

func labelSQLState(err error) string {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return ""
	}
	return pgErr.Code
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
