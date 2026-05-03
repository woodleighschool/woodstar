package buildinfo

import (
	"fmt"
	"runtime"
)

// Build metadata is filled by linker flags at build time.
var (
	Version   = "0.0.0-dev"
	Commit    = ""
	Date      = ""
	UserAgent = fmt.Sprintf("woodstar/%s (%s %s)", Version, runtime.GOOS, runtime.GOARCH)
)

// String returns the build metadata in a human-readable format.
func String() string {
	return fmt.Sprintf("Version: %s\nCommit: %s\nBuild date: %s\n", Version, Commit, Date)
}
