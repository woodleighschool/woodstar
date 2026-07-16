package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func requestJSON(
	t *testing.T,
	client *http.Client,
	method string,
	requestURL string,
	body any,
	wantStatus int,
	target any,
) *http.Response {
	t.Helper()

	var requestBody io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("encode %s %s JSON request: %v", method, requestURL, err)
		}
		requestBody = bytes.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(t.Context(), method, requestURL, requestBody)
	if err != nil {
		t.Fatalf("create %s %s request: %v", method, requestURL, err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("send %s %s request: %v", method, requestURL, err)
	}
	responseBody := readAndClose(t, response)
	if response.StatusCode != wantStatus {
		t.Fatalf("%s %s status = %d, want %d", method, requestURL, response.StatusCode, wantStatus)
	}
	if target != nil {
		if err := json.Unmarshal(responseBody, target); err != nil {
			t.Fatalf("decode %s %s JSON response: %v", method, requestURL, err)
		}
	}
	return response
}

func readAndClose(t *testing.T, response *http.Response) []byte {
	t.Helper()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		_ = response.Body.Close()
		t.Fatalf("read HTTP response: %v", err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatalf("close HTTP response: %v", err)
	}
	return body
}
