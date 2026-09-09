package destination

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"resty.dev/v3"
)

type client struct {
	config   config
	api      *resty.Client
	transfer *resty.Client
}

func newClient(cfg config) (*client, error) {
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
	return &client{config: cfg, api: api, transfer: newHTTPClient().SetResponseBodyLimit(1 << 20)}, nil
}

func (remote *client) request(ctx context.Context, method, endpoint string, body, output any) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	request := remote.api.R().SetContext(ctx).SetResult(output).SetResponseForceContentType("application/json")
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
		return httpError{method, endpoint, response.StatusCode()}
	}
	return nil
}

type httpError struct {
	method, path string
	status       int
}

func (err httpError) Error() string {
	return fmt.Sprintf("%s %s: HTTP %d", err.method, err.path, err.status)
}
