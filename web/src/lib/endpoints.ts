/**
 * Endpoint registry for the woodstar admin API.
 *
 * Endpoints with `implemented: false` are deliberate boundaries: the route
 * shape is decided but the backend handler is not yet wired. Hooks check this
 * flag and surface an honest "endpoint pending" empty state instead of
 * spamming the backend with requests that will 404.
 *
 * Flip the flag when the backend handler lands.
 */
export interface EndpointSpec {
  readonly path: string;
  readonly implemented: boolean;
}

const e = (path: string, implemented = false): EndpointSpec => ({ path, implemented });

export const endpoints = {
  health: e("/api/healthz", true),
  ready: e("/api/readyz", true),
  version: e("/api/version", true),

  setupStatus: e("/api/v1/setup/status"),
  setupComplete: e("/api/v1/setup"),

  authMe: e("/api/v1/auth/me"),
  authLogin: e("/api/v1/auth/login"),
  authLogout: e("/api/v1/auth/logout"),

  hosts: e("/api/v1/hosts"),
  host: (id: string): EndpointSpec => e(`/api/v1/hosts/${encodeURIComponent(id)}`),
  hostSoftware: (id: string): EndpointSpec =>
    e(`/api/v1/hosts/${encodeURIComponent(id)}/software`),
  hostMunkiIssues: (id: string): EndpointSpec =>
    e(`/api/v1/hosts/${encodeURIComponent(id)}/munki/issues`),
  hostReports: (id: string): EndpointSpec =>
    e(`/api/v1/hosts/${encodeURIComponent(id)}/reports`),
  hostChecks: (id: string): EndpointSpec =>
    e(`/api/v1/hosts/${encodeURIComponent(id)}/checks`),

  enrollSecrets: e("/api/v1/orbit/enroll-secrets"),

  santaTokens: e("/api/v1/santa/tokens"),
  santaProfiles: e("/api/v1/santa/profiles"),

  munkiTokens: e("/api/v1/munki/tokens"),
  munkiManifestProfiles: e("/api/v1/munki/manifest-profiles"),

  settingsOidc: e("/api/v1/settings/oidc"),
  settingsDirectory: e("/api/v1/settings/directory"),
  settingsTeam: e("/api/v1/settings/team"),
} as const;
