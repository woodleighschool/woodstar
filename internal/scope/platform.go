package scope

import (
	"slices"
	"strings"
)

// Platform is the small target family set.
type Platform string

const (
	PlatformUnknown Platform = "unknown"
	PlatformDarwin  Platform = "darwin"
	PlatformWindows Platform = "windows"
	PlatformLinux   Platform = "linux"
)

var linuxHostPlatforms = []string{
	"linux",
	"ubuntu",
	"debian",
	"rhel",
	"centos",
	"sles",
	"kali",
	"gentoo",
	"amzn",
	"pop",
	"arch",
	"linuxmint",
	"void",
	"nixos",
	"endeavouros",
	"manjaro",
	"manjaro-arm",
	"opensuse-leap",
	"opensuse-tumbleweed",
	"tuxedo",
	"neon",
	"archarm",
	"flatcar",
	"coreos",
}

// PlatformFromOsquery squashes osquery names into our small set.
func PlatformFromOsquery(platform, platformLike string) Platform {
	switch normalizedPlatform(platform) {
	case "darwin", "macos":
		return PlatformDarwin
	case "windows":
		return PlatformWindows
	case "":
	default:
		if isLinuxPlatform(platform) {
			return PlatformLinux
		}
	}
	if isLinuxPlatform(platformLike) {
		return PlatformLinux
	}
	return PlatformUnknown
}

func isLinuxPlatform(platform string) bool {
	return slices.Contains(linuxHostPlatforms, normalizedPlatform(platform))
}

func normalizedPlatform(platform string) string {
	return strings.ToLower(strings.TrimSpace(platform))
}
