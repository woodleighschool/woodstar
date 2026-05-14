package directory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

func TestReconcileLinksMatchesByUPNAndRespectsManual(t *testing.T) {
	database, ctx := dbtest.Open(t)
	store := NewStore(database)
	hostStore := hosts.NewStore(database)
	deviceMappings := hosts.NewDeviceMappingStore(database)

	if err := store.Apply(ctx, Snapshot{
		GeneratedAt: time.Now().UTC(),
		Users: []SnapshotUser{
			{ExternalID: "u-alice", UserPrincipalName: "alice@example.com", DisplayName: "Alice", Active: true},
			{ExternalID: "u-bob", UserPrincipalName: "bob@example.com", DisplayName: "Bob", Active: true},
		},
	}); err != nil {
		t.Fatalf("apply directory snapshot: %v", err)
	}

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.EnrollParams{
		HardwareUUID:   "fixture-uuid",
		HardwareSerial: "fixture-serial",
		Hostname:       "fixture",
		Platform:       "darwin",
		OrbitNodeKey:   "fixture-node",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	if err := deviceMappings.Upsert(
		ctx, host.ID, "alice@example.com", hosts.DeviceMappingSourceOrbitProfile,
	); err != nil {
		t.Fatalf("seed host_emails: %v", err)
	}

	if err := store.ReconcileLinks(ctx); err != nil {
		t.Fatalf("reconcile links: %v", err)
	}

	link, err := store.GetHostLink(ctx, host.ID)
	if err != nil {
		t.Fatalf("get host link: %v", err)
	}
	if link.Source != HostLinkSourceMDMEmail {
		t.Fatalf("source = %q, want %q", link.Source, HostLinkSourceMDMEmail)
	}

	aliceUPN, err := store.GetUserByUPN(ctx, "alice@example.com")
	if err != nil {
		t.Fatalf("lookup alice: %v", err)
	}
	if link.DirectoryUserID != aliceUPN.ID {
		t.Fatalf("link points to %d, want alice's id %d", link.DirectoryUserID, aliceUPN.ID)
	}

	bob, err := store.GetUserByUPN(ctx, "bob@example.com")
	if err != nil {
		t.Fatalf("lookup bob: %v", err)
	}
	if _, err := store.SetManualHostLink(ctx, host.ID, bob.ID); err != nil {
		t.Fatalf("manual override: %v", err)
	}

	if err := store.ReconcileLinks(ctx); err != nil {
		t.Fatalf("reconcile after manual: %v", err)
	}

	link, err = store.GetHostLink(ctx, host.ID)
	if err != nil {
		t.Fatalf("get host link after manual: %v", err)
	}
	if link.Source != HostLinkSourceManual {
		t.Fatalf("source = %q, want %q (manual must stick)", link.Source, HostLinkSourceManual)
	}
	if link.DirectoryUserID != bob.ID {
		t.Fatalf("link = %d, want bob %d", link.DirectoryUserID, bob.ID)
	}

	if err := store.DeleteHostLink(context.Background(), host.ID); err != nil {
		t.Fatalf("delete host link: %v", err)
	}
	if _, err := store.GetHostLink(ctx, host.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
