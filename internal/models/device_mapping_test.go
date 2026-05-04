package models

import "testing"

func TestDeviceMappingSources(t *testing.T) {
	if DeviceMappingSourceOrbitProfile != "orbit_profile" {
		t.Fatalf("orbit profile source = %q", DeviceMappingSourceOrbitProfile)
	}
}
