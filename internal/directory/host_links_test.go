package directory

import (
	"context"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

func TestReconcileLinksMatchesByUPNAndRespectsManual(t *testing.T) {
	database, ctx := dbtest.Open(t)
	store := NewStore(database)
	hostStore := hosts.NewStore(database)
	userAffinities := hosts.NewUserAffinityStore(database)

	if err := store.ApplyProviderSnapshot(ctx, SourceEntra, ProviderSnapshot{
		GeneratedAt: time.Now().UTC(),
		Users: []ProviderUser{
			{ExternalID: "u-alice", UserPrincipalName: "alice@example.com", DisplayName: "Alice", Enabled: true},
			{ExternalID: "u-bob", UserPrincipalName: "bob@example.com", DisplayName: "Bob", Enabled: true},
		},
	}); err != nil {
		t.Fatalf("apply entra snapshot: %v", err)
	}

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware: hosts.HostHardware{
			UUID:   "fixture-uuid",
			Serial: "fixture-serial",
		},
		Hostname:     "fixture",
		OrbitNodeKey: "fixture-node",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	if err := userAffinities.Upsert(
		ctx, host.ID, "alice@example.com", hosts.UserAffinitySourceOrbitProfile,
	); err != nil {
		t.Fatalf("seed host user affinity: %v", err)
	}
	santaHost, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware: hosts.HostHardware{
			UUID:   "fixture-santa-uuid",
			Serial: "fixture-santa-serial",
		},
		Hostname:     "fixture-santa",
		OrbitNodeKey: "fixture-santa-node",
	})
	if err != nil {
		t.Fatalf("enroll santa-backed host: %v", err)
	}
	if err := userAffinities.Upsert(
		ctx, santaHost.ID, "bob@example.com", hosts.UserAffinitySourceSantaPrimaryUser,
	); err != nil {
		t.Fatalf("seed santa host email: %v", err)
	}

	if err := store.ApplyProviderSnapshot(ctx, SourceEntra, ProviderSnapshot{
		GeneratedAt: time.Now().UTC(),
		Users: []ProviderUser{
			{ExternalID: "u-alice", UserPrincipalName: "alice@example.com", DisplayName: "Alice", Enabled: true},
			{ExternalID: "u-bob", UserPrincipalName: "bob@example.com", DisplayName: "Bob", Enabled: true},
		},
	}); err != nil {
		t.Fatalf("apply entra snapshot after host email: %v", err)
	}

	linkedUserID, source := hostUserLink(t, ctx, store, host.ID)
	if source != "reported_user_affinity" {
		t.Fatalf("source = %q, want reported_user_affinity", source)
	}

	aliceID := providerUserID(t, ctx, store, "alice@example.com")
	if linkedUserID != aliceID {
		t.Fatalf("link points to %d, want alice's id %d", linkedUserID, aliceID)
	}

	bobID := providerUserID(t, ctx, store, "bob@example.com")
	linkedUserID, source = hostUserLink(t, ctx, store, santaHost.ID)
	if source != "reported_user_affinity" {
		t.Fatalf("santa source = %q, want reported_user_affinity", source)
	}
	if linkedUserID != bobID {
		t.Fatalf("santa link points to %d, want bob's id %d", linkedUserID, bobID)
	}

	if _, err := store.db.Pool().Exec(ctx, `
		INSERT INTO host_user_links (host_id, user_id, source)
		VALUES ($1, $2, 'manual')
		ON CONFLICT (host_id) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			source = 'manual',
			updated_at = now()
	`, host.ID, bobID); err != nil {
		t.Fatalf("manual override: %v", err)
	}

	if err := store.ApplyProviderSnapshot(ctx, SourceEntra, ProviderSnapshot{
		GeneratedAt: time.Now().UTC(),
		Users: []ProviderUser{
			{ExternalID: "u-alice", UserPrincipalName: "alice@example.com", DisplayName: "Alice", Enabled: true},
			{ExternalID: "u-bob", UserPrincipalName: "bob@example.com", DisplayName: "Bob", Enabled: true},
		},
	}); err != nil {
		t.Fatalf("apply entra snapshot after manual: %v", err)
	}

	linkedUserID, source = hostUserLink(t, ctx, store, host.ID)
	if source != "manual" {
		t.Fatalf("source = %q, want manual", source)
	}
	if linkedUserID != bobID {
		t.Fatalf("link = %d, want bob %d", linkedUserID, bobID)
	}
}

func providerUserID(t *testing.T, ctx context.Context, store *Store, upn string) int64 {
	t.Helper()
	var id int64
	if err := store.db.Pool().QueryRow(ctx, `
		SELECT id
		FROM users
		WHERE user_principal_name = $1
	`, upn).Scan(&id); err != nil {
		t.Fatalf("lookup entra user %q: %v", upn, err)
	}
	return id
}

func hostUserLink(t *testing.T, ctx context.Context, store *Store, hostID int64) (int64, string) {
	t.Helper()
	var userID int64
	var source string
	if err := store.db.Pool().QueryRow(ctx, `
		SELECT user_id, source
		FROM host_user_links
		WHERE host_id = $1
	`, hostID).Scan(&userID, &source); err != nil {
		t.Fatalf("lookup host user link: %v", err)
	}
	return userID, source
}
