package handlers

func registerSanta(g Groups, deps Dependencies) {
	registerSantaHostState(g.Ordinary, deps.SantaState, deps.Hosts)
	registerSantaConfigurations(g.Ordinary, deps.SantaConfigurations)
	registerSantaRules(g.Ordinary, deps.SantaRules)
	registerSantaEvents(g.Ordinary, deps.SantaEvents)
	registerHostSantaRules(g.Ordinary, deps.SantaRules, deps.Hosts)
	registerSoftwareSantaReference(g.Ordinary, deps.SantaReferences)
}
