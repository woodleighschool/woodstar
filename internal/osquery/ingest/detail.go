package ingest

import (
	"strconv"
	"strings"
	"time"

	"github.com/woodleighschool/woodstar/internal/hosts"
)

// ParseHostDetails maps osquery detail rows to host fields.
func ParseHostDetails(details map[string]map[string]string) hosts.InventoryUpdate {
	var update hosts.InventoryUpdate
	if row := details["system_info"]; row != nil {
		update.Hardware.UUID = normalizeString(row["uuid"])
		update.Hostname = normalizeString(row["hostname"])
		update.ComputerName = normalizeString(row["computer_name"])
		update.Hardware.Serial = normalizeString(row["hardware_serial"])
		update.Hardware.ModelIdentifier = normalizeString(row["hardware_model"])
		update.Hardware.Vendor = normalizeString(row["hardware_vendor"])
		update.Hardware.CPU.Architecture = normalizeString(row["cpu_type"])
		update.Hardware.CPU.Subtype = normalizeString(row["cpu_subtype"])
		update.Hardware.CPU.Brand = normalizeString(row["cpu_brand"])
		update.Hardware.CPU.LogicalCores = parseInt32(normalizeString(row["cpu_logical_cores"]))
		update.Hardware.CPU.PhysicalCores = parseInt32(normalizeString(row["cpu_physical_cores"]))
		update.Hardware.MemoryBytes = parseInt64(normalizeString(row["physical_memory"]))
	}
	if row := details["osquery_info"]; row != nil {
		update.Agents.Osquery.Version = normalizeString(row["version"])
	}
	if row := details["orbit_info"]; row != nil {
		update.Agents.Orbit.Version = normalizeString(row["version"])
	}
	if row := details["os_version"]; row != nil {
		update.OS.Name = normalizeString(row["name"])
		update.OS.Version = versionString(row)
		update.OS.Build = normalizeString(row["build"])
		update.OS.Platform = normalizeString(row["platform"])
	}
	if row := details["platform_info"]; row != nil {
		update.OS.KernelVersion = normalizeString(row["extra"])
	}
	if row := details["uptime"]; row != nil {
		if seconds := parsePositiveInt64Ptr(normalizeString(row["total_seconds"])); seconds != nil {
			restarted := time.Now().Add(-time.Duration(*seconds) * time.Second)
			update.Timestamps.LastRestartedAt = &restarted
		}
	}
	if row := details["root_disk"]; row != nil {
		total := parseInt64(normalizeString(row["bytes_total"]))
		available := parseInt64(normalizeString(row["bytes_available"]))
		if total > 0 {
			update.Storage.BootVolume.TotalBytes = new(total)
			if available >= 0 {
				update.Storage.BootVolume.AvailableBytes = new(available)
			}
		}
	}
	if row := details["primary_interface"]; row != nil {
		update.Network.PrimaryIP = normalizeString(row["primary_ip"])
		update.Network.PrimaryMAC = normalizeString(row["primary_mac"])
	}
	return update
}

func normalizeString(value string) string {
	return strings.ReplaceAll(value, "\x00", "")
}

func versionString(row map[string]string) string {
	version := normalizeString(row["version"])
	if version == "" {
		version = dottedVersion(row)
	}
	return version
}

func dottedVersion(row map[string]string) string {
	parts := make([]string, 0, 4)
	for _, key := range []string{"major", "minor", "patch"} {
		if value := normalizeString(row[key]); value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, ".")
}

func parseInt32(value string) int32 {
	parsed, _ := strconv.ParseInt(value, 10, 32)
	return int32(parsed)
}

func parseInt64(value string) int64 {
	parsed, _ := strconv.ParseInt(value, 10, 64)
	return parsed
}

func parsePositiveInt64Ptr(value string) *int64 {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return nil
	}
	return new(parsed)
}
