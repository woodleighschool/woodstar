import { Link, useNavigate, useParams } from "@tanstack/react-router";
import { Loader2 } from "lucide-react";
import { useMemo, useState } from "react";
import { z } from "zod";

import { LabelPicker } from "@/components/labels/label-picker";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  useCreateSantaConfiguration,
  useSantaConfiguration,
  useSantaConfigurations,
  useUpdateSantaConfiguration,
  type SantaConfiguration,
  type SantaConfigurationMutation,
} from "@/hooks/use-santa";
import { MAX_PAGE_SIZE } from "@/lib/pagination";

import { CLIENT_MODE_OPTIONS } from "./shared";

type MediaAction = "none" | "allow" | "block" | "remount";

interface ConfigurationFormState {
  name: string;
  client_mode: SantaConfigurationMutation["client_mode"];
  label_ids: number[];
  enable_bundles: boolean;
  enable_transitive_rules: boolean;
  enable_all_event_upload: boolean;
  full_sync_interval_seconds: number;
  batch_size: number;
  allowed_path_regex: string;
  blocked_path_regex: string;
  removable_media_action: MediaAction;
  removable_media_remount_flags: string;
  encrypted_removable_media_action: MediaAction;
  encrypted_removable_media_remount_flags: string;
  event_detail_url: string;
  event_detail_text: string;
}

const MEDIA_ACTION_OPTIONS: { value: MediaAction; label: string }[] = [
  { value: "none", label: "No Policy" },
  { value: "allow", label: "Allow" },
  { value: "block", label: "Block" },
  { value: "remount", label: "Remount" },
];

const mediaActionSchema = z
  .object({
    action: z.enum(["none", "allow", "block", "remount"]),
    flags: z.string(),
  })
  .refine((value) => value.action !== "remount" || splitWords(value.flags).length > 0, {
    message: "Remount requires at least one mount flag.",
    path: ["flags"],
  });

const configurationFormSchema = z.object({
  removable_media: mediaActionSchema,
  encrypted_removable_media: mediaActionSchema,
});

// Santa client defaults sourced from upstream Santa. The form pre-fills these
// so the backend never substitutes hidden defaults.
const emptyConfigurationForm: ConfigurationFormState = {
  name: "",
  client_mode: "monitor",
  label_ids: [],
  enable_bundles: false,
  enable_transitive_rules: false,
  enable_all_event_upload: false,
  full_sync_interval_seconds: 600,
  batch_size: 50,
  allowed_path_regex: "",
  blocked_path_regex: "",
  removable_media_action: "none",
  removable_media_remount_flags: "",
  encrypted_removable_media_action: "none",
  encrypted_removable_media_remount_flags: "",
  event_detail_url: "",
  event_detail_text: "",
};

export function SantaConfigurationEditPage({ mode }: { mode: "create" | "edit" }) {
  const params = useParams({ strict: false });
  const configurationId = params.configurationId ?? "";
  const configurationID = mode === "edit" ? Number(configurationId) : null;
  const detail = useSantaConfiguration(configurationID);

  if (mode === "edit") {
    if (detail.error) {
      return (
        <PageShell>
          <Alert variant="destructive">
            <AlertTitle>Failed to Load Configuration</AlertTitle>
            <AlertDescription>{detail.error.message}</AlertDescription>
          </Alert>
        </PageShell>
      );
    }
    if (!detail.data) {
      return (
        <PageShell className="text-muted-foreground flex-row items-center gap-2 text-sm">
          <Loader2 className="animate-spin" /> Loading Configuration...
        </PageShell>
      );
    }
  }

  const initial = mode === "edit" && detail.data ? formFromConfiguration(detail.data) : emptyConfigurationForm;

  return (
    <ConfigurationForm key={configurationId || "new"} mode={mode} configurationId={configurationID} initial={initial} />
  );
}

function ConfigurationForm({
  mode,
  configurationId,
  initial,
}: {
  mode: "create" | "edit";
  configurationId: number | null;
  initial: ConfigurationFormState;
}) {
  const navigate = useNavigate();
  const create = useCreateSantaConfiguration();
  const update = useUpdateSantaConfiguration();
  const configurations = useSantaConfigurations({ page_size: MAX_PAGE_SIZE, sort: "position.asc" });
  const [form, setForm] = useState<ConfigurationFormState>(initial);
  const [showErrors, setShowErrors] = useState(false);
  const pending = create.isPending || update.isPending;
  const unavailableLabelIDs = useMemo(() => {
    const rows = configurations.data?.items ?? [];
    return rows.flatMap((configuration) =>
      configuration.id === configurationId ? [] : (configuration.label_ids ?? []),
    );
  }, [configurationId, configurations.data?.items]);
  const parsed = configurationFormSchema.safeParse({
    removable_media: {
      action: form.removable_media_action,
      flags: form.removable_media_remount_flags,
    },
    encrypted_removable_media: {
      action: form.encrypted_removable_media_action,
      flags: form.encrypted_removable_media_remount_flags,
    },
  });
  const mediaFlagsErrors = mediaFlagsErrorMap(parsed);

  async function submit() {
    if (!parsed.success) {
      setShowErrors(true);
      return;
    }
    const body = configurationBody(form);
    if (mode === "create") await create.mutateAsync(body);
    else await update.mutateAsync({ id: configurationId ?? 0, body });
    void navigate({ to: "/santa/configurations" });
  }

  return (
    <PageShell asChild>
      <form
        onSubmit={(event) => {
          event.preventDefault();
          void submit();
        }}
      >
        <PageHeader title={mode === "create" ? "New Configuration" : "Edit Configuration"} />

        <FieldGroup className="max-w-4xl">
          <Field>
            <FieldLabel htmlFor="santa-configuration-name">Name</FieldLabel>
            <Input
              id="santa-configuration-name"
              required
              value={form.name}
              onChange={(event) => setForm({ ...form, name: event.target.value })}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="santa-client-mode">Client Mode</FieldLabel>
            <Select
              value={form.client_mode}
              onValueChange={(client_mode) =>
                setForm({
                  ...form,
                  client_mode: client_mode as SantaConfigurationMutation["client_mode"],
                })
              }
            >
              <SelectTrigger id="santa-client-mode" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {CLIENT_MODE_OPTIONS.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
            <FieldDescription>Monitor observes. Lockdown enforces.</FieldDescription>
          </Field>
          <BoolField
            id="santa-enable-bundles"
            label="Bundles"
            description="Scan bundled applications."
            value={form.enable_bundles}
            onChange={(enable_bundles) => setForm({ ...form, enable_bundles })}
          />
          <BoolField
            id="santa-enable-transitive-rules"
            label="Transitive Rules"
            description="Allow compiled binaries to inherit allowlists."
            value={form.enable_transitive_rules}
            onChange={(enable_transitive_rules) => setForm({ ...form, enable_transitive_rules })}
          />
          <BoolField
            id="santa-upload-all-events"
            label="Upload All Events"
            description="Include explicitly allowed executions."
            value={form.enable_all_event_upload}
            onChange={(enable_all_event_upload) => setForm({ ...form, enable_all_event_upload })}
          />
          <Field>
            <FieldLabel htmlFor="santa-full-sync-interval">Full Sync Interval</FieldLabel>
            <Input
              id="santa-full-sync-interval"
              type="number"
              min={60}
              inputMode="numeric"
              value={form.full_sync_interval_seconds}
              onChange={(event) => setForm({ ...form, full_sync_interval_seconds: Number(event.target.value) })}
            />
            <FieldDescription>Clean sync cadence in seconds.</FieldDescription>
          </Field>
          <Field>
            <FieldLabel htmlFor="santa-batch-size">Batch Size</FieldLabel>
            <Input
              id="santa-batch-size"
              type="number"
              min={5}
              max={100}
              inputMode="numeric"
              value={form.batch_size}
              onChange={(event) => setForm({ ...form, batch_size: Number(event.target.value) })}
            />
            <FieldDescription>Rule rows per sync page.</FieldDescription>
          </Field>
          <Field>
            <FieldLabel htmlFor="santa-allowed-path-regex">Allowed Path Regex</FieldLabel>
            <Input
              id="santa-allowed-path-regex"
              value={form.allowed_path_regex}
              onChange={(event) => setForm({ ...form, allowed_path_regex: event.target.value })}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="santa-blocked-path-regex">Blocked Path Regex</FieldLabel>
            <Input
              id="santa-blocked-path-regex"
              value={form.blocked_path_regex}
              onChange={(event) => setForm({ ...form, blocked_path_regex: event.target.value })}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="santa-event-detail-url">Event Detail URL</FieldLabel>
            <Input
              id="santa-event-detail-url"
              value={form.event_detail_url}
              onChange={(event) => setForm({ ...form, event_detail_url: event.target.value })}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="santa-event-detail-text">Event Detail Text</FieldLabel>
            <Input
              id="santa-event-detail-text"
              value={form.event_detail_text}
              onChange={(event) => setForm({ ...form, event_detail_text: event.target.value })}
            />
          </Field>
          <MediaActionField
            id="santa-removable-media"
            label="Removable Media"
            action={form.removable_media_action}
            flags={form.removable_media_remount_flags}
            flagsError={showErrors ? mediaFlagsErrors.removable_media : undefined}
            onActionChange={(removable_media_action) => setForm({ ...form, removable_media_action })}
            onFlagsChange={(removable_media_remount_flags) => setForm({ ...form, removable_media_remount_flags })}
          />
          <MediaActionField
            id="santa-encrypted-removable-media"
            label="Encrypted Removable Media"
            action={form.encrypted_removable_media_action}
            flags={form.encrypted_removable_media_remount_flags}
            flagsError={showErrors ? mediaFlagsErrors.encrypted_removable_media : undefined}
            onActionChange={(encrypted_removable_media_action) =>
              setForm({ ...form, encrypted_removable_media_action })
            }
            onFlagsChange={(encrypted_removable_media_remount_flags) =>
              setForm({ ...form, encrypted_removable_media_remount_flags })
            }
          />
          <Field>
            <FieldLabel>Scope</FieldLabel>
            {configurations.isLoading ? (
              <p className="text-muted-foreground text-sm">Loading Configuration Scope...</p>
            ) : configurations.error ? (
              <p className="text-destructive text-sm">{configurations.error.message}</p>
            ) : (
              <LabelPicker
                value={form.label_ids}
                includeBuiltins
                unavailableLabelIDs={unavailableLabelIDs}
                emptyPlaceholder="No Unassigned Labels"
                emptyMessage="All labels are already assigned to configurations."
                onChange={(label_ids) => setForm({ ...form, label_ids })}
              />
            )}
            <FieldDescription>Labels receiving this configuration.</FieldDescription>
          </Field>
        </FieldGroup>

        <div className="flex max-w-4xl items-center gap-2 border-t pt-4">
          <Button type="submit" size="sm" disabled={pending}>
            {pending ? <Loader2 data-icon="inline-start" className="animate-spin" /> : null}
            Save
          </Button>
          <Button asChild type="button" variant="ghost" size="sm">
            <Link to="/santa/configurations">Cancel</Link>
          </Button>
        </div>
      </form>
    </PageShell>
  );
}

function BoolField({
  id,
  label,
  description,
  value,
  onChange,
}: {
  id: string;
  label: string;
  description: string;
  value: boolean;
  onChange: (value: boolean) => void;
}) {
  return (
    <Field>
      <div className="flex items-center gap-2">
        <Checkbox id={id} checked={value} onCheckedChange={(next) => onChange(next === true)} />
        <FieldLabel htmlFor={id}>{label}</FieldLabel>
      </div>
      <FieldDescription>{description}</FieldDescription>
    </Field>
  );
}

function MediaActionField({
  id,
  label,
  action,
  flags,
  flagsError,
  onActionChange,
  onFlagsChange,
}: {
  id: string;
  label: string;
  action: MediaAction;
  flags: string;
  flagsError?: string;
  onActionChange: (value: MediaAction) => void;
  onFlagsChange: (value: string) => void;
}) {
  return (
    <Field data-invalid={flagsError ? true : undefined}>
      <FieldLabel htmlFor={`${id}-action`}>{label}</FieldLabel>
      <Select value={action} onValueChange={(value) => onActionChange(value as MediaAction)}>
        <SelectTrigger id={`${id}-action`} className="w-full">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            {MEDIA_ACTION_OPTIONS.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
      {action === "remount" ? (
        <Input
          id={`${id}-flags`}
          placeholder="remount flags"
          required
          value={flags}
          onChange={(event) => onFlagsChange(event.target.value)}
        />
      ) : null}
      {flagsError ? <FieldError>{flagsError}</FieldError> : null}
      <FieldDescription>Remount requires one or more mount flags.</FieldDescription>
    </Field>
  );
}

function formFromConfiguration(configuration: SantaConfiguration): ConfigurationFormState {
  return {
    name: configuration.name,
    client_mode: configuration.client_mode,
    label_ids: configuration.label_ids ?? [],
    enable_bundles: configuration.enable_bundles,
    enable_transitive_rules: configuration.enable_transitive_rules,
    enable_all_event_upload: configuration.enable_all_event_upload,
    full_sync_interval_seconds: configuration.full_sync_interval_seconds,
    batch_size: configuration.batch_size,
    allowed_path_regex: configuration.allowed_path_regex ?? "",
    blocked_path_regex: configuration.blocked_path_regex ?? "",
    removable_media_action: configuration.removable_media_policy?.action ?? "none",
    removable_media_remount_flags: (configuration.removable_media_policy?.remount_flags ?? []).join(" "),
    encrypted_removable_media_action: configuration.encrypted_removable_media_policy?.action ?? "none",
    encrypted_removable_media_remount_flags: (configuration.encrypted_removable_media_policy?.remount_flags ?? []).join(
      " ",
    ),
    event_detail_url: configuration.event_detail_url ?? "",
    event_detail_text: configuration.event_detail_text ?? "",
  };
}

function configurationBody(form: ConfigurationFormState): SantaConfigurationMutation {
  return {
    name: form.name.trim(),
    client_mode: form.client_mode,
    label_ids: form.label_ids,
    enable_bundles: form.enable_bundles,
    enable_transitive_rules: form.enable_transitive_rules,
    enable_all_event_upload: form.enable_all_event_upload,
    full_sync_interval_seconds: form.full_sync_interval_seconds,
    batch_size: form.batch_size,
    allowed_path_regex: optionalText(form.allowed_path_regex),
    blocked_path_regex: optionalText(form.blocked_path_regex),
    removable_media_policy: removableMediaPolicyBody(form.removable_media_action, form.removable_media_remount_flags),
    encrypted_removable_media_policy: removableMediaPolicyBody(
      form.encrypted_removable_media_action,
      form.encrypted_removable_media_remount_flags,
    ),
    event_detail_url: optionalText(form.event_detail_url),
    event_detail_text: optionalText(form.event_detail_text),
  };
}

function removableMediaPolicyBody(action: MediaAction, flags: string) {
  if (action === "none") return undefined;
  return { action, remount_flags: splitWords(flags) };
}

function optionalText(value: string) {
  const trimmed = value.trim();
  return trimmed === "" ? undefined : trimmed;
}

function splitWords(value: string) {
  return value
    .split(/[\s,]+/)
    .map((part) => part.trim())
    .filter(Boolean);
}

function mediaFlagsErrorMap(result: ReturnType<typeof configurationFormSchema.safeParse>): {
  removable_media?: string;
  encrypted_removable_media?: string;
} {
  if (result.success) return {};
  const out: { removable_media?: string; encrypted_removable_media?: string } = {};
  for (const issue of result.error.issues) {
    const key = issue.path[0];
    if (key === "removable_media" && !out.removable_media) out.removable_media = issue.message;
    if (key === "encrypted_removable_media" && !out.encrypted_removable_media) {
      out.encrypted_removable_media = issue.message;
    }
  }
  return out;
}
