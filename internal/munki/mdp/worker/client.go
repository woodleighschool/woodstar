package worker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
)

// woodstarClient downloads mirrored installer bytes from Woodstar over HTTP. The
// control channel (desired set in, state out) runs over the WebSocket instead.
type woodstarClient struct {
	serverURL string
	key       string
	http      *http.Client
}

func newWoodstarClient(serverURL, key string) *woodstarClient {
	return &woodstarClient{serverURL: serverURL, key: key, http: &http.Client{}}
}

// download streams a package installer to path. Woodstar redirects to the
// storage backend, which the HTTP client follows; the worker never holds storage
// credentials.
func (c *woodstarClient) download(ctx context.Context, packageID int64, path string) error {
	url := fmt.Sprintf("%s/api/munki/distribution/packages/%d/content", c.serverURL, packageID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.key)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download package %d: unexpected status %d", packageID, resp.StatusCode)
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, resp.Body); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
