package directory

import (
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

func TestLoadHostUserAffinityUsesReportedEmailWithoutDirectoryFields(t *testing.T) {
	database, ctx := dbtest.Open(t)
	store := NewStore(database)
	hostStore := hosts.NewStore(database)
	deviceMappings := hosts.NewDeviceMappingStore(database)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.DetailUpdate{
		HardwareUUID: "affinity-no-directory-host",
		OrbitNodeKey: "affinity-no-directory-node",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	if err := deviceMappings.Upsert(
		ctx,
		host.ID,
		"test1@woodleigh.vic.edu.au",
		hosts.DeviceMappingSourceSantaPrimaryUser,
	); err != nil {
		t.Fatalf("upsert santa affinity: %v", err)
	}

	affinity, err := store.LoadHostUserAffinity(ctx, host.ID)
	if err != nil {
		t.Fatalf("load affinity: %v", err)
	}
	if affinity == nil {
		t.Fatal("affinity is nil")
	}
	if affinity.Email != "test1@woodleigh.vic.edu.au" ||
		affinity.Source != hosts.DeviceMappingSourceSantaPrimaryUser ||
		affinity.Username != "" ||
		affinity.Name != "" ||
		affinity.Department != "" ||
		len(affinity.Groups) != 0 {
		t.Fatalf("affinity = %+v, want reported email without directory-owned fields", affinity)
	}
}

func TestLoadHostUserAffinityAddsDirectoryFieldsAfterDirectoryLink(t *testing.T) {
	database, ctx := dbtest.Open(t)
	store := NewStore(database)
	hostStore := hosts.NewStore(database)
	deviceMappings := hosts.NewDeviceMappingStore(database)

	if err := store.Apply(ctx, Snapshot{
		GeneratedAt: time.Now().UTC(),
		Groups: []SnapshotGroup{
			{ExternalID: "group-a", DisplayName: "a"},
			{ExternalID: "group-b", DisplayName: "b"},
			{ExternalID: "group-c", DisplayName: "c"},
		},
		Users: []SnapshotUser{
			{
				ExternalID:        "test1-id",
				UserPrincipalName: "test1@woodleigh.vic.edu.au",
				MailNickname:      "test1",
				DisplayName:       "Test 1",
				Department:        "test",
				Active:            true,
				GroupExternalIDs:  []string{"group-a", "group-b", "group-c"},
			},
		},
	}); err != nil {
		t.Fatalf("apply directory snapshot: %v", err)
	}

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.DetailUpdate{
		HardwareUUID: "affinity-directory-host",
		OrbitNodeKey: "affinity-directory-node",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	if err := deviceMappings.Upsert(
		ctx,
		host.ID,
		"test1@woodleigh.vic.edu.au",
		hosts.DeviceMappingSourceSantaPrimaryUser,
	); err != nil {
		t.Fatalf("upsert santa affinity: %v", err)
	}
	if err := store.Apply(ctx, Snapshot{
		GeneratedAt: time.Now().UTC(),
		Groups: []SnapshotGroup{
			{ExternalID: "group-a", DisplayName: "a"},
			{ExternalID: "group-b", DisplayName: "b"},
			{ExternalID: "group-c", DisplayName: "c"},
		},
		Users: []SnapshotUser{
			{
				ExternalID:        "test1-id",
				UserPrincipalName: "test1@woodleigh.vic.edu.au",
				MailNickname:      "test1",
				DisplayName:       "Test 1",
				Department:        "test",
				Active:            true,
				GroupExternalIDs:  []string{"group-a", "group-b", "group-c"},
			},
		},
	}); err != nil {
		t.Fatalf("apply directory snapshot after affinity: %v", err)
	}

	affinity, err := store.LoadHostUserAffinity(ctx, host.ID)
	if err != nil {
		t.Fatalf("load affinity: %v", err)
	}
	if affinity == nil {
		t.Fatal("affinity is nil")
	}
	if affinity.Email != "test1@woodleigh.vic.edu.au" ||
		affinity.Source != hosts.DeviceMappingSourceSantaPrimaryUser ||
		affinity.Username != "test1" ||
		affinity.Name != "Test 1" ||
		affinity.Department != "test" ||
		!sameStrings(affinity.Groups, []string{"a", "b", "c"}) {
		t.Fatalf("affinity = %+v, want directory-owned fields", affinity)
	}
}

func sameStrings(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
