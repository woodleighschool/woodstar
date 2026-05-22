package platforms

import (
	"errors"
	"fmt"
	"strings"
)

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

func CleanTargets(values []Platform) ([]Platform, error) {
	if len(values) == 0 {
		return nil, errors.New("platforms are required")
	}
	seen := make(map[Platform]struct{}, len(values))
	out := make([]Platform, 0, len(values))
	for _, value := range values {
		platform := clean(value)
		if _, ok := canonicalPlatforms[platform]; !ok {
			return nil, fmt.Errorf("unknown platform %q", value)
		}
		if _, ok := seen[platform]; ok {
			continue
		}
		seen[platform] = struct{}{}
		out = append(out, platform)
	}
	if len(out) == 0 {
		return nil, errors.New("platforms are required")
	}
	return out, nil
}

func CleanPlatform(value string) string {
	platform := clean(Platform(value))
	if _, ok := canonicalPlatforms[platform]; ok {
		return string(platform)
	}
	return ""
}

func clean(value Platform) Platform {
	return Platform(strings.ToLower(strings.TrimSpace(string(value))))
}
