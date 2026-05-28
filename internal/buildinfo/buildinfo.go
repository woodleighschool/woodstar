package buildinfo

import (
	"fmt"
	"runtime"
)

var (
	Version   = "0.0.0-dev"
	Commit    = ""
	Date      = ""
	UserAgent = ""
)

func init() {
	UserAgent = fmt.Sprintf("woodstar/%s (%s %s)", Version, runtime.GOOS, runtime.GOARCH)
}
