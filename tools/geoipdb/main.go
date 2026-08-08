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
	release := flag.String("release", time.Now().UTC().Format("2006-01"), "DB-IP release in YYYY-MM format")
	output := flag.String("output", ".cache/geoip", "directory for decompressed MMDB files")
	flag.Parse()

	if err := run(context.Background(), *release, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, release, output string) error {
	if _, err := time.Parse("2006-01", release); err != nil {
		return fmt.Errorf("invalid DB-IP release %q: %w", release, err)
	}
	if err := os.MkdirAll(output, 0o750); err != nil {
		return fmt.Errorf("create GeoIP cache: %w", err)
	}
	if currentRelease(output) == release && databasesExist(output) {
		return nil
	}

	client := &http.Client{Timeout: 10 * time.Minute}
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
