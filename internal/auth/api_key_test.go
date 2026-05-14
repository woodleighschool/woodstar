package auth

import "testing"

func TestBearerToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		header    string
		wantToken string
		wantOK    bool
	}{
		{name: "valid", header: "Bearer abc.def", wantToken: "abc.def", wantOK: true},
		{name: "case insensitive scheme", header: "bearer abc", wantToken: "abc", wantOK: true},
		{name: "extra whitespace", header: "Bearer   abc  ", wantToken: "abc", wantOK: true},
		{name: "empty", header: "", wantOK: false},
		{name: "no scheme", header: "abc", wantOK: false},
		{name: "wrong scheme", header: "Basic abc", wantOK: false},
		{name: "scheme only", header: "Bearer ", wantOK: false},
		{name: "scheme without space", header: "Bearerabc", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := bearerToken(tt.header)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.wantToken {
				t.Fatalf("token = %q, want %q", got, tt.wantToken)
			}
		})
	}
}

func TestGenerateAPIKey(t *testing.T) {
	t.Parallel()
	a, err := generateAPIKey()
	if err != nil {
		t.Fatalf("generateAPIKey returned error: %v", err)
	}
	b, err := generateAPIKey()
	if err != nil {
		t.Fatalf("generateAPIKey returned error: %v", err)
	}
	if a == b {
		t.Fatalf("two consecutive keys collided: %q", a)
	}
	if len(a) < 32 {
		t.Fatalf("key length = %d, want >= 32 (24 random bytes base64-encoded)", len(a))
	}
}
