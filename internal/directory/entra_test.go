package directory

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

func TestEntraClientTokenFetchUsesRequestContext(t *testing.T) {
	tokenCalled := make(chan struct{})
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(tokenCalled)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"Bearer","expires_in":3600}`))
	}))
	defer tokenServer.Close()

	client := &EntraClient{
		creds: clientcredentials.Config{
			ClientID:     "client",
			ClientSecret: "secret",
			TokenURL:     tokenServer.URL,
			Scopes:       []string{"https://graph.microsoft.com/.default"},
		},
		http: tokenServer.Client(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var out struct{}
	err := client.get(ctx, "https://graph.microsoft.com/v1.0/users", &out)
	if err == nil {
		t.Fatal("get returned nil error after token context cancellation")
	}
	if !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("get error = %v, want context cancellation", err)
	}
	select {
	case <-tokenCalled:
		t.Fatal("token endpoint was called after request context cancellation")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestEntraClientFetchBatchesUserGroupMembership(t *testing.T) {
	var batchCalls int
	var directMembershipCalls int
	client := &EntraClient{
		http: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch {
			case req.URL.Path == "/v1.0/users":
				return jsonResponse(http.StatusOK, map[string]any{
					"value": []map[string]any{
						{"id": "u-1", "userPrincipalName": "one@example.com", "displayName": "One"},
						{"id": "u-2", "userPrincipalName": "two@example.com", "displayName": "Two"},
					},
				}), nil
			case req.URL.Path == "/v1.0/groups":
				return jsonResponse(http.StatusOK, map[string]any{
					"value": []map[string]any{
						{"id": "g-1", "displayName": "Group 1"},
						{"id": "g-2", "displayName": "Group 2"},
					},
				}), nil
			case req.URL.Path == "/v1.0/$batch":
				batchCalls++
				var body graphBatchRequestBody
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Fatalf("decode batch request: %v", err)
				}
				if len(body.Requests) != 2 {
					t.Fatalf("batch request count = %d, want 2", len(body.Requests))
				}
				return jsonResponse(http.StatusOK, map[string]any{
					"responses": []map[string]any{
						{
							"id":     body.Requests[1].ID,
							"status": http.StatusOK,
							"body": map[string]any{
								"value": []map[string]any{{"id": "g-2"}},
							},
						},
						{
							"id":     body.Requests[0].ID,
							"status": http.StatusOK,
							"body": map[string]any{
								"value": []map[string]any{{"id": "g-1"}},
							},
						},
					},
				}), nil
			case strings.Contains(req.URL.Path, "/memberOf/"):
				directMembershipCalls++
				return jsonResponse(http.StatusOK, map[string]any{"value": []map[string]any{}}), nil
			default:
				t.Fatalf("unexpected Graph request: %s", req.URL.String())
				return nil, errors.New("unexpected Graph request")
			}
		})},
		token: &oauth2.Token{AccessToken: "token", Expiry: time.Now().Add(time.Hour)},
	}

	snapshot, err := client.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
	if batchCalls != 1 {
		t.Fatalf("batch calls = %d, want 1", batchCalls)
	}
	if directMembershipCalls != 0 {
		t.Fatalf("direct membership calls = %d, want 0", directMembershipCalls)
	}
	if got := snapshot.Users[0].GroupExternalIDs; len(got) != 1 || got[0] != "g-1" {
		t.Fatalf("user 1 groups = %#v, want [g-1]", got)
	}
	if got := snapshot.Users[1].GroupExternalIDs; len(got) != 1 || got[0] != "g-2" {
		t.Fatalf("user 2 groups = %#v, want [g-2]", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(status int, body any) *http.Response {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		panic(err)
	}
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(&buf),
	}
}
