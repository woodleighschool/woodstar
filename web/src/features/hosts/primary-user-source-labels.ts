const primaryUserSourceLabels: Record<string, string> = {
  manual: "Manual",
  orbit_profile: "Orbit Profile",
};

export function primaryUserSourceLabel(source: string): string {
  return primaryUserSourceLabels[source] ?? source;
}
