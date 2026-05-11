package hosts

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseHostDetails converts osquery enroll details into inventory fields.
func ParseHostDetails(details map[string]map[string]string) HostDetailUpdate {
	var update HostDetailUpdate
	if row := details["system_info"]; row != nil {
		update.HardwareUUID = strings.TrimSpace(row["uuid"])
		update.Hostname = strings.TrimSpace(row["hostname"])
		update.ComputerName = strings.TrimSpace(row["computer_name"])
		update.HardwareSerial = strings.TrimSpace(row["hardware_serial"])
		update.HardwareModel = strings.TrimSpace(row["hardware_model"])
		update.HardwareVersion = strings.TrimSpace(row["hardware_version"])
		update.HardwareVendor = strings.TrimSpace(row["hardware_vendor"])
		update.CPUType = strings.TrimSpace(row["cpu_type"])
		update.CPUSubtype = strings.TrimSpace(row["cpu_subtype"])
		update.CPUBrand = strings.TrimSpace(row["cpu_brand"])
		update.CPULogicalCores = parseInt(row["cpu_logical_cores"])
		update.CPUPhysicalCores = parseInt(row["cpu_physical_cores"])
		update.PhysicalMemory = parseInt64(row["physical_memory"])
	}
	if row := details["osquery_info"]; row != nil {
		update.OsqueryVersion = strings.TrimSpace(row["version"])
	}
	if row := details["orbit_info"]; row != nil {
		update.OrbitVersion = strings.TrimSpace(row["version"])
	}
	if row := details["os_version"]; row != nil {
		update.OSName = strings.TrimSpace(row["name"])
		update.OSVersion = osVersion(row)
		update.OSBuild = strings.TrimSpace(row["build"])
		update.Platform = strings.TrimSpace(row["platform"])
		update.PlatformLike = strings.TrimSpace(row["platform_like"])
	}
	if row := details["platform_info"]; row != nil {
		update.KernelVersion = strings.TrimSpace(row["extra"])
	}
	if row := details["uptime"]; row != nil {
		update.UptimeSeconds = parsePositiveInt64Ptr(row["total_seconds"])
	}
	if row := details["root_disk"]; row != nil {
		total := parseInt64(row["bytes_total"])
		available := parseInt64(row["bytes_available"])
		if total > 0 {
			update.DiskSpaceTotalBytes = &total
			if available >= 0 {
				update.DiskSpaceAvailableBytes = &available
			}
		}
	}
	if row := details["primary_interface"]; row != nil {
		update.PrimaryIP = strings.TrimSpace(row["primary_ip"])
		update.PrimaryMAC = strings.TrimSpace(row["primary_mac"])
	}
	return update
}

func osVersion(row map[string]string) string {
	name := strings.TrimSpace(row["name"])
	version := strings.TrimSpace(row["version"])
	if version == "" {
		version = dottedVersion(row)
	}
	build := strings.TrimSpace(row["build"])
	switch {
	case name == "":
		return version
	case version == "":
		return name
	case build == "":
		return name + " " + version
	default:
		return fmt.Sprintf("%s %s (build %s)", name, version, build)
	}
}

func dottedVersion(row map[string]string) string {
	parts := make([]string, 0, 4)
	for _, key := range []string{"major", "minor", "patch"} {
		if value := strings.TrimSpace(row[key]); value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, ".")
}

func parseInt(value string) int {
	parsed, _ := strconv.Atoi(strings.TrimSpace(value))
	return parsed
}

func parseInt64(value string) int64 {
	parsed, _ := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	return parsed
}

func parsePositiveInt64Ptr(value string) *int64 {
	parsed := parseInt64(value)
	if parsed <= 0 {
		return nil
	}
	return new(parsed)
}
