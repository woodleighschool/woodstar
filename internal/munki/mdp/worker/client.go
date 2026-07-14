package worker

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

const (
	apiRequestTimeout      = 30 * time.Second
	downloadRequestTimeout = time.Hour
)

// woodstarClient talks to Woodstar over HTTPS: it asks for a fresh per-job
// download URL and then streams the installer bytes. The control channel
// (desired set in, package events out) runs over the WebSocket instead.
type woodstarClient struct {
	apiHTTP       *http.Client
	downloadHTTP  *http.Client
	websocketHTTP *http.Client
	serverURL     string
	key           string
}

func newWoodstarClient(serverURL, key, serverCAFile string) (*woodstarClient, error) {
	transport, err := clientTransport(serverCAFile)
	if err != nil {
		return nil, err
	}
	return &woodstarClient{
		apiHTTP:       &http.Client{Transport: transport, Timeout: apiRequestTimeout},
		downloadHTTP:  &http.Client{Transport: transport, Timeout: downloadRequestTimeout},
		websocketHTTP: &http.Client{Transport: transport},
		serverURL:     serverURL,
		key:           key,
	}, nil
}

func clientTransport(serverCAFile string) (*http.Transport, error) {
	defaultTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, errors.New("default HTTP transport is not configurable")
	}
	transport := defaultTransport.Clone()
	if serverCAFile == "" {
		return transport, nil
	}
	pem, err := os.ReadFile(serverCAFile)
	if err != nil {
		return nil, fmt.Errorf("read server CA file: %w", err)
	}
	roots, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("load system CA pool: %w", err)
	}
	if roots == nil {
		return nil, errors.New("system CA pool is unavailable")
	}
	if !roots.AppendCertsFromPEM(pem) {
		return nil, errors.New("server CA file contains no certificates")
	}
	transport.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    roots,
	}
	return transport, nil
}

type downloadURLResponse struct {
	DownloadURL string `json:"download_url"`
}

// downloadURL fetches a short-lived URL for one package's installer bytes,
// authenticating as this distribution point.
func (c *woodstarClient) downloadURL(ctx context.Context, packageID int64) (string, error) {
	endpoint := c.serverURL + "/api/munki/distribution/packages/" +
		strconv.FormatInt(packageID, 10) + "/download-url"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.key)

	resp, err := c.apiHTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download url: unexpected status %d", resp.StatusCode)
	}
	var body downloadURLResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("download url: decode response: %w", err)
	}
	if body.DownloadURL == "" {
		return "", errors.New("download url: empty response")
	}
	return body.DownloadURL, nil
}

// download streams a package installer from a presigned storage URL to path.
func (c *woodstarClient) download(ctx context.Context, downloadURL string, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return err
	}

	resp, err := c.downloadHTTP.Do(req)
	if err != nil {
		return sanitizedRequestError("download", downloadURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download: unexpected status %d", resp.StatusCode)
	}

	return writeBody(path, resp.Body)
}

func sanitizedRequestError(operation, rawURL string, err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		err = urlErr.Err
	}
	parsed, parseErr := url.Parse(rawURL)
	if parseErr != nil {
		return fmt.Errorf("%s request failed: %w", operation, err)
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.ForceQuery = false
	parsed.Fragment = ""
	return fmt.Errorf("%s %s: %w", operation, parsed.String(), err)
}

func writeBody(path string, body io.Reader) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, body); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
