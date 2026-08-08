// Package geoip enriches public IP addresses from bundled DB-IP databases.
package geoip

import (
	"errors"
	"fmt"
	"math"
	"net/netip"
	"os"
	"path/filepath"
	"strings"

	geoip2 "github.com/oschwald/geoip2-golang/v2"
)

const (
	// DirectoryEnvironment overrides the packaged GeoIP directory for local development.
	DirectoryEnvironment = "WOODSTAR_GEOIP_DIR"
	// CityFilename is the packaged DB-IP City Lite database filename.
	CityFilename = "dbip-city-lite.mmdb"
	// ASNFilename is the packaged DB-IP ASN Lite database filename.
	ASNFilename = "dbip-asn-lite.mmdb"
)

// Result is a complete location and network description for a public IP.
type Result struct {
	CountryCode  string
	Country      string
	RegionCode   string
	Region       string
	City         string
	Latitude     float64
	Longitude    float64
	ASN          uint32
	Organization string
}

// Reader owns the City Lite and ASN Lite MMDB readers.
type Reader struct {
	city *geoip2.Reader
	asn  *geoip2.Reader
}

// Open opens and validates both databases in dir.
func Open(dir string) (*Reader, error) {
	city, err := geoip2.Open(filepath.Join(dir, CityFilename))
	if err != nil {
		if city != nil {
			_ = city.Close()
		}
		return nil, fmt.Errorf("open DB-IP City Lite: %w", err)
	}
	asn, err := geoip2.Open(filepath.Join(dir, ASNFilename))
	if err != nil {
		_ = city.Close()
		if asn != nil {
			_ = asn.Close()
		}
		return nil, fmt.Errorf("open DB-IP ASN Lite: %w", err)
	}
	return &Reader{city: city, asn: asn}, nil
}

// OpenDefault opens the packaged databases, or the directory selected by Mise
// for local development and tests.
func OpenDefault() (*Reader, error) {
	dir := strings.TrimSpace(os.Getenv(DirectoryEnvironment))
	if dir == "" {
		dir = "/share/geoip"
	}
	return Open(dir)
}

// Lookup returns data only when both databases provide every field exposed by
// the host API. Missing or partial records are deliberately indistinguishable.
func (r *Reader) Lookup(addr netip.Addr) (*Result, error) {
	addr = addr.Unmap()
	if !shouldLookup(addr) {
		return nil, nil
	}
	city, err := r.city.City(addr)
	if err != nil {
		return nil, fmt.Errorf("look up DB-IP city: %w", err)
	}
	asn, err := r.asn.ASN(addr)
	if err != nil {
		return nil, fmt.Errorf("look up DB-IP ASN: %w", err)
	}
	return completeResult(city, asn), nil
}

// Close releases both memory-mapped databases.
func (r *Reader) Close() error {
	return errors.Join(r.city.Close(), r.asn.Close())
}

func shouldLookup(addr netip.Addr) bool {
	addr = addr.Unmap()
	return addr.IsValid() && addr.IsGlobalUnicast() && !addr.IsPrivate()
}

func completeResult(city *geoip2.City, asn *geoip2.ASN) *Result {
	if city == nil || asn == nil || !city.HasData() || !asn.HasData() {
		return nil
	}
	if len(city.Subdivisions) == 0 || city.Location.Latitude == nil || city.Location.Longitude == nil {
		return nil
	}
	subdivision := city.Subdivisions[0]
	if city.City.Names.English == "" ||
		subdivision.ISOCode == "" || subdivision.Names.English == "" ||
		city.Country.ISOCode == "" || city.Country.Names.English == "" ||
		asn.AutonomousSystemNumber == 0 ||
		uint64(asn.AutonomousSystemNumber) > math.MaxUint32 ||
		asn.AutonomousSystemOrganization == "" {
		return nil
	}
	return &Result{
		CountryCode:  city.Country.ISOCode,
		Country:      city.Country.Names.English,
		RegionCode:   subdivision.ISOCode,
		Region:       subdivision.Names.English,
		City:         city.City.Names.English,
		Latitude:     *city.Location.Latitude,
		Longitude:    *city.Location.Longitude,
		ASN:          uint32(asn.AutonomousSystemNumber),
		Organization: asn.AutonomousSystemOrganization,
	}
}
