package handlers

import "github.com/woodleighschool/woodstar/internal/api/middleware"

// Register mounts the app API handler wall.
func Register(g Groups, deps Dependencies) {
	RegisterAuth(g, deps.AuthService, deps.Users)
	registerDirectory(g, deps)
	registerHosts(g, deps)
	registerInventory(g, deps)
	registerLabels(g, deps)
	registerAgentAuth(g, deps)
	registerOsquery(g, deps)
	registerMunki(g, deps)
	registerSanta(g, deps)
}

type desiredNotifier interface {
	DesiredChanged()
}

func notifyDesired(notifier desiredNotifier) {
	if notifier != nil {
		notifier.DesiredChanged()
	}
}

func registerOsquery(g Groups, deps Dependencies) {
	registerOsqueryReports(g.Ordinary, deps.Reports)
	registerHostOsqueryReports(g.Ordinary, deps.Reports, deps.Hosts)
	registerOsqueryChecks(g.Ordinary, deps.Checks)
	registerHostOsqueryChecks(g.Ordinary, deps.Checks, deps.Hosts)
	registerLiveQueries(g.Sensitive, deps.LiveQueries, deps.Hosts)
}

func registerMunki(g Groups, deps Dependencies) {
	registerHostMunkiState(g.Ordinary, deps.MunkiHostState, deps.Hosts)
	registerMunkiSoftware(
		g.Ordinary,
		deps.MunkiSoftware,
		deps.MunkiPackages,
		deps.StorageObjects,
		deps.StorageBackend,
		deps.MunkiDistributionHub,
	)
	registerMunkiSoftwareIconContent(
		g.Router.With(middleware.RequireHTTPAuth(deps.AuthService)),
		deps.MunkiSoftware,
		deps.StorageObjects,
		deps.StorageBackend,
	)
	registerMunkiPackages(
		g.Ordinary,
		deps.MunkiPackages,
		deps.StorageObjects,
		deps.StorageBackend,
		deps.MunkiDistributionHub,
	)
	registerMunkiDistributionPoints(g.Sensitive, deps.MunkiDistribution)
}

func registerSanta(g Groups, deps Dependencies) {
	registerHostSantaState(g.Ordinary, deps.SantaState, deps.Hosts)
	registerSantaConfigurations(g.Ordinary, deps.SantaConfigurations)
	registerSantaRules(g.Ordinary, deps.SantaRules)
	registerSantaEvents(g.Ordinary, deps.SantaEvents)
	registerHostSantaRules(g.Ordinary, deps.SantaRules, deps.Hosts)
	registerSoftwareSantaReference(g.Ordinary, deps.SantaReferences)
}
