package wire

import (
	"encoding/json"
	"testing"
)

func TestParseSubprotocolVersion(t *testing.T) {
	tests := []struct {
		subprotocol string
		wantVersion int
		wantOK      bool
	}{
		{subprotocol: "woodstar-mdp.v0", wantVersion: 0, wantOK: true},
		{subprotocol: Subprotocol, wantVersion: ProtocolVersion, wantOK: true},
		{subprotocol: "woodstar-mdp.v2", wantVersion: 2, wantOK: true},
		{subprotocol: "woodstar-mdp.v", wantOK: false},
		{subprotocol: "other.v1", wantOK: false},
	}
	for _, test := range tests {
		t.Run(test.subprotocol, func(t *testing.T) {
			version, ok := ParseSubprotocolVersion(test.subprotocol)
			if version != test.wantVersion || ok != test.wantOK {
				t.Fatalf(
					"ParseSubprotocolVersion(%q) = (%d, %t), want (%d, %t)",
					test.subprotocol,
					version,
					ok,
					test.wantVersion,
					test.wantOK,
				)
			}
		})
	}
}

func TestServerMessageShapes(t *testing.T) {
	tests := []struct {
		name    string
		message ServerMessage
		want    string
	}{
		{
			name: "hello",
			message: ServerMessage{
				Type:              MessageHello,
				DistributionPoint: PointIdentity{ID: 1, Name: "Melbourne"},
			},
			want: `{"type":"hello","distribution_point":{"id":1,"name":"Melbourne"}}`,
		},
		{
			name: "empty desired set",
			message: ServerMessage{
				Type:     MessageDesiredSet,
				Packages: []DesiredPackage{},
			},
			want: `{"type":"desired_set","packages":[]}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := json.Marshal(test.message)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if got := string(data); got != test.want {
				t.Fatalf("Marshal = %s, want %s", got, test.want)
			}
		})
	}
}
