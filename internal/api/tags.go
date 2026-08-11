package api

import "github.com/danielgtaylor/huma/v2"

// OpenAPI resource tags are shared by capability operations and API metadata.
const (
	TagAccount                 = "Account"
	TagAgentSecrets            = "Agent secrets"
	TagDirectoryGroups         = "Directory groups"
	TagDirectorySync           = "Directory sync"
	TagDirectoryUsers          = "Directory users"
	TagHosts                   = "Hosts"
	TagLabels                  = "Labels"
	TagMunkiClientResources    = "Munki client resources"
	TagMunkiDistributionPoints = "Munki distribution points"
	TagMunkiIcons              = "Munki icons"
	TagMunkiPackageInstallers  = "Munki package installers"
	TagMunkiPackages           = "Munki packages"
	TagMunkiSoftware           = "Munki software"
	TagOsqueryChecks           = "Osquery checks"
	TagOsqueryLiveQueries      = "Osquery live queries"
	TagOsqueryReports          = "Osquery reports"
	TagSantaConfigurations     = "Santa configurations"
	TagSantaEvents             = "Santa events"
	TagSantaRules              = "Santa rules"
	TagSession                 = "Session"
	TagSoftware                = "Software"
)

type openAPITagGroup struct {
	Name string   `json:"name" yaml:"name"`
	Tags []string `json:"tags" yaml:"tags"`
}

// configureOpenAPI declares the resource hierarchy used by API documentation.
func configureOpenAPI(doc *huma.OpenAPI) {
	doc.Tags = []*huma.Tag{
		resourceTag(TagAccount, "Account"),
		resourceTag(TagAgentSecrets, "Agent secrets"),
		resourceTag(TagDirectoryGroups, "Groups"),
		resourceTag(TagDirectorySync, "Sync"),
		resourceTag(TagDirectoryUsers, "Users"),
		resourceTag(TagHosts, "Hosts"),
		resourceTag(TagLabels, "Labels"),
		resourceTag(TagMunkiClientResources, "Client resources"),
		resourceTag(TagMunkiDistributionPoints, "Distribution points"),
		resourceTag(TagMunkiIcons, "Icons"),
		resourceTag(TagMunkiPackageInstallers, "Package installers"),
		resourceTag(TagMunkiPackages, "Packages"),
		resourceTag(TagMunkiSoftware, "Software"),
		resourceTag(TagOsqueryChecks, "Checks"),
		resourceTag(TagOsqueryLiveQueries, "Live queries"),
		resourceTag(TagOsqueryReports, "Reports"),
		resourceTag(TagSantaConfigurations, "Configurations"),
		resourceTag(TagSantaEvents, "Events"),
		resourceTag(TagSantaRules, "Rules"),
		resourceTag(TagSession, "Session"),
		resourceTag(TagSoftware, "Software"),
	}
	doc.Extensions = map[string]any{
		"x-tagGroups": []openAPITagGroup{
			{Name: "Account", Tags: []string{TagAccount}},
			{Name: "Agent secrets", Tags: []string{TagAgentSecrets}},
			{Name: "Directory", Tags: []string{TagDirectorySync, TagDirectoryGroups, TagDirectoryUsers}},
			{Name: "Hosts", Tags: []string{TagHosts}},
			{Name: "Labels", Tags: []string{TagLabels}},
			{Name: "Munki", Tags: []string{
				TagMunkiClientResources,
				TagMunkiDistributionPoints,
				TagMunkiIcons,
				TagMunkiPackageInstallers,
				TagMunkiPackages,
				TagMunkiSoftware,
			}},
			{Name: "Osquery", Tags: []string{
				TagOsqueryChecks,
				TagOsqueryLiveQueries,
				TagOsqueryReports,
			}},
			{Name: "Santa", Tags: []string{
				TagSantaConfigurations,
				TagSantaEvents,
				TagSantaRules,
			}},
			{Name: "Session", Tags: []string{TagSession}},
			{Name: "Software", Tags: []string{TagSoftware}},
		},
	}
}

func resourceTag(name string, displayName string) *huma.Tag {
	return &huma.Tag{
		Name:       name,
		Extensions: map[string]any{"x-displayName": displayName},
	}
}
