package ingest

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/woodleighschool/woodstar/internal/hosts"
)

// ParseHostDetails maps osquery detail rows to host fields.
func ParseHostDetails(details map[string]map[string]string) hosts.DetailUpdate {
	var update hosts.DetailUpdate
	if row := details["system_info"]; row != nil {
		update.HardwareUUID = row["uuid"]
		update.Hostname = row["hostname"]
		update.ComputerName = row["computer_name"]
		update.HardwareSerial = row["hardware_serial"]
		update.HardwareModel = row["hardware_model"]
		update.HardwareVersion = row["hardware_version"]
		update.HardwareVendor = row["hardware_vendor"]
		update.CPUType = row["cpu_type"]
		update.CPUSubtype = row["cpu_subtype"]
		update.CPUBrand = row["cpu_brand"]
		update.CPULogicalCores = parseInt(row["cpu_logical_cores"])
		update.CPUPhysicalCores = parseInt(row["cpu_physical_cores"])
		update.PhysicalMemory = parseInt64(row["physical_memory"])
	}
	if row := details["osquery_info"]; row != nil {
		update.OsqueryVersion = row["version"]
	}
	if row := details["orbit_info"]; row != nil {
		update.OrbitVersion = row["version"]
	}
	if row := details["os_version"]; row != nil {
		update.OSName = row["name"]
		update.OSVersion = osVersion(row)
		update.OSBuild = row["build"]
	}
	if row := details["platform_info"]; row != nil {
		update.KernelVersion = row["extra"]
	}
	if row := details["uptime"]; row != nil {
		if seconds := parsePositiveInt64Ptr(row["total_seconds"]); seconds != nil {
			restarted := time.Now().Add(-time.Duration(*seconds) * time.Second)
			update.LastRestartedAt = &restarted
		}
	}
	if row := details["root_disk"]; row != nil {
		total := parseInt64(row["bytes_total"])
		available := parseInt64(row["bytes_available"])
		if total > 0 {
			update.DiskSpaceTotalBytes = new(total)
			if available >= 0 {
				update.DiskSpaceAvailableBytes = new(available)
			}
		}
	}
	if row := details["primary_interface"]; row != nil {
		update.PrimaryIP = row["primary_ip"]
		update.PrimaryMAC = row["primary_mac"]
	}
	return update
}

func osVersion(row map[string]string) string {
	name := row["name"]
	version := row["version"]
	if version == "" {
		version = dottedVersion(row)
	}
	build := row["build"]
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
		if value := row[key]; value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, ".")
}

func parseInt(value string) int {
	parsed, _ := strconv.Atoi(value)
	return parsed
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
