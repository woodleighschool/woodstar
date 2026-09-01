package entra

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/microsoft/kiota-abstractions-go/authentication"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"

	"github.com/woodleighschool/woodstar/internal/directory"
)

func TestClientFetchUsesGraphPagingAndBatchMembership(t *testing.T) {
	const baseURL = "https://graph.test/v1.0"
	transport := &graphFetchTransport{t: t, baseURL: baseURL}
	httpClient := &http.Client{Transport: transport}
	client := newTestClient(t, httpClient, baseURL, false)

	snapshot, err := client.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
	if transport.batchCalls != 2 {
		t.Fatalf("batch calls = %d, want 2 for membership pagination", transport.batchCalls)
	}

	wantUsers := []directory.ProviderUser{
		{
			ExternalID:        "u-1",
			UserPrincipalName: "one@example.invalid",
			Mail:              "one@example.invalid",
			MailNickname:      "one",
			DisplayName:       "One",
			GivenName:         "User",
			FamilyName:        "One",
			Department:        "IT",
			Enabled:           true,
			GroupExternalIDs:  []string{"g-1", "g-3"},
		},
		{
			ExternalID:        "u-2",
			UserPrincipalName: "two@example.invalid",
			DisplayName:       "Two",
			Enabled:           false,
			GroupExternalIDs:  []string{"g-2"},
		},
	}
	if !reflect.DeepEqual(snapshot.Users, wantUsers) {
		t.Fatalf("users = %#v, want %#v", snapshot.Users, wantUsers)
	}
	wantGroups := []directory.ProviderGroup{
		{ExternalID: "g-1", DisplayName: "Group 1", MailNickname: "group-1"},
		{ExternalID: "g-2", DisplayName: "Group 2"},
		{ExternalID: "g-3", DisplayName: "Group 3"},
	}
	if !reflect.DeepEqual(snapshot.Groups, wantGroups) {
		t.Fatalf("groups = %#v, want %#v", snapshot.Groups, wantGroups)
	}
}

type graphFetchTransport struct {
	t          *testing.T
	baseURL    string
	batchCalls int
}

func (f *graphFetchTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch req.URL.Path {
	case "/v1.0/users":
		return f.users(req), nil
	case "/v1.0/groups":
		return f.groups(req), nil
	case "/v1.0/$batch":
		return f.batch(req), nil
	default:
		return nil, errors.New("unexpected Graph request: " + req.URL.String())
	}
}

func (f *graphFetchTransport) users(req *http.Request) *http.Response {
	if req.URL.Query().Get("$skiptoken") == "users-2" {
		return jsonResponse(map[string]any{
			"value": []map[string]any{{
				"id":                "u-2",
				"userPrincipalName": "two@example.invalid",
				"displayName":       "Two",
				"accountEnabled":    false,
			}},
		})
	}
	checkSelectedFields(f.t, req, []string{
		"id",
		"userPrincipalName",
		"mail",
		"mailNickname",
		"displayName",
		"givenName",
		"surname",
		"department",
		"accountEnabled",
	})
	return jsonResponse(map[string]any{
		"@odata.nextLink": f.baseURL + "/users?$select=id,userPrincipalName,mail,mailNickname,displayName,givenName,surname,department,accountEnabled&$top=999&$skiptoken=users-2",
		"value": []map[string]any{{
			"id":                "u-1",
			"userPrincipalName": "one@example.invalid",
			"mail":              "one@example.invalid",
			"mailNickname":      "one",
			"displayName":       "One",
			"givenName":         "User",
			"surname":           "One",
			"department":        "IT",
		}},
	})
}

func (f *graphFetchTransport) groups(req *http.Request) *http.Response {
	if req.URL.Query().Get("$skiptoken") == "groups-2" {
		return jsonResponse(map[string]any{
			"value": []map[string]any{
				{"id": "g-2", "displayName": "Group 2"},
				{"id": "g-3", "displayName": "Group 3"},
			},
		})
	}
	checkSelectedFields(f.t, req, []string{"id", "displayName", "mailNickname"})
	return jsonResponse(map[string]any{
		"@odata.nextLink": f.baseURL + "/groups?$select=id,displayName,mailNickname&$top=999&$skiptoken=groups-2",
		"value": []map[string]any{{
			"id":           "g-1",
			"displayName":  "Group 1",
			"mailNickname": "group-1",
		}},
	})
}

func (f *graphFetchTransport) batch(req *http.Request) *http.Response {
	f.batchCalls++
	var body testBatchRequestBody
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		f.t.Fatalf("decode batch request: %v", err)
	}
	responses := make([]map[string]any, 0, len(body.Requests))
	for _, request := range body.Requests {
		requestURL, err := url.QueryUnescape(request.URL)
		if err != nil {
			f.t.Fatalf("decode batch URL %q: %v", request.URL, err)
		}
		responses = append([]map[string]any{{
			"id":      request.ID,
			"status":  http.StatusOK,
			"headers": map[string]string{"Content-Type": "application/json"},
			"body":    f.membershipResponse(requestURL),
		}}, responses...)
	}
	return jsonResponse(map[string]any{"responses": responses})
}

func (f *graphFetchTransport) membershipResponse(requestURL string) map[string]any {
	switch {
	case strings.Contains(requestURL, "/users/u-1/memberOf/graph.group") &&
		strings.Contains(requestURL, "$skiptoken=membership-2"):
		return map[string]any{"value": []map[string]any{{"id": "g-3"}}}
	case strings.Contains(requestURL, "/users/u-1/memberOf/graph.group"):
		return map[string]any{
			"@odata.nextLink": f.baseURL + "/users/u-1/memberOf/graph.group?$select=id&$top=999&$skiptoken=membership-2",
			"value":           []map[string]any{{"id": "g-1"}},
		}
	case strings.Contains(requestURL, "/users/u-2/memberOf/graph.group"):
		return map[string]any{"value": []map[string]any{{"id": "g-2"}}}
	default:
		f.t.Fatalf("unexpected membership request URL %q", requestURL)
		return nil
	}
}

func TestClientFetchUsesTransitiveGroupRelationship(t *testing.T) {
	const baseURL = "https://graph.test/v1.0"
	var membershipURL string
	httpClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/v1.0/users":
			return jsonResponse(map[string]any{
				"value": []map[string]any{{
					"id":                "u-1",
					"userPrincipalName": "one@example.invalid",
					"displayName":       "One",
				}},
			}), nil
		case "/v1.0/groups":
			return jsonResponse(map[string]any{"value": []map[string]any{}}), nil
		case "/v1.0/$batch":
			var body testBatchRequestBody
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatalf("decode batch request: %v", err)
			}
			if len(body.Requests) != 1 {
				t.Fatalf("batch request count = %d, want 1", len(body.Requests))
			}
			membershipURL, _ = url.QueryUnescape(body.Requests[0].URL)
			return jsonResponse(map[string]any{
				"responses": []map[string]any{{
					"id":      body.Requests[0].ID,
					"status":  http.StatusOK,
					"headers": map[string]string{"Content-Type": "application/json"},
					"body":    map[string]any{"value": []map[string]any{}},
				}},
			}), nil
		default:
			return nil, errors.New("unexpected Graph request: " + req.URL.String())
		}
	})}
	client := newTestClient(t, httpClient, baseURL, true)

	if _, err := client.Fetch(context.Background()); err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
	if !strings.Contains(membershipURL, "/users/u-1/transitiveMemberOf/graph.group") {
		t.Fatalf("membership URL = %q, want transitiveMemberOf group cast", membershipURL)
	}
}

func newTestClient(
	t *testing.T,
	httpClient *http.Client,
	baseURL string,
	transitiveGroups bool,
) *Client {
	t.Helper()
	adapter, err := msgraphsdk.NewGraphRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClient(
		&authentication.AnonymousAuthenticationProvider{},
		nil,
		nil,
		httpClient,
	)
	if err != nil {
		t.Fatalf("create Graph request adapter: %v", err)
	}
	adapter.SetBaseUrl(baseURL)
	return newClient(msgraphsdk.NewGraphServiceClient(adapter), transitiveGroups)
}

func checkSelectedFields(t *testing.T, req *http.Request, want []string) {
	t.Helper()
	got := strings.Split(req.URL.Query().Get("$select"), ",")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("$select = %#v, want %#v", got, want)
	}
}

type testBatchRequestBody struct {
	Requests []struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"requests"`
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(body any) *http.Response {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		panic(err)
	}
	return &http.Response{
		StatusCode:    http.StatusOK,
		Status:        "200 OK",
		Header:        http.Header{"Content-Type": []string{"application/json"}},
		Body:          io.NopCloser(&buf),
		ContentLength: int64(buf.Len()),
	}
}
