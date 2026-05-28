import type { Agent } from "@/hooks/use-agent-secrets";

export type Integration = Agent;

export function integrationLabel(integration?: Integration) {
  if (integration === "santa") return "Santa";
  if (integration === "orbit") return "Orbit";
  return "Integration";
}

export function enrollmentTitle(integration: Integration) {
  return `${integrationLabel(integration)} Enrollment`;
}

export function enrollmentDescription(integration: Integration) {
  if (integration === "orbit") {
    return "Orbit package, configuration profile, and enroll secrets.";
  }
  return "Configuration profile and bearer secrets for Santa clients.";
}

export function secretUsageDescription(integration: Integration) {
  if (integration === "orbit") {
    return "Use these secrets to enroll Orbit and osquery hosts.";
  }
  return "Use these bearer secrets to authenticate Santa sync clients.";
}

export function deleteDescription(integration: Integration) {
  if (integration === "orbit") {
    return "Future Orbit and osquery enrollments using this secret will fail. Existing hosts keep their issued node keys.";
  }
  return "Santa clients using this bearer secret will be rejected until they receive another active secret.";
}
