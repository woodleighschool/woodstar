package ingest

import (
	"testing"

	"github.com/woodleighschool/woodstar/internal/hosts"
)

func TestParseHostDetails(t *testing.T) {
	details := map[string]map[string]string{
		"os_version": {
			"name":          "macOS",
			"version":       "26.5",
			"build":         "25F5068a",
			"major":         "26",
			"minor":         "5",
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
	want := hosts.DetailUpdate{
		OSVersion:               "macOS 26.5 (build 25F5068a)",
		Platform:                "darwin",
		PlatformLike:            "darwin",
		Hostname:                "example-host",
		ComputerName:            "EXAMPLE-HOST",
		HardwareUUID:            "00000000-0000-4000-8000-000000000000",
		HardwareSerial:          "ABCDEFGHIJ",
		HardwareModel:           "Mac15,8",
		HardwareVersion:         "1.0",
		HardwareVendor:          "Apple Inc.",
		OSName:                  "macOS",
		OSBuild:                 "25F5068a",
		CPUType:                 "arm64e",
		CPUSubtype:              "2",
		CPUBrand:                "Apple M3",
		CPULogicalCores:         8,
		CPUPhysicalCores:        8,
		PhysicalMemory:          68719476736,
		OsqueryVersion:          "5.22.1",
		OrbitVersion:            "1.47.0",
		KernelVersion:           "Darwin Kernel Version 25.5.0",
		UptimeSeconds:           new(int64(3600)),
		DiskSpaceAvailableBytes: new(int64(102400)),
		DiskSpaceTotalBytes:     new(int64(409600)),
		PrimaryIP:               "192.0.2.10",
		PrimaryMAC:              "aa:bb:cc:dd:ee:ff",
	}
	assertDetailUpdate(t, got, want)
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

func assertDetailUpdate(t *testing.T, got hosts.DetailUpdate, want hosts.DetailUpdate) {
	t.Helper()
	if got.HardwareUUID != want.HardwareUUID ||
		got.Hostname != want.Hostname ||
		got.ComputerName != want.ComputerName ||
		got.HardwareSerial != want.HardwareSerial ||
		got.HardwareModel != want.HardwareModel ||
		got.HardwareVersion != want.HardwareVersion ||
		got.HardwareVendor != want.HardwareVendor ||
		got.OSName != want.OSName ||
		got.OSVersion != want.OSVersion ||
		got.OSBuild != want.OSBuild ||
		got.Platform != want.Platform ||
		got.PlatformLike != want.PlatformLike ||
		got.KernelVersion != want.KernelVersion ||
		got.OrbitVersion != want.OrbitVersion ||
		got.CPUType != want.CPUType ||
		got.CPUSubtype != want.CPUSubtype ||
		got.CPUBrand != want.CPUBrand ||
		got.CPULogicalCores != want.CPULogicalCores ||
		got.CPUPhysicalCores != want.CPUPhysicalCores ||
		got.PhysicalMemory != want.PhysicalMemory ||
		got.OsqueryVersion != want.OsqueryVersion ||
		got.PrimaryIP != want.PrimaryIP ||
		got.PrimaryMAC != want.PrimaryMAC {
		t.Fatalf("ParseHostDetails() = %#v, want %#v", got, want)
	}
	assertInt64Ptr(t, "UptimeSeconds", got.UptimeSeconds, want.UptimeSeconds)
	assertInt64Ptr(t, "DiskSpaceAvailableBytes", got.DiskSpaceAvailableBytes, want.DiskSpaceAvailableBytes)
	assertInt64Ptr(t, "DiskSpaceTotalBytes", got.DiskSpaceTotalBytes, want.DiskSpaceTotalBytes)
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
