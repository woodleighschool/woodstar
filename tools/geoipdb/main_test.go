package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func liteLinks(kind string, releases ...string) string {
	var page strings.Builder
	for _, release := range releases {
		fmt.Fprintf(&page, "<a class='download' href='https://download.db-ip.com/free/dbip-%s-lite-%s.mmdb.gz'>Download MMDB</a>\n", kind, release)
	}
	return page.String()
}

func TestLatestRelease(t *testing.T) {
	for _, tt := range []struct {
		name, city, asn, want string
	}{
		{"double quoted links", strings.ReplaceAll(liteLinks("city", "2026-10"), "'", "\""), strings.ReplaceAll(liteLinks("asn", "2026-10"), "'", "\""), "2026-10"},
		{"same month", liteLinks("city", "2026-10"), liteLinks("asn", "2026-10"), "2026-10"},
		{"city ahead", liteLinks("city", "2026-10", "2026-09"), liteLinks("asn", "2026-09"), "2026-09"},
		{"asn ahead", liteLinks("city", "2026-09"), liteLinks("asn", "2026-10", "2026-09"), "2026-09"},
		{"unordered across years", liteLinks("city", "2026-01", "2025-12", "2026-01"), liteLinks("asn", "2025-12", "2026-01"), "2026-01"},
		{"no overlap", liteLinks("city", "2026-10"), liteLinks("asn", "2026-09"), ""},
		{"missing city", "<html>Unavailable</html>", liteLinks("asn", "2026-09"), ""},
		{"invalid month", liteLinks("city", "2026-13"), liteLinks("asn", "2026-13"), ""},
		{"csv only", strings.ReplaceAll(liteLinks("city", "2026-10"), ".mmdb.gz", ".csv.gz"), liteLinks("asn", "2026-10"), ""},
		{"unrelated links", liteLinks("asn", "2026-10") + strings.ReplaceAll(liteLinks("city", "2026-10"), "download.db-ip.com", "example.com"), liteLinks("asn", "2026-10"), ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				var page string
				switch r.URL.String() {
				case "https://db-ip.com/db/download/ip-to-city-lite":
					page = tt.city
				case "https://db-ip.com/db/download/ip-to-asn-lite":
					page = tt.asn
				default:
					t.Fatalf("unexpected request: %s", r.URL)
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(page))}, nil
			})}
			got, err := latestRelease(t.Context(), client)
			if tt.want == "" {
				if err == nil {
					t.Fatalf("got release %q, want discovery error", got)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("latestRelease = %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}

func TestLatestReleaseHTTPError(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusServiceUnavailable, Status: "503 Service Unavailable", Body: io.NopCloser(strings.NewReader("unavailable"))}, nil
	})}
	if _, err := latestRelease(t.Context(), client); err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("want HTTP error, got %v", err)
	}
}

func TestRunPublishedReleaseAndCache(t *testing.T) {
	downloads := 0
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body []byte
		switch r.URL.String() {
		case "https://db-ip.com/db/download/ip-to-city-lite":
			body = []byte(liteLinks("city", "2026-10", "2026-09"))
		case "https://db-ip.com/db/download/ip-to-asn-lite":
			body = []byte(liteLinks("asn", "2026-09"))
		case "https://download.db-ip.com/free/dbip-city-lite-2026-09.mmdb.gz",
			"https://download.db-ip.com/free/dbip-asn-lite-2026-09.mmdb.gz":
			downloads++
			var archive bytes.Buffer
			compressed := gzip.NewWriter(&archive)
			if _, err := compressed.Write([]byte("synthetic mmdb")); err != nil {
				t.Fatal(err)
			}
			if err := compressed.Close(); err != nil {
				t.Fatal(err)
			}
			body = archive.Bytes()
		default:
			t.Fatalf("unexpected request: %s", r.URL)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(body))}, nil
	})}
	output := t.TempDir()
	for range 2 {
		if err := run(t.Context(), client, "", output); err != nil {
			t.Fatal(err)
		}
	}
	if downloads != 2 || currentRelease(output) != "2026-09" {
		t.Fatalf("downloads = %d, release = %q", downloads, currentRelease(output))
	}
	for _, filename := range []string{"dbip-city-lite.mmdb", "dbip-asn-lite.mmdb"} {
		//nolint:gosec // Both filenames are fixed fixtures inside the test temporary directory.
		data, err := os.ReadFile(filepath.Join(output, filename))
		if err != nil || string(data) != "synthetic mmdb" {
			t.Fatalf("%s = %q, %v", filename, data, err)
		}
	}
}

func TestRunExplicitReleaseDoesNotDiscoverOrFallback(t *testing.T) {
	requests := 0
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		if r.URL.String() != "https://download.db-ip.com/free/dbip-city-lite-2025-12.mmdb.gz" {
			t.Fatalf("unexpected request: %s", r.URL)
		}
		return &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found", Body: io.NopCloser(strings.NewReader(""))}, nil
	})}
	output := t.TempDir()
	if err := run(t.Context(), client, "2025-12", output); err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("want pinned release failure, got %v", err)
	}
	if requests != 1 || currentRelease(output) != "" {
		t.Fatalf("requests = %d, release = %q", requests, currentRelease(output))
	}
	if err := run(t.Context(), client, "invalid", output); err == nil || requests != 1 {
		t.Fatalf("invalid release: error = %v, requests = %d", err, requests)
	}
}
