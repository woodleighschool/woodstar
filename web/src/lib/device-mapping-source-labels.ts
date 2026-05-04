const deviceMappingSourceLabels: Record<string, string> = {
  manual: "Manual",
  orbit_profile: "Enrollment profile",
};

export function deviceMappingSourceLabel(source: string): string {
  return deviceMappingSourceLabels[source] ?? source;
}
