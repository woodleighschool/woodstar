package geoip

import (
	"net/netip"
	"os"
	"testing"

	geoip2 "github.com/oschwald/geoip2-golang/v2"
)

func TestOpenDatabases(t *testing.T) {
	cityFile := os.Getenv("WOODSTAR_GEOIP_CITY_FILE")
	asnFile := os.Getenv("WOODSTAR_GEOIP_ASN_FILE")
	if cityFile == "" || asnFile == "" {
		t.Skip("GeoIP database files are not configured")
	}
	reader, err := Open(cityFile, asnFile)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestShouldLookup(t *testing.T) {
	t.Parallel()
	tests := []struct {
		address string
		want    bool
	}{
		{address: "1.1.1.1", want: true},
		{address: "::ffff:1.1.1.1", want: true},
		{address: "10.0.0.1", want: false},
		{address: "127.0.0.1", want: false},
		{address: "::1", want: false},
	}
	for _, test := range tests {
		t.Run(test.address, func(t *testing.T) {
			t.Parallel()
			if got := shouldLookup(netip.MustParseAddr(test.address)); got != test.want {
				t.Fatalf("shouldLookup(%s) = %t, want %t", test.address, got, test.want)
			}
		})
	}
}

func TestCompleteResultRequiresEveryPublicField(t *testing.T) {
	t.Parallel()
	latitude := -38.15
	longitude := 145.12
	city := &geoip2.City{
		City: geoip2.CityRecord{Names: geoip2.Names{English: "Langwarrin"}},
		Subdivisions: []geoip2.CitySubdivision{{
			Names: geoip2.Names{English: "Victoria"},
		}},
		Country: geoip2.CountryRecord{
			ISOCode: "AU",
			Names:   geoip2.Names{English: "Australia"},
		},
		Location: geoip2.Location{Latitude: &latitude, Longitude: &longitude},
	}
	asn := &geoip2.ASN{
		AutonomousSystemNumber:       1221,
		AutonomousSystemOrganization: "Telstra",
	}

	result := completeResult(city, asn)
	if result == nil {
		t.Fatal("completeResult returned nil for a complete record")
	}
	if result.City != "Langwarrin" || result.Region != "Victoria" || result.ASN != 1221 {
		t.Fatalf("completeResult = %+v", result)
	}

	city.Location.Longitude = nil
	if result := completeResult(city, asn); result != nil {
		t.Fatalf("completeResult with missing longitude = %+v, want nil", result)
	}
}
