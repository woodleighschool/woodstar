export const queryKeys = {
  version: ["version"] as const,
  authMe: ["auth", "me"] as const,
  users: ["users"] as const,
  hosts: (params?: unknown) => ["hosts", params ?? {}] as const,
  host: (id: string) => ["hosts", id] as const,
  hostSoftware: (id: string, params?: unknown) => ["hosts", id, "software", params ?? {}] as const,
  software: (params?: unknown) => ["software", params ?? {}] as const,
  softwareTitle: (id: string) => ["software", id] as const,
  enrollSecrets: ["orbit", "enroll-secrets"] as const,
  santaTokens: ["santa", "tokens"] as const,
  munkiTokens: ["munki", "tokens"] as const,
};
