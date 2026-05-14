package directory

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

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

// EntraClient fetches directory users and groups from Microsoft Graph.
type EntraClient struct {
	cfg  EntraConfig
	http *http.Client
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
		cfg:  cfg,
		http: creds.Client(context.Background()),
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

	for i := range users {
		ids, err := c.fetchUserGroupIDs(ctx, users[i].ExternalID)
		if err != nil {
			return Snapshot{}, fmt.Errorf("fetch groups for %s: %w", users[i].UserPrincipalName, err)
		}
		users[i].GroupExternalIDs = ids
	}

	return Snapshot{Users: users, Groups: groups, GeneratedAt: now}, nil
}

type graphUser struct {
	ID                string  `json:"id"`
	UserPrincipalName string  `json:"userPrincipalName"`
	Mail              *string `json:"mail"`
	MailNickname      *string `json:"mailNickname"`
	DisplayName       string  `json:"displayName"`
	GivenName         *string `json:"givenName"`
	Surname           *string `json:"surname"`
	Department        *string `json:"department"`
	AccountEnabled    *bool   `json:"accountEnabled"`
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

type graphGroup struct {
	ID            string  `json:"id"`
	DisplayName   string  `json:"displayName"`
	MailNickname  *string `json:"mailNickname"`
	ODataType     string  `json:"@odata.type"`
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

func (c *EntraClient) fetchUserGroupIDs(ctx context.Context, userID string) ([]string, error) {
	relation := "memberOf"
	if c.cfg.TransitiveGroups {
		relation = "transitiveMemberOf"
	}
	endpoint := fmt.Sprintf(
		"https://graph.microsoft.com/v1.0/users/%s/%s/microsoft.graph.group?$select=id&$top=999",
		url.PathEscape(userID), relation,
	)
	var out []string
	for endpoint != "" {
		var page struct {
			NextLink string       `json:"@odata.nextLink"`
			Value    []graphGroup `json:"value"`
		}
		if err := c.get(ctx, endpoint, &page); err != nil {
			return nil, err
		}
		for _, g := range page.Value {
			out = append(out, g.ID)
		}
		endpoint = page.NextLink
	}
	return out, nil
}

func (c *EntraClient) get(ctx context.Context, endpoint string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
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

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
