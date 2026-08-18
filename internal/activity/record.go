package activity

import (
	"context"
	"log/slog"

	"github.com/woodleighschool/woodstar/internal/api/ctxkeys"
)

// Recorder accepts activity events from capability owners.
type Recorder interface {
	Record(context.Context, NewEvent) error
}

// RecordUser records a user action without changing the outcome of the action itself.
func RecordUser(
	ctx context.Context,
	recorder Recorder,
	logger *slog.Logger,
	area Area,
	action Action,
	subject Subject,
) {
	if recorder == nil {
		return
	}
	user, ok := ctxkeys.User(ctx)
	if !ok {
		activityLogger(logger).ErrorContext(ctx, "record activity without authenticated user", "action", action)
		return
	}
	userID := user.ID
	record(ctx, recorder, logger, NewEvent{
		Area:   area,
		Action: action,
		Actor: Actor{
			Kind:   ActorKindUser,
			UserID: &userID,
			Name:   user.Name,
			Email:  user.Email,
		},
		Subject: subject,
	})
}

// RecordSystem records a Woodstar-initiated action without changing the operation outcome.
func RecordSystem(
	ctx context.Context,
	recorder Recorder,
	logger *slog.Logger,
	area Area,
	action Action,
	subject Subject,
) {
	if recorder == nil {
		return
	}
	record(ctx, recorder, logger, NewEvent{
		Area:    area,
		Action:  action,
		Actor:   Actor{Kind: ActorKindSystem, Name: "Woodstar"},
		Subject: subject,
	})
}

func record(ctx context.Context, recorder Recorder, logger *slog.Logger, event NewEvent) {
	if err := recorder.Record(ctx, event); err != nil {
		activityLogger(logger).ErrorContext(ctx, "record activity failed",
			"area", event.Area,
			"action", event.Action,
			"subject_type", event.Subject.Type,
			"subject_id", event.Subject.ID,
			"err", err,
		)
	}
}

func activityLogger(logger *slog.Logger) *slog.Logger {
	if logger == nil {
		return slog.Default()
	}
	return logger
}

// Resource identifies an activity subject with a durable numeric ID.
func Resource(resourceType string, id int64, name string) Subject {
	return Subject{Type: resourceType, ID: &id, Name: name}
}

// Collection identifies an aggregate activity without a single resource ID.
func Collection(resourceType string, name string) Subject {
	return Subject{Type: resourceType, Name: name}
}
