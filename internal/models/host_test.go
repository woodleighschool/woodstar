package models

import (
	"context"
	"os"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database"
)

func TestDisplayNameForPriority(t *testing.T) {
	tests := []struct {
		name string
		in   EnrollParams
		want string
	}{
		{
			name: "computer name wins",
			in: EnrollParams{
				ComputerName: "Example MacBook Pro",
				Hostname:     "example-macbook-pro",
				HardwareUUID: "uuid-1",
			},
			want: "Example MacBook Pro",
		},
		{
			name: "hostname when no computer name",
			in:   EnrollParams{Hostname: "example-macbook-pro", HardwareUUID: "uuid-1"},
			want: "example-macbook-pro",
		},
		{
			name: "uuid when no friendly name",
			in:   EnrollParams{HardwareUUID: "uuid-1"},
			want: "uuid-1",
		},
		{
			name: "whitespace-only fields fall through",
			in:   EnrollParams{ComputerName: "  ", Hostname: " ", HardwareUUID: "uuid-2"},
			want: "uuid-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := displayNameFor(tt.in); got != tt.want {
				t.Fatalf("displayNameFor = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestApplyDetailAcceptsBigPhysicalMemory(t *testing.T) {
	databaseURL := os.Getenv("WOODSTAR_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("WOODSTAR_TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	db, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(db.Close)

	store := NewHostStore(db)
	host, err := store.UpsertOnOsqueryEnroll(ctx, HostDetailUpdate{
		HardwareUUID:   "test-apply-detail-big-memory",
		OsqueryNodeKey: "node-key",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	const memoryBytes = int64(68719476736)
	if err := store.ApplyDetail(ctx, host.ID, HostDetailUpdate{PhysicalMemory: memoryBytes}); err != nil {
		t.Fatalf("apply detail: %v", err)
	}

	got, err := store.GetByID(ctx, host.ID)
	if err != nil {
		t.Fatalf("get host: %v", err)
	}
	if got.PhysicalMemory != memoryBytes {
		t.Fatalf("PhysicalMemory = %d, want %d", got.PhysicalMemory, memoryBytes)
	}
}
