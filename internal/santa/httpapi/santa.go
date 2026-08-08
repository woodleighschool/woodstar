package httpapi

import (
	"log/slog"

	"github.com/woodleighschool/woodstar/internal/api"
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
	logger *slog.Logger,
) {
	registerAPI(routes, hostState, configurationStore, ruleStore, eventStore, logger)
}

// RegisterOpenAPI documents Santa endpoints without runtime services.
func RegisterOpenAPI(routes api.AppRoutes) {
	registerAPI(routes, nil, nil, nil, nil, nil)
}

func registerAPI(
	routes api.AppRoutes,
	hostState *santa.HostStateService,
	configurationStore *configurations.Store,
	ruleStore *rules.Store,
	eventStore *events.Store,
	logger *slog.Logger,
) {
	humaAPI := routes.Ordinary
	registerHostSantaState(humaAPI, hostState, logger)
	registerSantaConfigurations(humaAPI, configurationStore, logger)
	registerSantaRules(humaAPI, ruleStore, logger)
	registerSantaEvents(humaAPI, eventStore, logger)
	registerHostSantaRules(humaAPI, configurationStore, ruleStore, logger)
}
