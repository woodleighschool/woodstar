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

	"github.com/woodleighschool/woodstar/internal/directory"
)

// Config holds the credentials needed to call Microsoft Graph as an
// application.
type Config struct {
	TenantID         string
	ClientID         string
	ClientSecret     string
	TransitiveGroups bool
}

// Client fetches Entra users and groups from Microsoft Graph.
type Client struct {
	cfg   Config
	creds clientcredentials.Config
	http  *http.Client
	mu    sync.Mutex
	token *oauth2.Token
}

// NewClient returns a Graph client that signs requests with an application token.
func NewClient(cfg Config) *Client {
	creds := clientcredentials.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		TokenURL:     fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", cfg.TenantID),
		Scopes:       []string{"https://graph.microsoft.com/.default"},
	}
	return &Client{
		cfg:   cfg,
		creds: creds,
		http:  http.DefaultClient,
	}
}

// Fetch builds a directory snapshot from Graph. It pages through /users and /groups,
// then resolves each user's group membership (memberOf or
// transitiveMemberOf per config) filtered to microsoft.graph.group.
func (c *Client) Fetch(ctx context.Context) (directory.ProviderSnapshot, error) {
	now := time.Now().UTC()

	users, err := c.fetchUsers(ctx)
	if err != nil {
		return directory.ProviderSnapshot{}, fmt.Errorf("fetch users: %w", err)
	}
	groups, err := c.fetchGroups(ctx)
	if err != nil {
		return directory.ProviderSnapshot{}, fmt.Errorf("fetch groups: %w", err)
	}

	groupIDsByUser, err := c.fetchUsersGroupIDs(ctx, users)
	if err != nil {
		return directory.ProviderSnapshot{}, err
	}
	for i := range users {
		users[i].GroupExternalIDs = groupIDsByUser[users[i].ExternalID]
	}

	return directory.ProviderSnapshot{Users: users, Groups: groups, GeneratedAt: now}, nil
}

func (c *Client) fetchUsers(ctx context.Context) ([]directory.ProviderUser, error) {
	endpoint := "https://graph.microsoft.com/v1.0/users?$select=id,userPrincipalName,mail,mailNickname,displayName,givenName,surname,department,accountEnabled&$top=999"
	var out []directory.ProviderUser
	for endpoint != "" {
		var page struct {
			NextLink string      `json:"@odata.nextLink"`
			Value    []graphUser `json:"value"`
		}
		if err := c.get(ctx, endpoint, &page); err != nil {
			return nil, err
		}
		for _, u := range page.Value {
			out = append(out, directory.ProviderUser{
				ExternalID:        u.ID,
				UserPrincipalName: u.UserPrincipalName,
				Mail:              deref(u.Mail),
				MailNickname:      deref(u.MailNickname),
				DisplayName:       u.DisplayName,
				GivenName:         deref(u.GivenName),
				FamilyName:        deref(u.Surname),
				Department:        deref(u.Department),
				Enabled:           u.AccountEnabled == nil || *u.AccountEnabled,
			})
		}
		endpoint = page.NextLink
	}
	return out, nil
}

func (c *Client) fetchGroups(ctx context.Context) ([]directory.ProviderGroup, error) {
	endpoint := "https://graph.microsoft.com/v1.0/groups?$select=id,displayName,mailNickname&$top=999"
	var out []directory.ProviderGroup
	for endpoint != "" {
		var page struct {
			NextLink string       `json:"@odata.nextLink"`
			Value    []graphGroup `json:"value"`
		}
		if err := c.get(ctx, endpoint, &page); err != nil {
			return nil, err
		}
		for _, g := range page.Value {
			out = append(out, directory.ProviderGroup{
				ExternalID:   g.ID,
				DisplayName:  g.DisplayName,
				MailNickname: deref(g.MailNickname),
			})
		}
		endpoint = page.NextLink
	}
	return out, nil
}

func (c *Client) fetchUsersGroupIDs(
	ctx context.Context,
	users []directory.ProviderUser,
) (map[string][]string, error) {
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

		followups, err := c.applyMembershipBatch(ctx, batch, out)
		if err != nil {
			return nil, err
		}
		pending = append(pending, followups...)
	}
	return out, nil
}

// applyMembershipBatch sends one batch, records each user's group IDs into out,
// and returns follow-up requests for any paged responses.
func (c *Client) applyMembershipBatch(
	ctx context.Context,
	batch []graphMembershipRequest,
	out map[string][]string,
) ([]graphMembershipRequest, error) {
	responses, err := c.fetchMembershipBatch(ctx, batch)
	if err != nil {
		return nil, err
	}
	var followups []graphMembershipRequest
	for _, request := range batch {
		response, ok := responses[request.ID]
		if !ok {
			return nil, fmt.Errorf("graph batch missing response for %s", request.UserID)
		}
		next, err := parseMembershipResponse(request, response, out)
		if err != nil {
			return nil, err
		}
		followups = append(followups, next...)
	}
	return followups, nil
}

// parseMembershipResponse validates one batch response, appends its group IDs
// to out, and returns any follow-up request when the response is paged.
func parseMembershipResponse(
	request graphMembershipRequest,
	response graphBatchResponse,
	out map[string][]string,
) ([]graphMembershipRequest, error) {
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
	if page.NextLink == "" {
		return nil, nil
	}
	nextURL, err := graphBatchRelativeURL(page.NextLink)
	if err != nil {
		return nil, fmt.Errorf("fetch groups for %s: %w", request.UserID, err)
	}
	return []graphMembershipRequest{{UserID: request.UserID, URL: nextURL}}, nil
}

func (c *Client) userGroupMembershipURL(userID string) string {
	relation := "memberOf"
	if c.cfg.TransitiveGroups {
		relation = "transitiveMemberOf"
	}
	return fmt.Sprintf(
		"/users/%s/%s/microsoft.graph.group?$select=id&$top=999",
		url.PathEscape(userID), relation,
	)
}

func (c *Client) fetchMembershipBatch(
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

func (c *Client) get(ctx context.Context, endpoint string, out any) error {
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

func (c *Client) post(ctx context.Context, endpoint string, body any, out any) error {
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

func (c *Client) authToken(ctx context.Context) (*oauth2.Token, error) {
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
