package handlers

import (
	"log/slog"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	"github.com/woodleighschool/woodstar/internal/santa/events"
	"github.com/woodleighschool/woodstar/internal/santa/references"
	"github.com/woodleighschool/woodstar/internal/santa/rules"
)

// RegisterSanta mounts Santa policy, event, software-reference, and host state
// endpoints.
func RegisterSanta(
	api huma.API,
	hostState *santa.HostStateService,
	configurationStore *configurations.Store,
	ruleStore *rules.Store,
	eventStore *events.Store,
	referenceStore *references.Store,
	logger *slog.Logger,
) {
	registerHostSantaState(api, hostState, logger)
	registerSantaConfigurations(api, configurationStore, logger)
	registerSantaRules(api, ruleStore, logger)
	registerSantaEvents(api, eventStore, logger)
	registerHostSantaRules(api, ruleStore, logger)
	registerSoftwareSantaReference(api, referenceStore, logger)
}
