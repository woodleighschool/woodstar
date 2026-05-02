export interface VersionInfo {
  version: string;
  started_at: string;
}

export interface CurrentUser {
  id: string;
  email: string;
  name: string;
  role: "admin" | "viewer";
}

export type HostStatus = "online" | "offline";

export interface Host {
  id: string;
  display_name: string;
  hostname: string;
  computer_name: string;
  hardware_serial: string;
  hardware_model: string;
  uuid: string;
  os_version: string;
  osquery_version: string;
  primary_ip: string;
  primary_mac: string;
  memory_bytes: number;
  status: HostStatus;
  last_seen_at: string;
  detail_updated_at: string;
  issues_count: number;
}

export interface HostDetail extends Host {
  cpu_type: string;
  disk_total_bytes: number;
  disk_available_bytes: number;
  battery_cycle_count?: number;
  filevault_enabled?: boolean;
  mdm_server_url?: string;
  munki_version?: string;
  public_ip?: string;
  last_restarted_at?: string;
  labels: Label[];
}

export interface Label {
  id: string;
  name: string;
  source: "builtin" | "manual" | "dynamic" | "directory";
}

export interface EnrollSecret {
  id: string;
  secret_preview: string;
  created_at: string;
  rotated_at?: string;
  host_count: number;
}

export interface SantaToken {
  id: string;
  label?: string;
  preview: string;
  created_at: string;
  last_used_at?: string;
}

export interface SantaProfile {
  id: string;
  name: string;
  client_mode: "MONITOR" | "LOCKDOWN" | "STANDALONE";
  scope: "default" | "label" | "directory_group" | "host";
  scope_target?: string;
  rule_count: number;
  updated_at: string;
}

export interface MunkiToken {
  id: string;
  label?: string;
  preview: string;
  created_at: string;
  last_used_at?: string;
}

export interface MunkiManifestProfile {
  id: string;
  name: string;
  scope: "baseline" | "label" | "directory_group" | "host";
  scope_target?: string;
  catalog_count: number;
  package_count: number;
  updated_at: string;
}
