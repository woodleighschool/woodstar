import type { AgentSecret } from "@lib/api";

export type Integration = AgentSecret["agent"];

const DOCS_BASE_URL = "https://woodleighschool.github.io/woodstar/docs/agent-protocols";

export function integrationLabel(integration?: Integration) {
  if (integration === "santa") return "Santa";
  if (integration === "munki") return "Munki";
  if (integration === "orbit") return "Host";
  return "Integration";
}

export function enrollmentDialogTitle(integration: Integration) {
  if (integration === "orbit") return "Enroll Hosts";
  return `Manage ${integrationLabel(integration)} Enrollment Secrets`;
}

export function enrollmentCardTitle(integration: Integration) {
  return integration === "orbit" ? "Host Enrollment" : "Enrollment Secrets";
}

export function enrollmentCardDescription(integration: Integration) {
  if (integration === "orbit") return "Shared by Orbit and direct osquery enrollment.";
  if (integration === "munki") return "Bearer credentials used by Munki clients.";
  return "Bearer credentials used by Santa clients.";
}

export function secretUsageDescription(integration: Integration) {
  if (integration === "orbit") {
    return "Use these shared secrets to enroll hosts through Orbit or osquery.";
  }
  if (integration === "munki") {
    return "Use these bearer secrets for Munki.";
  }
  return "Use these bearer secrets for Santa.";
}

export function deleteDescription(integration: Integration) {
  if (integration === "orbit") {
    return "New Orbit and osquery enrollments using this secret will fail. Existing hosts keep their issued node keys.";
  }
  if (integration === "munki") {
    return "Munki clients using this bearer secret will be rejected until they receive another active secret.";
  }
  return "Santa clients using this bearer secret will be rejected until they receive another active secret.";
}

export function enrollmentDocsURL(integration: Integration) {
  if (integration === "orbit") return `${DOCS_BASE_URL}/orbit-and-osquery`;
  if (integration === "munki") return `${DOCS_BASE_URL}/munki-repository`;
  return `${DOCS_BASE_URL}/santa-sync`;
}
