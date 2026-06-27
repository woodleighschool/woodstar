package handlers

func registerOsquery(g Groups, deps Dependencies) {
	registerOsqueryReports(g.Ordinary, deps.Reports)
	registerHostOsqueryReports(g.Ordinary, deps.Reports, deps.Hosts)
	registerOsqueryChecks(g.Ordinary, deps.Checks)
	registerHostOsqueryChecks(g.Ordinary, deps.Checks, deps.Hosts)
	registerLiveQueries(g.Sensitive, deps.LiveQueries, deps.Hosts)
}
