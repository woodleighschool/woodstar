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
		update.Hardware.UUID = row["uuid"]
		update.Hostname = row["hostname"]
		update.ComputerName = row["computer_name"]
		update.Hardware.Serial = row["hardware_serial"]
		update.Hardware.ModelIdentifier = row["hardware_model"]
		update.Hardware.Vendor = row["hardware_vendor"]
		update.Hardware.CPU.Architecture = row["cpu_type"]
		update.Hardware.CPU.Subtype = row["cpu_subtype"]
		update.Hardware.CPU.Brand = row["cpu_brand"]
		update.Hardware.CPU.LogicalCores = parseInt32(row["cpu_logical_cores"])
		update.Hardware.CPU.PhysicalCores = parseInt32(row["cpu_physical_cores"])
		update.Hardware.MemoryBytes = parseInt64(row["physical_memory"])
	}
	if row := details["osquery_info"]; row != nil {
		update.Agents.Osquery.Version = row["version"]
	}
	if row := details["orbit_info"]; row != nil {
		update.Agents.Orbit.Version = row["version"]
	}
	if row := details["os_version"]; row != nil {
		update.OS.Name = row["name"]
		update.OS.Version = versionString(row)
		update.OS.Build = row["build"]
		update.OS.Platform = row["platform"]
	}
	if row := details["platform_info"]; row != nil {
		update.OS.KernelVersion = row["extra"]
	}
	if row := details["uptime"]; row != nil {
		if seconds := parsePositiveInt64Ptr(row["total_seconds"]); seconds != nil {
			restarted := time.Now().Add(-time.Duration(*seconds) * time.Second)
			update.Timestamps.LastRestartedAt = &restarted
		}
	}
	if row := details["root_disk"]; row != nil {
		total := parseInt64(row["bytes_total"])
		available := parseInt64(row["bytes_available"])
		if total > 0 {
			update.Storage.BootVolume.TotalBytes = new(total)
			if available >= 0 {
				update.Storage.BootVolume.AvailableBytes = new(available)
			}
		}
	}
	if row := details["primary_interface"]; row != nil {
		update.Network.PrimaryIP = row["primary_ip"]
		update.Network.PrimaryMAC = row["primary_mac"]
	}
	return update
}

func versionString(row map[string]string) string {
	version := row["version"]
	if version == "" {
		version = dottedVersion(row)
	}
	return version
}

func dottedVersion(row map[string]string) string {
	parts := make([]string, 0, 4)
	for _, key := range []string{"major", "minor", "patch"} {
		if value := row[key]; value != "" {
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
