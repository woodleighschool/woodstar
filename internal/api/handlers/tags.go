package handlers

import "github.com/danielgtaylor/huma/v2"

const (
	accountTag                 = "Account"
	agentSecretsTag            = "Agent secrets"
	directoryGroupsTag         = "Directory groups"
	directoryUsersTag          = "Directory users"
	hostsTag                   = "Hosts"
	labelsTag                  = "Labels"
	munkiClientResourcesTag    = "Munki client resources"
	munkiDistributionPointsTag = "Munki distribution points"
	munkiIconsTag              = "Munki icons"
	munkiPackageInstallersTag  = "Munki package installers"
	munkiPackagesTag           = "Munki packages"
	munkiSoftwareTag           = "Munki software"
	osqueryChecksTag           = "Osquery checks"
	osqueryLiveQueriesTag      = "Osquery live queries"
	osqueryReportsTag          = "Osquery reports"
	santaConfigurationsTag     = "Santa configurations"
	santaEventsTag             = "Santa events"
	santaRulesTag              = "Santa rules"
	sessionTag                 = "Session"
	softwareTag                = "Software"
)

type openAPITagGroup struct {
	Name string   `json:"name" yaml:"name"`
	Tags []string `json:"tags" yaml:"tags"`
}

// ConfigureOpenAPI declares the resource hierarchy used by API documentation.
func ConfigureOpenAPI(doc *huma.OpenAPI) {
	doc.Tags = []*huma.Tag{
		resourceTag(accountTag, "Account"),
		resourceTag(agentSecretsTag, "Agent secrets"),
		resourceTag(directoryGroupsTag, "Groups"),
		resourceTag(directoryUsersTag, "Users"),
		resourceTag(hostsTag, "Hosts"),
		resourceTag(labelsTag, "Labels"),
		resourceTag(munkiClientResourcesTag, "Client resources"),
		resourceTag(munkiDistributionPointsTag, "Distribution points"),
		resourceTag(munkiIconsTag, "Icons"),
		resourceTag(munkiPackageInstallersTag, "Package installers"),
		resourceTag(munkiPackagesTag, "Packages"),
		resourceTag(munkiSoftwareTag, "Software"),
		resourceTag(osqueryChecksTag, "Checks"),
		resourceTag(osqueryLiveQueriesTag, "Live queries"),
		resourceTag(osqueryReportsTag, "Reports"),
		resourceTag(santaConfigurationsTag, "Configurations"),
		resourceTag(santaEventsTag, "Events"),
		resourceTag(santaRulesTag, "Rules"),
		resourceTag(sessionTag, "Session"),
		resourceTag(softwareTag, "Software"),
	}
	doc.Extensions = map[string]any{
		"x-tagGroups": []openAPITagGroup{
			{Name: "Account", Tags: []string{accountTag}},
			{Name: "Agent secrets", Tags: []string{agentSecretsTag}},
			{Name: "Directory", Tags: []string{directoryGroupsTag, directoryUsersTag}},
			{Name: "Hosts", Tags: []string{hostsTag}},
			{Name: "Labels", Tags: []string{labelsTag}},
			{Name: "Munki", Tags: []string{
				munkiClientResourcesTag,
				munkiDistributionPointsTag,
				munkiIconsTag,
				munkiPackageInstallersTag,
				munkiPackagesTag,
				munkiSoftwareTag,
			}},
			{Name: "Osquery", Tags: []string{
				osqueryChecksTag,
				osqueryLiveQueriesTag,
				osqueryReportsTag,
			}},
			{Name: "Santa", Tags: []string{
				santaConfigurationsTag,
				santaEventsTag,
				santaRulesTag,
			}},
			{Name: "Session", Tags: []string{sessionTag}},
			{Name: "Software", Tags: []string{softwareTag}},
		},
	}
}

func resourceTag(name string, displayName string) *huma.Tag {
	return &huma.Tag{
		Name:       name,
		Extensions: map[string]any{"x-displayName": displayName},
	}
}
