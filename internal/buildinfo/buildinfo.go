package buildinfo

import (
	"fmt"
	"runtime"
)

const Version = "0.0.0"

var UserAgent = fmt.Sprintf("woodstar/%s (%s %s)", Version, runtime.GOOS, runtime.GOARCH)
