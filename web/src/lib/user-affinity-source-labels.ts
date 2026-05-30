const userAffinitySourceLabels: Record<string, string> = {
  manual: "Manual",
  orbit_profile: "Orbit profile",
  santa_primary_user: "Santa profile",
};

export function userAffinitySourceLabel(source: string): string {
  return userAffinitySourceLabels[source] ?? source;
}
