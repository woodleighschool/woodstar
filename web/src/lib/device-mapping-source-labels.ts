const deviceMappingSourceLabels: Record<string, string> = {
  manual: "Manual",
  orbit_profile: "Orbit profile",
  santa_primary_user: "Santa profile",
};

export function deviceMappingSourceLabel(source: string): string {
  return deviceMappingSourceLabels[source] ?? source;
}
