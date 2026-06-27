package httpx

import "testing"

func TestBearerToken(t *testing.T) {
	tests := []struct {
		name          string
		authorization string
		want          string
		wantOK        bool
	}{
		{name: "missing", wantOK: false},
		{name: "wrong scheme", authorization: "Token abc", wantOK: false},
		{name: "empty bearer", authorization: "Bearer ", wantOK: false},
		{name: "spaces in token", authorization: "Bearer abc def", wantOK: false},
		{name: "valid", authorization: "Bearer abc", want: "abc", wantOK: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := BearerToken(tt.authorization)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("BearerToken() = %q, %v; want %q, %v", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
