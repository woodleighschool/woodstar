//go:build postgres

package activity

import (
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/listing"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestActivityStoreFiltersSnapshotsAndSweeps(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := NewStore(db)

	var userID int64
	if err := db.QueryRow(ctx, `
		INSERT INTO users (email, name, role)
		VALUES ('admin@example.test', 'Admin User', 'admin')
		RETURNING id`).Scan(&userID); err != nil {
		t.Fatalf("insert actor: %v", err)
	}

	if err := store.Record(ctx, NewEvent{
		Area:   AreaOsquery,
		Action: ActionPolicyCreated,
		Actor: Actor{
			Kind: ActorKindUser, UserID: &userID, Name: "Admin User", Email: "admin@example.test",
		},
		Subject: Resource("policy", 42, "Gatekeeper enabled"),
	}); err != nil {
		t.Fatalf("record osquery activity: %v", err)
	}
	if err := store.Record(ctx, NewEvent{
		Area:    AreaHosts,
		Action:  ActionHostDeleted,
		Actor:   Actor{Kind: ActorKindSystem, Name: "System"},
		Subject: Resource("host", 7, "Lab Mac"),
	}); err != nil {
		t.Fatalf("record host activity: %v", err)
	}

	if _, err := db.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID); err != nil {
		t.Fatalf("delete actor: %v", err)
	}
	events, count, err := store.List(ctx, ListParams{
		ListParams: listing.Params{PageSize: 10},
		Area:       AreaOsquery,
	})
	if err != nil {
		t.Fatalf("list osquery activity: %v", err)
	}
	if count != 1 || len(events) != 1 {
		t.Fatalf("events = %+v, count = %d, want one", events, count)
	}
	if events[0].Actor.UserID != nil || events[0].Actor.Name != "Admin User" ||
		events[0].Subject.Name != "Gatekeeper enabled" {
		t.Fatalf("activity snapshot = %+v, want retained actor and subject names", events[0])
	}

	recent := time.Now().UTC()
	if _, err := db.Exec(ctx, `
		UPDATE activity_events SET occurred_at = $1 WHERE action = $2`,
		recent, ActionPolicyCreated); err != nil {
		t.Fatalf("set policy activity time: %v", err)
	}
	events, count, err = store.List(ctx, ListParams{
		ListParams:  listing.Params{PageSize: 10},
		Area:        AreaOsquery,
		ActorKind:   ActorKindUser,
		Action:      ActionPolicyCreated,
		Since:       recent.Add(-time.Minute),
		Before:      recent.Add(time.Minute),
		SubjectType: "policy",
		SubjectID:   42,
	})
	if err != nil {
		t.Fatalf("list filtered activity: %v", err)
	}
	if count != 1 || len(events) != 1 || events[0].Subject.Name != "Gatekeeper enabled" {
		t.Fatalf("filtered events = %+v, count = %d, want policy activity", events, count)
	}
	if _, _, err := store.List(ctx, ListParams{SubjectID: 42}); err == nil {
		t.Fatal("list activity with subject id but no type succeeded")
	}
	if _, _, err := store.List(ctx, ListParams{Since: recent, Before: recent}); err == nil {
		t.Fatal("list activity with empty time range succeeded")
	}

	cutoff := time.Now().Add(-time.Hour)
	if _, err := db.Exec(ctx, `
		UPDATE activity_events SET occurred_at = $1 WHERE area = 'hosts'`, cutoff.Add(-time.Minute)); err != nil {
		t.Fatalf("backdate host activity: %v", err)
	}
	deleted, err := store.SweepBefore(ctx, cutoff)
	if err != nil {
		t.Fatalf("sweep activity: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}
}
