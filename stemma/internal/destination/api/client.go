// Package api calls the administrative API that holds Munki software, packages
// and their content.
package api

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
	"strings"
	"time"

	"resty.dev/v3"
)

// Config holds the destination's connection settings.
type Config struct {
	URL    string `json:"url" jsonschema:"pattern=^https://" jsonschema_description:"HTTPS origin of the administrative API, without a path, query or credentials."`
	APIKey string `json:"api_key" jsonschema:"minLength=1,writeOnly=true" jsonschema_description:"API key with software and package management access. Supply it through an environment reference."`
	CAFile string `json:"ca_file,omitempty" jsonschema_description:"PEM certificate file to trust in addition to the system certificate authorities."`
}

// Validate requires an HTTPS origin and a key that fits in a header.
func (cfg Config) Validate() error {
	parsed, err := url.Parse(cfg.URL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return errors.New("url must be an HTTPS origin without credentials, query or path")
	}
	if strings.TrimSpace(cfg.APIKey) == "" || strings.ContainsAny(cfg.APIKey, "\r\n\x00") {
		return errors.New("api_key is required")
	}
	return nil
}

// Client authenticates API requests with the key. Content goes to the signed
// targets the API issues, which never receive the key.
type Client struct {
	api      *resty.Client
	transfer *resty.Client
}

// New connects to the configured origin, trusting a configured CA file in
// addition to the system roots.
func New(cfg Config) (*Client, error) {
	defaultTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, errors.New("default HTTP transport is not configurable")
	}
	transport := defaultTransport.Clone()
	if cfg.CAFile != "" {
		pem, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, errors.New("read configured CA certificate")
		}
		roots, err := x509.SystemCertPool()
		if err != nil {
			roots = x509.NewCertPool()
		}
		if !roots.AppendCertsFromPEM(pem) {
			return nil, errors.New("configured CA file contains no certificates")
		}
		transport.TLSClientConfig = &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}
	}
	newHTTPClient := func() *resty.Client {
		return resty.NewWithClient(&http.Client{Transport: transport}).
			SetRedirectPolicy(resty.RedirectNoPolicy()).
			SetRetryCount(2).
			AddRetryConditions(resty.RetryConditionStatusTooManyRequests, resty.RetryConditionStatus5XX, resty.RetryConditionStatusZero).
			SetResponseBodyLimit(8 << 20)
	}
	api := newHTTPClient().SetBaseURL(strings.TrimRight(cfg.URL, "/")).SetAuthToken(cfg.APIKey).
		SetHeader("Accept", "application/json").
		AddContentTypeDecoder("json", func(reader io.Reader, value any) error {
			decoder := json.NewDecoder(reader)
			decoder.UseNumber()
			return decoder.Decode(value)
		})
	return &Client{api: api, transfer: newHTTPClient().SetResponseBodyLimit(1 << 20)}, nil
}

// Close releases the connections of both HTTP clients.
func (c *Client) Close() error {
	return errors.Join(c.api.Close(), c.transfer.Close())
}

func (c *Client) request(ctx context.Context, method, endpoint string, body, output any) error {
	// Finalization reads full stored installers, so API calls share the transfer budget.
	ctx, cancel := context.WithTimeout(ctx, time.Hour)
	defer cancel()
	request := c.api.R().SetContext(ctx).SetResult(output).SetResponseForceContentType("application/json")
	if body != nil {
		request.SetHeader("Content-Type", "application/json").SetBody(body)
	}
	response, err := request.Execute(method, endpoint)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("%s %s: request failed", method, endpoint)
	}
	if !response.IsStatusSuccess() {
		return StatusError{Method: method, Path: endpoint, Status: response.StatusCode()}
	}
	return nil
}

func list[T any](ctx context.Context, c *Client, endpoint string, query url.Values) ([]T, error) {
	var items []T
	query.Set("per_page", "1000")
	for page := 1; page <= 10000; page++ {
		query.Set("page", strconv.Itoa(page))
		var response struct {
			Items []T `json:"items"`
			Count int `json:"count"`
		}
		if err := c.request(ctx, http.MethodGet, endpoint+"?"+query.Encode(), nil, &response); err != nil {
			return nil, err
		}
		items = append(items, response.Items...)
		if len(items) >= response.Count {
			return items, nil
		}
		if len(response.Items) == 0 {
			return nil, errors.New("incomplete discovery page")
		}
	}
	return nil, errors.New("discovery exceeds page limit")
}

// StatusError is an API reply outside the success range.
type StatusError struct {
	Method string
	Path   string
	Status int
}

func (err StatusError) Error() string {
	return fmt.Sprintf("%s %s: HTTP %d", err.Method, err.Path, err.Status)
}
