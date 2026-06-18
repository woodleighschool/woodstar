package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
)

// woodstarClient talks to Woodstar over HTTP: it asks for a fresh per-job
// download URL and then streams the installer bytes. The control channel
// (desired set in, package events out) runs over the WebSocket instead.
type woodstarClient struct {
	http      *http.Client
	serverURL string
	key       string
}

func newWoodstarClient(serverURL, key string) *woodstarClient {
	return &woodstarClient{http: &http.Client{}, serverURL: serverURL, key: key}
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

	resp, err := c.http.Do(req)
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

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download: unexpected status %d", resp.StatusCode)
	}

	return writeBody(path, resp.Body)
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
