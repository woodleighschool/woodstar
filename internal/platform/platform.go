package platform

import "strings"

type Platform string

const (
	PlatformDarwin  Platform = "darwin"
	PlatformWindows Platform = "windows"
	PlatformLinux   Platform = "linux"
	PlatformChrome  Platform = "chrome"
)

var canonicalPlatforms = map[Platform]struct{}{
	PlatformDarwin:  {},
	PlatformWindows: {},
	PlatformLinux:   {},
	PlatformChrome:  {},
}

func CleanPtr(value *string) *string {
	if value == nil {
		return nil
	}
	cleaned := CleanPlatform(*value)
	if cleaned == "" {
		return nil
	}
	return new(cleaned)
}

func CleanPlatform(value string) string {
	platform := Platform(strings.ToLower(strings.TrimSpace(value)))
	if _, ok := canonicalPlatforms[platform]; ok {
		return string(platform)
	}
	return ""
}

func Matches(selector string, hostPlatform string) bool {
	hostPlatform = strings.ToLower(strings.TrimSpace(hostPlatform))
	for item := range strings.SplitSeq(selector, ",") {
		platform := Platform(strings.ToLower(strings.TrimSpace(item)))
		if platform == "" {
			continue
		}
		if string(platform) == hostPlatform {
			return true
		}
		if platform == PlatformDarwin && hostPlatform == "macos" {
			return true
		}
		if platform == PlatformLinux && isLinuxPlatform(hostPlatform) {
			return true
		}
	}
	return strings.TrimSpace(selector) == ""
}

func isLinuxPlatform(platform string) bool {
	switch platform {
	case "", "darwin", "macos", "windows", "chrome":
		return false
	default:
		return true
	}
}
