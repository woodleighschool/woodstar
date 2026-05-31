package munki

import (
	"context"
	"errors"
	"strings"

	"howett.net/plist"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

// ErrNotFound reports that a requested Munki repository object does not exist.
var ErrNotFound = errors.New("munki resource not found")

type hostResolver interface {
	GetByHardwareSerial(context.Context, string) (*hosts.Host, error)
}

// ClientHost identifies the existing Woodstar host making a Munki request.
type ClientHost struct {
	ID          int64
	Serial      string
	DisplayName string
}

// Service renders the Munki client-facing repository surface.
type Service struct {
	hosts hostResolver
}

// NewService returns the default Munki repository renderer.
func NewService(hosts hostResolver) *Service {
	return &Service{hosts: hosts}
}

// ResolveClient resolves the Munki request identity to an existing host.
func (s *Service) ResolveClient(ctx context.Context, serial string) (ClientHost, error) {
	if s.hosts == nil {
		return ClientHost{}, ErrNotFound
	}
	host, err := s.hosts.GetByHardwareSerial(ctx, strings.TrimSpace(serial))
	if errors.Is(err, dbutil.ErrNotFound) {
		return ClientHost{}, ErrNotFound
	}
	if err != nil {
		return ClientHost{}, err
	}
	return ClientHost{
		ID:          host.ID,
		Serial:      host.Hardware.Serial,
		DisplayName: host.DisplayName,
	}, nil
}

// Manifest returns a Munki manifest plist for name.
func (s *Service) Manifest(_ context.Context, client ClientHost, name string) ([]byte, error) {
	if !validResourceName(name) || name != client.Serial {
		return nil, ErrNotFound
	}
	displayName := client.DisplayName
	if displayName == "" {
		displayName = client.Serial
	}
	return encodePlist(renderedManifest{
		Catalogs:          []string{"production"},
		DisplayName:       displayName,
		ManagedInstalls:   []string{},
		ManagedUninstalls: []string{},
		ManagedUpdates:    []string{},
		OptionalInstalls:  []string{},
	})
}

// Catalog returns a Munki catalog plist for name.
func (s *Service) Catalog(_ context.Context, _ ClientHost, name string) ([]byte, error) {
	if name != "production" || !validResourceName(name) {
		return nil, ErrNotFound
	}
	return encodePlist([]renderedPkginfo{})
}

func validResourceName(name string) bool {
	name = strings.TrimSpace(name)
	return name != "" && !strings.ContainsAny(name, `/\`)
}

func encodePlist(value any) ([]byte, error) {
	return plist.Marshal(value, plist.XMLFormat)
}

type renderedManifest struct {
	Catalogs          []string `plist:"catalogs"`
	DisplayName       string   `plist:"display_name"`
	ManagedInstalls   []string `plist:"managed_installs"`
	ManagedUninstalls []string `plist:"managed_uninstalls"`
	ManagedUpdates    []string `plist:"managed_updates"`
	OptionalInstalls  []string `plist:"optional_installs"`
}

type renderedPkginfo struct{}
