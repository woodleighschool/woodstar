// Package rbac owns the application's resource catalogue and role policy.
package rbac

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/woodleighschool/goodies/auth/authz"

	"github.com/woodleighschool/woodstar/internal/openapischema"
)

// Application authorization resources.
const (
	ResourceActivity                authz.Resource = "activity"
	ResourceUsers                   authz.Resource = "users"
	ResourceGroups                  authz.Resource = "groups"
	ResourceDirectory               authz.Resource = "directory"
	ResourceHosts                   authz.Resource = "hosts"
	ResourceLabels                  authz.Resource = "labels"
	ResourceSoftware                authz.Resource = "software"
	ResourceAgentSecrets            authz.Resource = "agents.secrets"
	ResourceMunkiSoftware           authz.Resource = "munki.software"
	ResourceMunkiPackages           authz.Resource = "munki.packages"
	ResourceMunkiDistributionPoints authz.Resource = "munki.distribution-points"
	ResourceMunkiClientResources    authz.Resource = "munki.client-resources"
	ResourceOsqueryOverview         authz.Resource = "osquery.overview"
	ResourceOsqueryReports          authz.Resource = "osquery.reports"
	ResourceOsqueryPolicies         authz.Resource = "osquery.policies"
	ResourceOsqueryLiveQueries      authz.Resource = "osquery.live-queries"
	ResourceOsqueryRemediations     authz.Resource = "osquery.remediations"
	ResourceSantaConfigurations     authz.Resource = "santa.configurations"
	ResourceSantaEvents             authz.Resource = "santa.events"
	ResourceSantaRules              authz.Resource = "santa.rules"
)

// Resources returns the application's resource catalogue.
func Resources() []authz.Resource {
	return []authz.Resource{
		ResourceActivity,
		ResourceUsers,
		ResourceGroups,
		ResourceDirectory,
		ResourceHosts,
		ResourceLabels,
		ResourceSoftware,
		ResourceAgentSecrets,
		ResourceMunkiSoftware,
		ResourceMunkiPackages,
		ResourceMunkiDistributionPoints,
		ResourceMunkiClientResources,
		ResourceOsqueryOverview,
		ResourceOsqueryReports,
		ResourceOsqueryPolicies,
		ResourceOsqueryLiveQueries,
		ResourceOsqueryRemediations,
		ResourceSantaConfigurations,
		ResourceSantaEvents,
		ResourceSantaRules,
	}
}

// ResourceSchema describes the application catalogue for API clients.
func ResourceSchema() *huma.Schema { return openapischema.StringEnum(Resources()...) }
