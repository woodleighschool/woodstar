package handlers

import "github.com/woodleighschool/woodstar/internal/api/middleware"

type desiredNotifier interface {
	DesiredChanged()
}

func notifyDesired(notifier desiredNotifier) {
	if notifier != nil {
		notifier.DesiredChanged()
	}
}

func registerMunki(g Groups, deps Dependencies) {
	registerMunkiHostState(g.Ordinary, deps.MunkiHostState, deps.Hosts)
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
