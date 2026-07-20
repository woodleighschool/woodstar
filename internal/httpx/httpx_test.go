package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBearerToken(t *testing.T) {
	tests := []struct {
		name          string
		authorization string
		want          string
		wantOK        bool
	}{
		{name: "missing"},
		{name: "wrong scheme", authorization: "Token abc"},
		{name: "empty bearer", authorization: "Bearer "},
		{name: "spaces in token", authorization: "Bearer abc def"},
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

func TestDecodeRejectsTrailingJSON(t *testing.T) {
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(`{"value":1}{"value":2}`))
	_, err := Decode[struct {
		Value int `json:"value"`
	}](httptest.NewRecorder(), req, 1024)
	if err == nil {
		t.Fatal("Decode returned nil error for multiple JSON values")
	}
}

func TestDecodeReportsOversizedBody(t *testing.T) {
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(`{"value":"too large"}`))
	rec := httptest.NewRecorder()
	_, err := Decode[struct {
		Value string `json:"value"`
	}](rec, req, 8)
	if err == nil {
		t.Fatal("Decode returned nil error for oversized body")
	}

	WriteDecodeError(rec, err)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusRequestEntityTooLarge, rec.Body.String())
	}
}
