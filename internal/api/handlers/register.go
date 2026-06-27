package handlers

import "github.com/woodleighschool/woodstar/internal/api/middleware"

// Register mounts the app API handler wall.
func Register(g Groups, deps Dependencies) {
	deps.Logger = deps.Logger.With("component", "api")

	RegisterAuth(g, deps.AuthService, deps.Users, deps.Logger)
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
	registerOsqueryReports(g.Ordinary, deps.Reports, deps.Logger)
	registerHostOsqueryReports(g.Ordinary, deps.Reports, deps.Hosts, deps.Logger)
	registerOsqueryChecks(g.Ordinary, deps.Checks, deps.Logger)
	registerHostOsqueryChecks(g.Ordinary, deps.Checks, deps.Hosts, deps.Logger)
	registerLiveQueries(g.Sensitive, deps.LiveQueries, deps.Hosts, deps.Logger)
}

func registerMunki(g Groups, deps Dependencies) {
	registerHostMunkiState(g.Ordinary, deps.MunkiHostState, deps.Hosts, deps.Logger)
	registerMunkiSoftware(
		g.Ordinary,
		deps.MunkiSoftware,
		deps.MunkiPackages,
		deps.StorageObjects,
		deps.StorageBackend,
		deps.MunkiDistributionHub,
		deps.Logger,
	)
	registerMunkiSoftwareIconContent(
		g.Router.With(middleware.RequireHTTPAuth(deps.AuthService)),
		deps.MunkiSoftware,
		deps.StorageObjects,
		deps.StorageBackend,
		deps.Logger,
	)
	registerMunkiPackages(
		g.Ordinary,
		deps.MunkiPackages,
		deps.StorageObjects,
		deps.StorageBackend,
		deps.MunkiDistributionHub,
		deps.Logger,
	)
	registerMunkiDistributionPoints(g.Sensitive, deps.MunkiDistribution, deps.Logger)
}

func registerSanta(g Groups, deps Dependencies) {
	registerHostSantaState(g.Ordinary, deps.SantaState, deps.Hosts, deps.Logger)
	registerSantaConfigurations(g.Ordinary, deps.SantaConfigurations, deps.Logger)
	registerSantaRules(g.Ordinary, deps.SantaRules, deps.Logger)
	registerSantaEvents(g.Ordinary, deps.SantaEvents, deps.Logger)
	registerHostSantaRules(g.Ordinary, deps.SantaRules, deps.Hosts, deps.Logger)
	registerSoftwareSantaReference(g.Ordinary, deps.SantaReferences, deps.Logger)
}
