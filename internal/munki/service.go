package munki

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strings"
)

// ErrNotFound reports that a requested Munki repository object does not exist.
var ErrNotFound = errors.New("munki resource not found")

// Service renders the Munki client-facing repository surface.
type Service struct{}

// NewService returns the default Munki repository renderer.
func NewService() *Service {
	return &Service{}
}

// Manifest returns a Munki manifest plist for name.
func (s *Service) Manifest(_ context.Context, name string) ([]byte, error) {
	if !validResourceName(name) {
		return nil, ErrNotFound
	}
	return fmt.Appendf(nil, manifestTemplate, html.EscapeString(name)), nil
}

// Catalog returns a Munki catalog plist for name.
func (s *Service) Catalog(_ context.Context, name string) ([]byte, error) {
	if !validResourceName(name) {
		return nil, ErrNotFound
	}
	return []byte(catalogTemplate), nil
}

func validResourceName(name string) bool {
	name = strings.TrimSpace(name)
	return name != "" && !strings.ContainsAny(name, `/\`)
}

const manifestTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>catalogs</key>
	<array>
		<string>production</string>
	</array>
	<key>display_name</key>
	<string>%s</string>
	<key>managed_installs</key>
	<array/>
	<key>managed_uninstalls</key>
	<array/>
	<key>managed_updates</key>
	<array/>
	<key>optional_installs</key>
	<array/>
</dict>
</plist>
`

const catalogTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<array/>
</plist>
`
