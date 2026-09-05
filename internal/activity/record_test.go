package activity

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/woodleighschool/goodies/auth/authn"
)

func TestRecordUserBuildsActorSnapshot(t *testing.T) {
	ctx := authn.WithPrincipal(t.Context(), &authn.Principal{
		ID:    42,
		Name:  "Admin User",
		Email: "admin@example.test",
	})
	var got NewEvent
	RecordUser(ctx, recorderFunc(func(_ context.Context, event NewEvent) error {
		got = event
		return nil
	}), slog.New(slog.DiscardHandler), AreaOsquery, ActionPolicyUpdated, Resource("policy", 7, "Gatekeeper"))

	if got.Area != AreaOsquery || got.Action != ActionPolicyUpdated || got.Actor.Kind != ActorKindUser ||
		got.Actor.UserID == nil || *got.Actor.UserID != 42 || got.Actor.Name != "Admin User" ||
		got.Actor.Email != "admin@example.test" || got.Subject.ID == nil || *got.Subject.ID != 7 ||
		got.Subject.Name != "Gatekeeper" {
		t.Fatalf("recorded event = %+v, want authenticated actor and policy subject", got)
	}
}

func TestRecordUserIgnoresRecorderFailure(t *testing.T) {
	ctx := authn.WithPrincipal(t.Context(), &authn.Principal{ID: 42})
	RecordUser(ctx, recorderFunc(func(context.Context, NewEvent) error {
		return errors.New("database unavailable")
	}), slog.New(slog.DiscardHandler), AreaHosts, ActionHostDeleted, Resource("host", 7, "Lab Mac"))
}

func TestRecordSystemUsesSystemActor(t *testing.T) {
	var got NewEvent
	RecordSystem(t.Context(), recorderFunc(func(_ context.Context, event NewEvent) error {
		got = event
		return nil
	}), slog.New(slog.DiscardHandler), AreaOsquery, ActionOrbitHostEnrolled, Resource("host", 7, "Lab Mac"))

	if got.Actor.Kind != ActorKindSystem || got.Actor.Name != "System" {
		t.Fatalf("recorded actor = %+v, want system actor named System", got.Actor)
	}
}

type recorderFunc func(context.Context, NewEvent) error

func (record recorderFunc) Record(ctx context.Context, event NewEvent) error {
	return record(ctx, event)
}
