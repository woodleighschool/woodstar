package integration

import (
	"os"
	"strings"
	"unicode"
)

const integrationRequiredEnvironment = "WOODSTAR_INTEGRATION_REQUIRED"

func integrationRequired(target string) bool {
	for _, name := range strings.FieldsFunc(os.Getenv(integrationRequiredEnvironment), func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	}) {
		if name == target || name == "all" {
			return true
		}
	}
	return false
}
