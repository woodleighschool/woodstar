import type { SantaConfiguration, SantaConfigurationMutation } from "@/lib/api";
import {
  isSantaRemountFlag,
  type SantaMediaAction,
  type SantaRemountFlag,
} from "@/lib/santa-configurations";
import { emptyLabelTargetSet } from "@/lib/targeting";
import { nonEmpty } from "@/lib/utils";

import type { ConfigurationFormInput, ConfigurationFormOutput } from "./form-schema";

// Santa client defaults sourced from upstream Santa. The form pre-fills these
// so the backend never substitutes hidden defaults.
export const emptyConfigurationForm = {
  name: "",
  description: "",
  client_mode: "monitor",
  targets: emptyLabelTargetSet(),
  enable_bundles: false,
  enable_transitive_rules: false,
  enable_all_event_upload: false,
  disable_unknown_event_upload: false,
  override_file_access_action: "none",
  full_sync_interval_seconds: 600,
  batch_size: 50,
  allowed_path_regex: "",
  blocked_path_regex: "",
  removable_media_action: "none",
  removable_media_remount_flags: [],
  encrypted_removable_media_action: "none",
  encrypted_removable_media_remount_flags: [],
  event_detail_url: "",
  event_detail_text: "",
} satisfies ConfigurationFormInput;

export function formFromConfiguration(configuration: SantaConfiguration): ConfigurationFormInput {
  return {
    name: configuration.name,
    description: configuration.description,
    client_mode: configuration.client_mode,
    targets: configuration.targets,
    enable_bundles: configuration.enable_bundles,
    enable_transitive_rules: configuration.enable_transitive_rules,
    enable_all_event_upload: configuration.enable_all_event_upload,
    disable_unknown_event_upload: configuration.disable_unknown_event_upload,
    override_file_access_action: configuration.override_file_access_action,
    full_sync_interval_seconds: configuration.full_sync_interval_seconds,
    batch_size: configuration.batch_size,
    allowed_path_regex: configuration.allowed_path_regex ?? "",
    blocked_path_regex: configuration.blocked_path_regex ?? "",
    removable_media_action: configuration.removable_media_policy?.action ?? "none",
    removable_media_remount_flags: filterRemountFlags(
      configuration.removable_media_policy?.remount_flags ?? [],
    ),
    encrypted_removable_media_action:
      configuration.encrypted_removable_media_policy?.action ?? "none",
    encrypted_removable_media_remount_flags: filterRemountFlags(
      configuration.encrypted_removable_media_policy?.remount_flags ?? [],
    ),
    event_detail_url: configuration.event_detail_url ?? "",
    event_detail_text: configuration.event_detail_text ?? "",
  };
}

export function configurationBody(form: ConfigurationFormOutput): SantaConfigurationMutation {
  const mutation = {
    name: form.name.trim(),
    description: nonEmpty(form.description),
    client_mode: form.client_mode,
    targets: form.targets,
    enable_bundles: form.enable_bundles,
    enable_transitive_rules: form.enable_transitive_rules,
    enable_all_event_upload: form.enable_all_event_upload,
    disable_unknown_event_upload: form.disable_unknown_event_upload,
    override_file_access_action: form.override_file_access_action,
    full_sync_interval_seconds: form.full_sync_interval_seconds,
    batch_size: form.batch_size,
    allowed_path_regex: nonEmpty(form.allowed_path_regex),
    blocked_path_regex: nonEmpty(form.blocked_path_regex),
    removable_media_policy: removableMediaPolicyBody(
      form.removable_media_action,
      form.removable_media_remount_flags,
    ),
    encrypted_removable_media_policy: removableMediaPolicyBody(
      form.encrypted_removable_media_action,
      form.encrypted_removable_media_remount_flags,
    ),
    event_detail_url: nonEmpty(form.event_detail_url),
    event_detail_text: nonEmpty(form.event_detail_text),
  } satisfies Record<keyof SantaConfigurationMutation, unknown>;

  return mutation;
}

function removableMediaPolicyBody(action: SantaMediaAction, flags: SantaRemountFlag[]) {
  if (action === "none") return undefined;
  return action === "remount" ? { action, remount_flags: flags } : { action };
}

function filterRemountFlags(flags: string[]) {
  return flags.filter(isSantaRemountFlag);
}
