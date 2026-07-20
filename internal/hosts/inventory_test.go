package hosts

import "testing"

func TestInventoryDisplayNamePriority(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   InventoryUpdate
		want string
	}{
		{
			name: "computer name wins",
			in: InventoryUpdate{
				ComputerName: "Example MacBook Pro",
				Hostname:     "example-macbook-pro",
				Hardware:     HostHardware{UUID: "uuid-1"},
			},
			want: "Example MacBook Pro",
		},
		{
			name: "hostname when no computer name",
			in:   InventoryUpdate{Hostname: "example-macbook-pro", Hardware: HostHardware{UUID: "uuid-1"}},
			want: "example-macbook-pro",
		},
		{
			name: "uuid when no friendly name",
			in:   InventoryUpdate{Hardware: HostHardware{UUID: "uuid-1"}},
			want: "uuid-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := inventoryDisplayName(tt.in.Hardware.UUID, tt.in.Hostname, tt.in.ComputerName); got != tt.want {
				t.Fatalf("inventoryDisplayName = %q, want %q", got, tt.want)
			}
		})
	}
}
