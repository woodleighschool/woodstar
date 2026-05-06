package osquery

import (
	"testing"

	"github.com/woodleighschool/woodstar/internal/models"
)

func TestParseHostDetails(t *testing.T) {
	details := map[string]map[string]string{
		"os_version": {
			"name":          "macOS",
			"version":       "26.5",
			"build":         "25F5068a",
			"platform":      "darwin",
			"platform_like": "darwin",
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
			"hardware_vendor":    "Apple Inc.",
			"cpu_brand":          "Apple M3",
			"cpu_logical_cores":  "8",
			"cpu_physical_cores": "8",
			"physical_memory":    "68719476736",
		},
		"platform_info": {
			"extra": "Darwin Kernel Version 25.5.0",
		},
		"orbit_info": {
			"version": "1.47.0",
		},
	}

	got := ParseHostDetails(details)
	want := models.HostDetailUpdate{
		OSVersion:        "macOS 26.5 (build 25F5068a)",
		Platform:         "darwin",
		PlatformLike:     "darwin",
		Hostname:         "example-host",
		ComputerName:     "EXAMPLE-HOST",
		HardwareUUID:     "00000000-0000-4000-8000-000000000000",
		HardwareSerial:   "ABCDEFGHIJ",
		HardwareModel:    "Mac15,8",
		HardwareVendor:   "Apple Inc.",
		CPUBrand:         "Apple M3",
		CPULogicalCores:  8,
		CPUPhysicalCores: 8,
		PhysicalMemory:   68719476736,
		OsqueryVersion:   "5.22.1",
		OrbitVersion:     "1.47.0",
		KernelVersion:    "Darwin Kernel Version 25.5.0",
	}
	if got != want {
		t.Fatalf("ParseHostDetails() = %#v, want %#v", got, want)
	}
}

func TestParseHostDetailsMissingFields(t *testing.T) {
	got := ParseHostDetails(map[string]map[string]string{
		"system_info": {
			"cpu_logical_cores": "not-a-number",
			"physical_memory":   "",
		},
	})

	if got.CPULogicalCores != 0 {
		t.Fatalf("CPULogicalCores = %d, want 0", got.CPULogicalCores)
	}
	if got.PhysicalMemory != 0 {
		t.Fatalf("PhysicalMemory = %d, want 0", got.PhysicalMemory)
	}
}
