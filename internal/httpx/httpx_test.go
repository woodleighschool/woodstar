package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
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

func TestPathParamDecodesRequestPathOnce(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		target string
		want   string
	}{
		{
			name:   "noncanonical raw path",
			target: "/files/Zoom-7.1.5%20(84650).pkg",
			want:   "Zoom-7.1.5 (84650).pkg",
		},
		{
			name:   "canonical escaped path",
			target: "/files/Microsoft%20365.pkg",
			want:   "Microsoft 365.pkg",
		},
		{
			name:   "literal percent sequence",
			target: "/files/Literal%2520Name.pkg",
			want:   "Literal%20Name.pkg",
		},
		{
			name:   "URL delimiters",
			target: "/files/Hash%23Question%3FPercent%25.pkg",
			want:   "Hash#Question?Percent%.pkg",
		},
		{
			name:   "Unicode",
			target: "/files/Caf%C3%A9%20%E2%9C%A8.pkg",
			want:   "Café ✨.pkg",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var got string
			router := chi.NewRouter()
			router.Get("/files/*", func(_ http.ResponseWriter, r *http.Request) {
				got = PathParam(r, "*")
			})

			router.ServeHTTP(
				httptest.NewRecorder(),
				httptest.NewRequestWithContext(t.Context(), http.MethodGet, tc.target, nil),
			)

			if got != tc.want {
				t.Fatalf("PathParam() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestEscapePathEncodesSegmentsWithoutEncodingSeparators(t *testing.T) {
	t.Parallel()

	const logical = "packages/38/installer/Zoom #1? 100% Café.pkg"
	const want = "packages/38/installer/Zoom%20%231%3F%20100%25%20Caf%C3%A9.pkg"
	if got := EscapePath(logical); got != want {
		t.Fatalf("EscapePath() = %q, want %q", got, want)
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
