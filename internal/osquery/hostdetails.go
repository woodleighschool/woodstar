package osquery

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/woodleighschool/woodstar/internal/models"
)

// ParseHostDetails converts osquery enroll details into inventory fields.
func ParseHostDetails(details map[string]map[string]string) models.HostDetailUpdate {
	var update models.HostDetailUpdate
	if row := details["system_info"]; row != nil {
		update.HardwareUUID = strings.TrimSpace(row["uuid"])
		update.Hostname = strings.TrimSpace(row["hostname"])
		update.ComputerName = strings.TrimSpace(row["computer_name"])
		update.HardwareSerial = strings.TrimSpace(row["hardware_serial"])
		update.HardwareModel = strings.TrimSpace(row["hardware_model"])
		update.HardwareVendor = strings.TrimSpace(row["hardware_vendor"])
		update.CPUBrand = strings.TrimSpace(row["cpu_brand"])
		update.CPULogicalCores = parseInt(row["cpu_logical_cores"])
		update.CPUPhysicalCores = parseInt(row["cpu_physical_cores"])
		update.PhysicalMemory = parseInt64(row["physical_memory"])
	}
	if row := details["osquery_info"]; row != nil {
		update.OsqueryVersion = strings.TrimSpace(row["version"])
	}
	if row := details["os_version"]; row != nil {
		update.OSVersion = osVersion(row)
		update.Platform = strings.TrimSpace(row["platform"])
		update.PlatformLike = strings.TrimSpace(row["platform_like"])
	}
	if row := details["platform_info"]; row != nil {
		update.KernelVersion = strings.TrimSpace(row["extra"])
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
