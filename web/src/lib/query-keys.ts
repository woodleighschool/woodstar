export const queryKeys = {
  version: ["version"] as const,
  authMe: ["auth", "me"] as const,
  users: ["users"] as const,
  hosts: ["hosts"] as const,
  host: (id: string) => ["hosts", id] as const,
  hostSoftware: (id: string) => ["hosts", id, "software"] as const,
  software: ["software"] as const,
  enrollSecrets: ["orbit", "enroll-secrets"] as const,
  santaTokens: ["santa", "tokens"] as const,
  munkiTokens: ["munki", "tokens"] as const,
};
