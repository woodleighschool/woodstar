package handlers

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
