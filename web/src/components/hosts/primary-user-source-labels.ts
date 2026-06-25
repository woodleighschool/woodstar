const primaryUserSourceLabels: Record<string, string> = {
  manual: "Manual",
  orbit_profile: "Orbit profile",
};

export function primaryUserSourceLabel(source: string): string {
  return primaryUserSourceLabels[source] ?? source;
}
