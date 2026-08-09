package entra_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/directory/entra"
)

func TestSyncReportsAppliedSnapshot(t *testing.T) {
	service := entra.NewService(
		discardingApplier{},
		staticFetcher{snapshot: directory.ProviderSnapshot{
			Users:  []directory.ProviderUser{{ExternalID: "user-1"}},
			Groups: []directory.ProviderGroup{{ExternalID: "group-1"}},
		}},
		slog.New(slog.DiscardHandler),
	)

	result, err := service.Sync(t.Context())
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if result.Users != 1 || result.Groups != 1 {
		t.Fatalf("Sync() result = %+v, want one user and one group", result)
	}
}

type discardingApplier struct{}

func (discardingApplier) ApplyProviderSnapshot(
	context.Context,
	directory.Source,
	directory.ProviderSnapshot,
) error {
	return nil
}

type staticFetcher struct {
	snapshot directory.ProviderSnapshot
}

func (f staticFetcher) Fetch(context.Context) (directory.ProviderSnapshot, error) {
	return f.snapshot, nil
}
