type QueryParams = Record<string, unknown>;

export const queryKeys = {
  session: ["auth", "session"] as const,
  account: ["account"] as const,
  // Root prefixes for invalidation: a list factory key is ["users", params], so the
  // bare prefix matches both the list and its detail/sub-resource keys.
  usersAll: ["users"] as const,
  groupsAll: ["groups"] as const,
  hostsAll: ["hosts"] as const,
  labelsAll: ["labels"] as const,
  checksAll: ["checks"] as const,
  reportsAll: ["reports"] as const,
  munkiSoftwareAll: ["munki", "software"] as const,
  munkiPackagesAll: ["munki", "packages"] as const,
  munkiDistributionPointsAll: ["munki", "distribution-points"] as const,
  santaConfigurationsAll: ["santa", "configurations"] as const,
  santaRulesAll: ["santa", "rules"] as const,
  users: (params?: QueryParams) => ["users", "list", params ?? {}] as const,
  user: (id: number | null) => ["users", "detail", id] as const,
  userDepartments: (params?: QueryParams) =>
    ["users", "departments", "list", params ?? {}] as const,
  groups: (params?: QueryParams) => ["groups", "list", params ?? {}] as const,
  group: (id: number | null) => ["groups", "detail", id] as const,
  osquerySchema: ["osquery-schema"] as const,
  hosts: (params?: QueryParams) => ["hosts", "list", params ?? {}] as const,
  host: (id: number | null) => ["hosts", "detail", id] as const,
  hostSoftware: (id: number | null, params?: QueryParams) =>
    ["hosts", "detail", id, "software", "list", params ?? {}] as const,
  hostMunkiState: (id: number | null) => ["hosts", "detail", id, "munki"] as const,
  hostOsqueryReports: (id: number | null) => ["hosts", "detail", id, "osquery", "reports"] as const,
  hostOsqueryChecks: (id: number | null) => ["hosts", "detail", id, "osquery", "checks"] as const,
  hostSantaState: (id: number | null) => ["hosts", "detail", id, "santa"] as const,
  hostSantaRules: (id: number | null, params?: QueryParams) =>
    ["hosts", "detail", id, "santa", "rules", "list", params ?? {}] as const,
  labels: (params?: QueryParams) => ["labels", "list", params ?? {}] as const,
  label: (id: number | null) => ["labels", "detail", id] as const,
  reports: (params?: QueryParams) => ["reports", "list", params ?? {}] as const,
  report: (id: number | null) => ["reports", "detail", id] as const,
  reportResults: (id: number | null) => ["reports", "detail", id, "results"] as const,
  checks: (params?: QueryParams) => ["checks", "list", params ?? {}] as const,
  check: (id: number | null) => ["checks", "detail", id] as const,
  checkResults: (id: number | null, params?: QueryParams) =>
    ["checks", "detail", id, "results", params ?? {}] as const,
  software: (params?: QueryParams) => ["software", "list", params ?? {}] as const,
  softwareTitle: (id: number | null) => ["software", "detail", id] as const,
  softwareSantaReference: (id: number | null) => ["software", "detail", id, "santa"] as const,
  munkiSoftware: (params?: QueryParams) => ["munki", "software", "list", params ?? {}] as const,
  munkiSoftwareDetail: (id: number | null) => ["munki", "software", "detail", id] as const,
  munkiPackages: (params?: QueryParams) => ["munki", "packages", "list", params ?? {}] as const,
  munkiPackage: (id: number | null) => ["munki", "packages", "detail", id] as const,
  munkiIconsAll: ["munki", "icons"] as const,
  munkiIcons: (params?: QueryParams) => ["munki", "icons", "list", params ?? {}] as const,
  munkiDistributionPoints: (params?: QueryParams) =>
    ["munki", "distribution-points", "list", params ?? {}] as const,
  munkiDistributionPoint: (id: number | null) =>
    ["munki", "distribution-points", "detail", id] as const,
  agentSecrets: ["agent-secrets"] as const,
  santaConfigurations: (params?: QueryParams) =>
    ["santa", "configurations", "list", params ?? {}] as const,
  santaConfiguration: (id: number | null) => ["santa", "configurations", "detail", id] as const,
  santaEvents: (params?: QueryParams) => ["santa", "events", "list", params ?? {}] as const,
  santaEvent: (id: number | null) => ["santa", "events", "detail", id] as const,
  santaFileAccessEvents: (params?: QueryParams) =>
    ["santa", "file-access-events", "list", params ?? {}] as const,
  santaFileAccessEvent: (id: number | null) =>
    ["santa", "file-access-events", "detail", id] as const,
  santaRules: (params?: QueryParams) => ["santa", "rules", "list", params ?? {}] as const,
  santaRule: (id: number | null) => ["santa", "rules", "detail", id] as const,
  santaRuleReferences: (params?: QueryParams) =>
    ["santa", "rule-references", "list", params ?? {}] as const,
  liveQueryTargetCount: (params?: QueryParams) =>
    ["live-query-target-count", params ?? {}] as const,
};
