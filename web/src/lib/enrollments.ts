import type { AgentSecret } from "@/lib/api";

export type Integration = AgentSecret["agent"];

export function integrationLabel(integration?: Integration) {
  if (integration === "santa") return "Santa";
  if (integration === "munki") return "Munki";
  if (integration === "orbit") return "Orbit";
  return "Integration";
}

export function enrollmentTitle(integration: Integration) {
  return `${integrationLabel(integration)} Enrollment`;
}

export function enrollmentDescription(integration: Integration) {
  if (integration === "orbit") {
    return "Orbit package, profile, and enroll secrets.";
  }
  if (integration === "munki") {
    return "Profile and bearer secrets for Munki.";
  }
  return "Profile and bearer secrets for Santa.";
}

export function secretUsageDescription(integration: Integration) {
  if (integration === "orbit") {
    return "Use these secrets for Orbit enrollment.";
  }
  if (integration === "munki") {
    return "Use these bearer secrets for Munki.";
  }
  return "Use these bearer secrets for Santa.";
}

export function deleteDescription(integration: Integration) {
  if (integration === "orbit") {
    return "New enrollments using this secret will fail. Existing hosts keep their issued node keys.";
  }
  if (integration === "munki") {
    return "Munki clients using this bearer secret will be rejected until they receive another active secret.";
  }
  return "Santa clients using this bearer secret will be rejected until they receive another active secret.";
}
