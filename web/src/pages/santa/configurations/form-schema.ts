import { z } from "zod";

import { integerRange, requiredString } from "@/lib/form-validation";
import {
  CLIENT_MODE_VALUES,
  FILE_ACCESS_ACTION_VALUES,
  MEDIA_ACTION_VALUES,
  REMOUNT_FLAG_VALUES,
} from "@/lib/santa-configurations";
import { labelTargetSetSchema } from "@/lib/targeting";

export const configurationFormSchema = z
  .object({
    name: requiredString("Name"),
    description: z.string().trim(),
    client_mode: z.enum(CLIENT_MODE_VALUES),
    targets: labelTargetSetSchema,
    enable_bundles: z.boolean(),
    enable_transitive_rules: z.boolean(),
    enable_all_event_upload: z.boolean(),
    disable_unknown_event_upload: z.boolean(),
    override_file_access_action: z.enum(FILE_ACCESS_ACTION_VALUES),
    full_sync_interval_seconds: integerRange("Full sync interval", 60),
    batch_size: integerRange("Batch size", 5, 100),
    allowed_path_regex: z.string().trim(),
    blocked_path_regex: z.string().trim(),
    removable_media_action: z.enum(MEDIA_ACTION_VALUES),
    removable_media_remount_flags: z.array(z.enum(REMOUNT_FLAG_VALUES)),
    encrypted_removable_media_action: z.enum(MEDIA_ACTION_VALUES),
    encrypted_removable_media_remount_flags: z.array(z.enum(REMOUNT_FLAG_VALUES)),
    event_detail_url: z
      .string()
      .trim()
      .refine(
        (value) => value === "" || isHTTPSURL(value),
        "Event detail URL must be an HTTPS URL.",
      ),
    event_detail_text: z.string().trim(),
  })
  .superRefine((value, ctx) => {
    if (
      value.removable_media_action === "remount" &&
      value.removable_media_remount_flags.length === 0
    ) {
      ctx.addIssue({
        code: "custom",
        message: "Remount requires at least one mount flag.",
        path: ["removable_media_remount_flags"],
      });
    }
    if (
      value.encrypted_removable_media_action === "remount" &&
      value.encrypted_removable_media_remount_flags.length === 0
    ) {
      ctx.addIssue({
        code: "custom",
        message: "Remount requires at least one mount flag.",
        path: ["encrypted_removable_media_remount_flags"],
      });
    }
    if (
      value.removable_media_action !== "remount" &&
      value.removable_media_remount_flags.length > 0
    ) {
      ctx.addIssue({
        code: "custom",
        message: "Mount flags are only valid for remount.",
        path: ["removable_media_remount_flags"],
      });
    }
    if (
      value.encrypted_removable_media_action !== "remount" &&
      value.encrypted_removable_media_remount_flags.length > 0
    ) {
      ctx.addIssue({
        code: "custom",
        message: "Mount flags are only valid for remount.",
        path: ["encrypted_removable_media_remount_flags"],
      });
    }
  });

export type ConfigurationFormInput = z.input<typeof configurationFormSchema>;
export type ConfigurationFormOutput = z.output<typeof configurationFormSchema>;

function isHTTPSURL(value: string) {
  try {
    const url = new URL(value);
    return (
      url.protocol === "https:" && url.host !== "" && url.username === "" && url.password === ""
    );
  } catch {
    return false;
  }
}
