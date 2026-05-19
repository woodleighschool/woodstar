package platform

import "strings"

type Platform string

const (
	PlatformDarwin  Platform = "darwin"
	PlatformWindows Platform = "windows"
	PlatformLinux   Platform = "linux"
)

var canonicalPlatforms = map[Platform]struct{}{
	PlatformDarwin:  {},
	PlatformWindows: {},
	PlatformLinux:   {},
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
