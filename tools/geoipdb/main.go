package main

import (
	"compress/gzip"
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const releaseMarker = ".release"

var databases = []struct {
	kind     string
	filename string
}{
	{kind: "city", filename: "dbip-city-lite.mmdb"},
	{kind: "asn", filename: "dbip-asn-lite.mmdb"},
}

func main() {
	var release string
	flag.Func("release", "DB-IP release in YYYY-MM format (default: latest published for City and ASN)", func(value string) error {
		if _, err := time.Parse("2006-01", value); err != nil {
			return fmt.Errorf("invalid DB-IP release %q: %w", value, err)
		}
		release = value
		return nil
	})
	output := flag.String("output", ".local/geoip", "directory for decompressed MMDB files")
	flag.Parse()

	client := &http.Client{Timeout: 10 * time.Minute}
	if err := run(context.Background(), client, release, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, client *http.Client, release, output string) error {
	if release == "" {
		var err error
		release, err = latestRelease(ctx, client)
		if err != nil {
			return err
		}
	}
	if _, err := time.Parse("2006-01", release); err != nil {
		return fmt.Errorf("invalid DB-IP release %q: %w", release, err)
	}
	if err := os.MkdirAll(output, 0o750); err != nil {
		return fmt.Errorf("create GeoIP cache: %w", err)
	}
	if currentRelease(output) == release && databasesExist(output) {
		return nil
	}

	for _, database := range databases {
		if err := download(ctx, client, release, output, database.kind, database.filename); err != nil {
			return err
		}
	}
	if err := writeRelease(output, release); err != nil {
		return err
	}
	return nil
}

func latestRelease(ctx context.Context, client *http.Client) (string, error) {
	city, err := publishedReleases(ctx, client, "city")
	if err != nil {
		return "", err
	}
	asn, err := publishedReleases(ctx, client, "asn")
	if err != nil {
		return "", err
	}
	var latest string
	for release := range city {
		if asn[release] && release > latest {
			latest = release
		}
	}
	if latest == "" {
		return "", fmt.Errorf("DB-IP Lite: no common published City and ASN MMDB release")
	}
	return latest, nil
}

func publishedReleases(ctx context.Context, client *http.Client, kind string) (map[string]bool, error) {
	url := "https://db-ip.com/db/download/ip-to-" + kind + "-lite"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create DB-IP %s release request: %w", kind, err)
	}
	request.Header.Set("User-Agent", "Woodstar GeoIP database downloader")
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("discover DB-IP %s releases: %w", kind, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discover DB-IP %s releases: HTTP %s", kind, response.Status)
	}
	page, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read DB-IP %s releases: %w", kind, err)
	}
	// Match only the official MMDB download links, not CSV releases or page dates.
	links := regexp.MustCompile(`href\s*=\s*["']https://download\.db-ip\.com/free/dbip-` + kind + `-lite-([0-9]{4}-[0-9]{2})\.mmdb\.gz["']`)
	releases := make(map[string]bool)
	for _, match := range links.FindAllSubmatch(page, -1) {
		release := string(match[1])
		if _, err := time.Parse("2006-01", release); err == nil {
			releases[release] = true
		}
	}
	return releases, nil
}

func currentRelease(output string) string {
	//nolint:gosec // The caller intentionally selects this build-tool output directory.
	contents, err := os.ReadFile(filepath.Join(output, releaseMarker))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(contents))
}

func databasesExist(output string) bool {
	for _, database := range databases {
		info, err := os.Stat(filepath.Join(output, database.filename))
		if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
			return false
		}
	}
	return true
}

func download(
	ctx context.Context,
	client *http.Client,
	release string,
	output string,
	kind string,
	filename string,
) error {
	url := fmt.Sprintf(
		"https://download.db-ip.com/free/dbip-%s-lite-%s.mmdb.gz",
		kind,
		release,
	)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create DB-IP %s request: %w", kind, err)
	}
	request.Header.Set("User-Agent", "Woodstar GeoIP database downloader")
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("download DB-IP %s: %w", kind, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, response.Body)
		return fmt.Errorf("download DB-IP %s: HTTP %s", kind, response.Status)
	}

	compressed, err := gzip.NewReader(response.Body)
	if err != nil {
		return fmt.Errorf("open DB-IP %s archive: %w", kind, err)
	}
	defer func() { _ = compressed.Close() }()

	temporary, err := os.CreateTemp(output, "."+filename+"-*")
	if err != nil {
		return fmt.Errorf("create temporary DB-IP %s database: %w", kind, err)
	}
	temporaryName := temporary.Name()
	installed := false
	defer func() {
		_ = temporary.Close()
		if !installed {
			_ = os.Remove(temporaryName)
		}
	}()

	//nolint:gosec // DB-IP is the trusted database source for this build tool.
	if _, err := io.Copy(temporary, compressed); err != nil {
		return fmt.Errorf("decompress DB-IP %s: %w", kind, err)
	}
	if err := compressed.Close(); err != nil {
		return fmt.Errorf("close DB-IP %s archive: %w", kind, err)
	}
	if err := temporary.Chmod(0o644); err != nil {
		return fmt.Errorf("set DB-IP %s permissions: %w", kind, err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close DB-IP %s database: %w", kind, err)
	}
	if err := os.Rename(temporaryName, filepath.Join(output, filename)); err != nil {
		return fmt.Errorf("install DB-IP %s database: %w", kind, err)
	}
	installed = true
	return nil
}

func writeRelease(output, release string) error {
	temporary, err := os.CreateTemp(output, ".release-*")
	if err != nil {
		return fmt.Errorf("create DB-IP release marker: %w", err)
	}
	temporaryName := temporary.Name()
	installed := false
	defer func() {
		_ = temporary.Close()
		if !installed {
			_ = os.Remove(temporaryName)
		}
	}()
	if _, err := temporary.WriteString(release + "\n"); err != nil {
		return fmt.Errorf("write DB-IP release marker: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close DB-IP release marker: %w", err)
	}
	if err := os.Rename(temporaryName, filepath.Join(output, releaseMarker)); err != nil {
		return fmt.Errorf("install DB-IP release marker: %w", err)
	}
	installed = true
	return nil
}
