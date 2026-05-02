package buildinfo

import (
	"encoding/json"
	"fmt"
	"runtime"
)

var (
	Version   = "0.0.0-dev"
	Commit    = ""
	Date      = ""
	UserAgent = fmt.Sprintf("woodstar/%s (%s %s)", Version, runtime.GOOS, runtime.GOARCH)
)

type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

func Current() Info {
	return Info{
		Version: Version,
		Commit:  Commit,
		Date:    Date,
	}
}

func String() string {
	return fmt.Sprintf("Version: %s\nCommit: %s\nBuild date: %s\n", Version, Commit, Date)
}

func JSON() ([]byte, error) {
	return json.Marshal(Current())
}
