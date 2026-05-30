package ingest

import (
	"testing"

	"github.com/woodleighschool/woodstar/internal/hosts"
)

func TestParseHostDetails(t *testing.T) {
	details := map[string]map[string]string{
		"os_version": {
			"name":     "macOS",
			"version":  "26.5",
			"build":    "25F5068a",
			"major":    "26",
			"minor":    "5",
			"platform": "darwin",
		},
		"osquery_info": {
			"version": "5.22.1",
		},
		"system_info": {
			"hostname":           "example-host",
			"computer_name":      "EXAMPLE-HOST",
			"uuid":               "00000000-0000-4000-8000-000000000000",
			"hardware_serial":    "ABCDEFGHIJ",
			"hardware_model":     "Mac15,8",
			"hardware_version":   "1.0",
			"hardware_vendor":    "Apple Inc.",
			"cpu_type":           "arm64e",
			"cpu_subtype":        "2",
			"cpu_brand":          "Apple M3",
			"cpu_logical_cores":  "8",
			"cpu_physical_cores": "8",
			"physical_memory":    "68719476736",
		},
		"uptime": {
			"total_seconds": "3600",
		},
		"root_disk": {
			"bytes_total":     "409600",
			"bytes_available": "102400",
		},
		"primary_interface": {
			"primary_ip":  "192.0.2.10",
			"primary_mac": "aa:bb:cc:dd:ee:ff",
		},
		"platform_info": {
			"extra": "Darwin Kernel Version 25.5.0",
		},
		"orbit_info": {
			"version": "1.47.0",
		},
	}

	got := ParseHostDetails(details)
	want := hosts.InventoryUpdate{
		Hostname:     "example-host",
		ComputerName: "EXAMPLE-HOST",
		Hardware: hosts.HostHardware{
			UUID:            "00000000-0000-4000-8000-000000000000",
			Serial:          "ABCDEFGHIJ",
			ModelIdentifier: "Mac15,8",
			Vendor:          "Apple Inc.",
			MemoryBytes:     68719476736,
			CPU: hosts.HostCPU{
				Architecture:  "arm64e",
				Subtype:       "2",
				Brand:         "Apple M3",
				LogicalCores:  8,
				PhysicalCores: 8,
			},
		},
		OS: hosts.HostOS{
			Name:          "macOS",
			Version:       "26.5",
			Build:         "25F5068a",
			Platform:      "darwin",
			KernelVersion: "Darwin Kernel Version 25.5.0",
		},
		Storage: hosts.HostStorage{BootVolume: hosts.HostBootVolume{
			AvailableBytes: new(int64(102400)),
			TotalBytes:     new(int64(409600)),
		}},
		Network: hosts.InventoryNetwork{
			PrimaryIP:  "192.0.2.10",
			PrimaryMAC: "aa:bb:cc:dd:ee:ff",
		},
		Agents: hosts.HostAgents{
			Osquery: hosts.HostOsqueryAgent{Version: "5.22.1"},
			Orbit:   hosts.HostOrbitAgent{Version: "1.47.0"},
		},
	}
	assertInventoryUpdate(t, got, want)
}

func TestParseHostDetailsMissingFields(t *testing.T) {
	got := ParseHostDetails(map[string]map[string]string{
		"system_info": {
			"cpu_logical_cores": "not-a-number",
			"physical_memory":   "",
		},
	})

	if got.Hardware.CPU.LogicalCores != 0 {
		t.Fatalf("logical cores = %d, want 0", got.Hardware.CPU.LogicalCores)
	}
	if got.Hardware.MemoryBytes != 0 {
		t.Fatalf("memory bytes = %d, want 0", got.Hardware.MemoryBytes)
	}
}

func assertInventoryUpdate(t *testing.T, got hosts.InventoryUpdate, want hosts.InventoryUpdate) {
	t.Helper()
	if got.Hardware.UUID != want.Hardware.UUID ||
		got.Hostname != want.Hostname ||
		got.ComputerName != want.ComputerName ||
		got.Hardware.Serial != want.Hardware.Serial ||
		got.Hardware.ModelIdentifier != want.Hardware.ModelIdentifier ||
		got.Hardware.Vendor != want.Hardware.Vendor ||
		got.Hardware.MemoryBytes != want.Hardware.MemoryBytes ||
		got.Hardware.CPU != want.Hardware.CPU ||
		got.OS != want.OS ||
		got.Agents != want.Agents ||
		got.Network.PrimaryIP != want.Network.PrimaryIP ||
		got.Network.PrimaryMAC != want.Network.PrimaryMAC {
		t.Fatalf("ParseHostDetails() = %#v, want %#v", got, want)
	}
	if got.Timestamps.LastRestartedAt == nil {
		t.Fatalf("LastRestartedAt is nil, want timestamp")
	}
	assertInt64Ptr(
		t,
		"storage.boot_volume.available_bytes",
		got.Storage.BootVolume.AvailableBytes,
		want.Storage.BootVolume.AvailableBytes,
	)
	assertInt64Ptr(
		t,
		"storage.boot_volume.total_bytes",
		got.Storage.BootVolume.TotalBytes,
		want.Storage.BootVolume.TotalBytes,
	)
}

func assertInt64Ptr(t *testing.T, name string, got *int64, want *int64) {
	t.Helper()
	switch {
	case got == nil && want == nil:
	case got == nil || want == nil:
		t.Fatalf("%s = %v, want %v", name, got, want)
	case *got != *want:
		t.Fatalf("%s = %d, want %d", name, *got, *want)
	}
}
