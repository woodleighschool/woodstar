package entra

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	abstractions "github.com/microsoft/kiota-abstractions-go"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	graphcore "github.com/microsoftgraph/msgraph-sdk-go-core"
	graphgroups "github.com/microsoftgraph/msgraph-sdk-go/groups"
	graphmodels "github.com/microsoftgraph/msgraph-sdk-go/models"
	graphusers "github.com/microsoftgraph/msgraph-sdk-go/users"

	"github.com/woodleighschool/woodstar/internal/directory"
)

const graphPageSize int32 = 999

var graphScopes = []string{"https://graph.microsoft.com/.default"}

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
	graph            *msgraphsdk.GraphServiceClient
	transitiveGroups bool
}

// NewClient returns a Graph SDK client authenticated with an Entra application.
func NewClient(cfg Config) (*Client, error) {
	credential, err := azidentity.NewClientSecretCredential(
		cfg.TenantID,
		cfg.ClientID,
		cfg.ClientSecret,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create Entra credential: %w", err)
	}
	graph, err := msgraphsdk.NewGraphServiceClientWithCredentials(credential, graphScopes)
	if err != nil {
		return nil, fmt.Errorf("create Microsoft Graph client: %w", err)
	}
	return newClient(graph, cfg.TransitiveGroups), nil
}

func newClient(graph *msgraphsdk.GraphServiceClient, transitiveGroups bool) *Client {
	return &Client{graph: graph, transitiveGroups: transitiveGroups}
}

// Fetch builds a directory snapshot from Graph. It pages through users and
// groups, then resolves each user's direct or transitive group membership.
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
	page, err := c.graph.Users().Get(ctx, &graphusers.UsersRequestBuilderGetRequestConfiguration{
		QueryParameters: &graphusers.UsersRequestBuilderGetQueryParameters{
			Select: []string{
				"id",
				"userPrincipalName",
				"mail",
				"mailNickname",
				"displayName",
				"givenName",
				"surname",
				"department",
				"accountEnabled",
			},
			Top: int32Pointer(graphPageSize),
		},
	})
	if err != nil {
		return nil, err
	}

	iterator, err := graphcore.NewPageIterator[graphmodels.Userable](
		page,
		c.graph.GetAdapter(),
		graphmodels.CreateUserCollectionResponseFromDiscriminatorValue,
	)
	if err != nil {
		return nil, err
	}

	var users []directory.ProviderUser
	err = iterator.Iterate(ctx, func(user graphmodels.Userable) bool {
		users = append(users, directory.ProviderUser{
			ExternalID:        deref(user.GetId()),
			UserPrincipalName: deref(user.GetUserPrincipalName()),
			Mail:              deref(user.GetMail()),
			MailNickname:      deref(user.GetMailNickname()),
			DisplayName:       deref(user.GetDisplayName()),
			GivenName:         deref(user.GetGivenName()),
			FamilyName:        deref(user.GetSurname()),
			Department:        deref(user.GetDepartment()),
			Enabled:           enabled(user.GetAccountEnabled()),
		})
		return true
	})
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (c *Client) fetchGroups(ctx context.Context) ([]directory.ProviderGroup, error) {
	page, err := c.graph.Groups().Get(ctx, &graphgroups.GroupsRequestBuilderGetRequestConfiguration{
		QueryParameters: &graphgroups.GroupsRequestBuilderGetQueryParameters{
			Select: []string{"id", "displayName", "mailNickname"},
			Top:    int32Pointer(graphPageSize),
		},
	})
	if err != nil {
		return nil, err
	}

	iterator, err := graphcore.NewPageIterator[graphmodels.Groupable](
		page,
		c.graph.GetAdapter(),
		graphmodels.CreateGroupCollectionResponseFromDiscriminatorValue,
	)
	if err != nil {
		return nil, err
	}

	var groups []directory.ProviderGroup
	err = iterator.Iterate(ctx, func(group graphmodels.Groupable) bool {
		groups = append(groups, directory.ProviderGroup{
			ExternalID:   deref(group.GetId()),
			DisplayName:  deref(group.GetDisplayName()),
			MailNickname: deref(group.GetMailNickname()),
		})
		return true
	})
	if err != nil {
		return nil, err
	}
	return groups, nil
}

func (c *Client) fetchUsersGroupIDs(
	ctx context.Context,
	users []directory.ProviderUser,
) (map[string][]string, error) {
	out := make(map[string][]string, len(users))
	pending := make([]graphMembershipRequest, 0, len(users))
	for _, user := range users {
		out[user.ExternalID] = nil
		pending = append(pending, graphMembershipRequest{UserID: user.ExternalID})
	}
	for len(pending) > 0 {
		followups, err := c.applyMembershipBatch(ctx, pending, out)
		if err != nil {
			return nil, err
		}
		pending = followups
	}
	return out, nil
}

// applyMembershipBatch lets the Graph SDK split requests at Graph's batch
// limit, records each user's group IDs, and returns any paged follow-ups.
func (c *Client) applyMembershipBatch(
	ctx context.Context,
	requests []graphMembershipRequest,
	out map[string][]string,
) ([]graphMembershipRequest, error) {
	adapter := c.graph.GetAdapter()
	batch := graphcore.NewBatchRequestCollectionWithLimit(adapter, len(requests))
	requestsByID := make(map[string]graphMembershipRequest, len(requests))

	for _, request := range requests {
		requestInfo, err := c.membershipRequestInfo(ctx, request)
		if err != nil {
			return nil, fmt.Errorf("build groups request for %s: %w", request.UserID, err)
		}
		item, err := batch.AddBatchRequestStep(*requestInfo)
		if err != nil {
			return nil, fmt.Errorf("batch groups request for %s: %w", request.UserID, err)
		}
		if item.GetId() == nil {
			return nil, fmt.Errorf("batch groups request for %s: missing request ID", request.UserID)
		}
		requestsByID[*item.GetId()] = request
	}

	response, err := batch.Send(ctx, adapter)
	if err != nil {
		return nil, fmt.Errorf("fetch user groups: %w", err)
	}

	var followups []graphMembershipRequest
	for requestID, request := range requestsByID {
		if response.GetResponseById(requestID) == nil {
			return nil, fmt.Errorf("graph batch missing response for %s", request.UserID)
		}
		page, err := graphcore.GetBatchResponseById[graphmodels.GroupCollectionResponseable](
			response,
			requestID,
			graphmodels.CreateGroupCollectionResponseFromDiscriminatorValue,
		)
		if err != nil {
			return nil, fmt.Errorf("fetch groups for %s: %w", request.UserID, err)
		}
		for _, group := range page.GetValue() {
			out[request.UserID] = append(out[request.UserID], deref(group.GetId()))
		}
		if nextLink := deref(page.GetOdataNextLink()); nextLink != "" {
			followups = append(followups, graphMembershipRequest{
				UserID:   request.UserID,
				NextLink: nextLink,
			})
		}
	}
	return followups, nil
}

func (c *Client) membershipRequestInfo(
	ctx context.Context,
	request graphMembershipRequest,
) (*abstractions.RequestInformation, error) {
	user := c.graph.Users().ByUserId(request.UserID)
	if c.transitiveGroups {
		builder := user.TransitiveMemberOf().GraphGroup()
		if request.NextLink != "" {
			return builder.WithUrl(request.NextLink).ToGetRequestInformation(ctx, nil)
		}
		return builder.ToGetRequestInformation(
			ctx,
			&graphusers.ItemTransitiveMemberOfGraphGroupRequestBuilderGetRequestConfiguration{
				QueryParameters: &graphusers.ItemTransitiveMemberOfGraphGroupRequestBuilderGetQueryParameters{
					Select: []string{"id"},
					Top:    int32Pointer(graphPageSize),
				},
			},
		)
	}

	builder := user.MemberOf().GraphGroup()
	if request.NextLink != "" {
		return builder.WithUrl(request.NextLink).ToGetRequestInformation(ctx, nil)
	}
	return builder.ToGetRequestInformation(
		ctx,
		&graphusers.ItemMemberOfGraphGroupRequestBuilderGetRequestConfiguration{
			QueryParameters: &graphusers.ItemMemberOfGraphGroupRequestBuilderGetQueryParameters{
				Select: []string{"id"},
				Top:    int32Pointer(graphPageSize),
			},
		},
	)
}

type graphMembershipRequest struct {
	UserID   string
	NextLink string
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func enabled(value *bool) bool {
	return value == nil || *value
}

func int32Pointer(value int32) *int32 {
	return new(value)
}
