package httpapi

import (
	"log/slog"

	authhuma "github.com/woodleighschool/goodies/auth/huma"

	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/rbac"
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	"github.com/woodleighschool/woodstar/internal/santa/events"
	"github.com/woodleighschool/woodstar/internal/santa/rules"
)

// RegisterAPI mounts Santa policy, event, and host state endpoints.
func RegisterAPI(
	routes api.AppRoutes,
	hostState *santa.HostStateService,
	configurationStore *configurations.Store,
	ruleStore *rules.Store,
	eventStore *events.Store,
	authorizer authhuma.Authorizer,
	logger *slog.Logger,
) {
	registerAPI(routes, hostState, configurationStore, ruleStore, eventStore, authorizer, logger)
}

// RegisterOpenAPI documents Santa endpoints without runtime services.
func RegisterOpenAPI(routes api.AppRoutes) {
	registerAPI(routes, nil, nil, nil, nil, nil, nil)
}

func registerAPI(
	routes api.AppRoutes,
	hostState *santa.HostStateService,
	configurationStore *configurations.Store,
	ruleStore *rules.Store,
	eventStore *events.Store,
	authorizer authhuma.Authorizer,
	logger *slog.Logger,
) {
	configurationsAPI := authhuma.ResourceAPI(routes.Protected, authorizer, logger, rbac.ResourceSantaConfigurations)
	eventsAPI := authhuma.ResourceAPI(routes.Protected, authorizer, logger, rbac.ResourceSantaEvents)
	rulesAPI := authhuma.ResourceAPI(routes.Protected, authorizer, logger, rbac.ResourceSantaRules)

	registerHostSantaState(configurationsAPI, hostState, logger)
	registerSantaConfigurations(configurationsAPI, configurationStore, logger)
	registerSantaRules(rulesAPI, ruleStore, logger)
	registerSantaEvents(eventsAPI, eventStore, logger)
	registerHostSantaRules(rulesAPI, ruleStore, logger)
}
