package osquery

import (
	"strconv"
	"strings"
)

const labelQueryPrefix = "woodstar_label_query_"

func labelQueryName(labelID int64) string {
	return labelQueryPrefix + strconv.FormatInt(labelID, 10)
}

func parseLabelQueryName(name string) (int64, bool) {
	raw, ok := strings.CutPrefix(name, labelQueryPrefix)
	if !ok || raw == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}
