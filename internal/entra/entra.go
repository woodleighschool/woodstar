package entra

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// EntraConfig holds the credentials needed to call Microsoft Graph as an
// application (client-credentials flow).
type EntraConfig struct {
	TenantID         string
	ClientID         string
	ClientSecret     string
	TransitiveGroups bool
}

// EntraClient fetches Entra users and groups from Microsoft Graph.
type EntraClient struct {
	cfg   EntraConfig
	creds clientcredentials.Config
	http  *http.Client
	mu    sync.Mutex
	token *oauth2.Token
}

// NewEntraClient returns a Graph client that signs requests with an
// application token from the v2.0 token endpoint. Tokens are refreshed
// automatically by the underlying oauth2 transport.
func NewEntraClient(cfg EntraConfig) *EntraClient {
	creds := clientcredentials.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		TokenURL:     fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", cfg.TenantID),
		Scopes:       []string{"https://graph.microsoft.com/.default"},
	}
	return &EntraClient{
		cfg:   cfg,
		creds: creds,
		http:  http.DefaultClient,
	}
}

// Fetch builds a Snapshot from Graph. It pages through /users and /groups,
// then resolves each user's group membership (memberOf or
// transitiveMemberOf per config) filtered to microsoft.graph.group.
func (c *EntraClient) Fetch(ctx context.Context) (Snapshot, error) {
	now := time.Now().UTC()

	users, err := c.fetchUsers(ctx)
	if err != nil {
		return Snapshot{}, fmt.Errorf("fetch users: %w", err)
	}
	groups, err := c.fetchGroups(ctx)
	if err != nil {
		return Snapshot{}, fmt.Errorf("fetch groups: %w", err)
	}

	groupIDsByUser, err := c.fetchUsersGroupIDs(ctx, users)
	if err != nil {
		return Snapshot{}, err
	}
	for i := range users {
		users[i].GroupExternalIDs = groupIDsByUser[users[i].ExternalID]
	}

	return Snapshot{Users: users, Groups: groups, GeneratedAt: now}, nil
}

func (c *EntraClient) fetchUsers(ctx context.Context) ([]SnapshotUser, error) {
	endpoint := "https://graph.microsoft.com/v1.0/users?$select=id,userPrincipalName,mail,mailNickname,displayName,givenName,surname,department,accountEnabled&$top=999"
	var out []SnapshotUser
	for endpoint != "" {
		var page struct {
			NextLink string      `json:"@odata.nextLink"`
			Value    []graphUser `json:"value"`
		}
		if err := c.get(ctx, endpoint, &page); err != nil {
			return nil, err
		}
		for _, u := range page.Value {
			out = append(out, SnapshotUser{
				ExternalID:        u.ID,
				UserPrincipalName: u.UserPrincipalName,
				Mail:              deref(u.Mail),
				MailNickname:      deref(u.MailNickname),
				DisplayName:       u.DisplayName,
				GivenName:         deref(u.GivenName),
				FamilyName:        deref(u.Surname),
				Department:        deref(u.Department),
				Active:            u.AccountEnabled == nil || *u.AccountEnabled,
			})
		}
		endpoint = page.NextLink
	}
	return out, nil
}

func (c *EntraClient) fetchGroups(ctx context.Context) ([]SnapshotGroup, error) {
	endpoint := "https://graph.microsoft.com/v1.0/groups?$select=id,displayName,mailNickname&$top=999"
	var out []SnapshotGroup
	for endpoint != "" {
		var page struct {
			NextLink string       `json:"@odata.nextLink"`
			Value    []graphGroup `json:"value"`
		}
		if err := c.get(ctx, endpoint, &page); err != nil {
			return nil, err
		}
		for _, g := range page.Value {
			out = append(out, SnapshotGroup{
				ExternalID:   g.ID,
				DisplayName:  g.DisplayName,
				MailNickname: deref(g.MailNickname),
			})
		}
		endpoint = page.NextLink
	}
	return out, nil
}

func (c *EntraClient) fetchUsersGroupIDs(ctx context.Context, users []SnapshotUser) (map[string][]string, error) {
	out := make(map[string][]string, len(users))
	pending := make([]graphMembershipRequest, 0, len(users))
	for _, user := range users {
		out[user.ExternalID] = nil
		pending = append(pending, graphMembershipRequest{
			UserID: user.ExternalID,
			URL:    c.userGroupMembershipURL(user.ExternalID),
		})
	}
	for len(pending) > 0 {
		size := min(len(pending), graphBatchMaxRequests)
		batch := pending[:size]
		pending = pending[size:]

		responses, err := c.fetchMembershipBatch(ctx, batch)
		if err != nil {
			return nil, err
		}
		for _, request := range batch {
			response, ok := responses[request.ID]
			if !ok {
				return nil, fmt.Errorf("graph batch missing response for %s", request.UserID)
			}
			if response.Status >= 300 {
				return nil, fmt.Errorf("fetch groups for %s: graph batch status %d", request.UserID, response.Status)
			}
			var page graphGroupPage
			if err := json.Unmarshal(response.Body, &page); err != nil {
				return nil, fmt.Errorf("fetch groups for %s: %w", request.UserID, err)
			}
			for _, group := range page.Value {
				out[request.UserID] = append(out[request.UserID], group.ID)
			}
			if page.NextLink != "" {
				nextURL, err := graphBatchRelativeURL(page.NextLink)
				if err != nil {
					return nil, fmt.Errorf("fetch groups for %s: %w", request.UserID, err)
				}
				pending = append(pending, graphMembershipRequest{
					UserID: request.UserID,
					URL:    nextURL,
				})
			}
		}
	}
	return out, nil
}

func (c *EntraClient) userGroupMembershipURL(userID string) string {
	relation := "memberOf"
	if c.cfg.TransitiveGroups {
		relation = "transitiveMemberOf"
	}
	return fmt.Sprintf(
		"/users/%s/%s/microsoft.graph.group?$select=id&$top=999",
		url.PathEscape(userID), relation,
	)
}

func (c *EntraClient) fetchMembershipBatch(
	ctx context.Context,
	requests []graphMembershipRequest,
) (map[string]graphBatchResponse, error) {
	body := graphBatchRequestBody{Requests: make([]graphBatchRequest, len(requests))}
	for i := range requests {
		requests[i].ID = strconv.Itoa(i + 1)
		body.Requests[i] = graphBatchRequest{
			ID:     requests[i].ID,
			Method: http.MethodGet,
			URL:    requests[i].URL,
		}
	}
	var batch graphBatchResponseBody
	if err := c.post(ctx, "https://graph.microsoft.com/v1.0/$batch", body, &batch); err != nil {
		return nil, err
	}
	out := make(map[string]graphBatchResponse, len(batch.Responses))
	for _, response := range batch.Responses {
		out[response.ID] = response
	}
	return out, nil
}

func graphBatchRelativeURL(endpoint string) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	if !parsed.IsAbs() {
		if strings.HasPrefix(endpoint, "/") {
			return endpoint, nil
		}
		return "/" + endpoint, nil
	}
	path := parsed.EscapedPath()
	path = strings.TrimPrefix(path, "/v1.0")
	path = strings.TrimPrefix(path, "/beta")
	if path == "" {
		path = "/"
	}
	if parsed.RawQuery != "" {
		path += "?" + parsed.RawQuery
	}
	return path, nil
}

func (c *EntraClient) get(ctx context.Context, endpoint string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	token, err := c.authToken(ctx)
	if err != nil {
		return err
	}
	token.SetAuthHeader(req)
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("graph %s: %s", endpoint, res.Status)
	}
	return json.NewDecoder(res.Body).Decode(out)
}

func (c *EntraClient) post(ctx context.Context, endpoint string, body any, out any) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	token, err := c.authToken(ctx)
	if err != nil {
		return err
	}
	token.SetAuthHeader(req)
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("graph %s: %s", endpoint, res.Status)
	}
	return json.NewDecoder(res.Body).Decode(out)
}

func (c *EntraClient) authToken(ctx context.Context) (*oauth2.Token, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token.Valid() {
		return c.token, nil
	}
	token, err := c.creds.TokenSource(ctx).Token()
	if err != nil {
		return nil, err
	}
	c.token = token
	return token, nil
}

const graphBatchMaxRequests = 20

type graphMembershipRequest struct {
	ID     string
	UserID string
	URL    string
}

type graphBatchRequestBody struct {
	Requests []graphBatchRequest `json:"requests"`
}

type graphBatchRequest struct {
	ID     string `json:"id"`
	Method string `json:"method"`
	URL    string `json:"url"`
}

type graphBatchResponseBody struct {
	Responses []graphBatchResponse `json:"responses"`
}

type graphBatchResponse struct {
	ID     string          `json:"id"`
	Status int             `json:"status"`
	Body   json.RawMessage `json:"body"`
}

type graphGroupPage struct {
	NextLink string       `json:"@odata.nextLink"`
	Value    []graphGroup `json:"value"`
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
