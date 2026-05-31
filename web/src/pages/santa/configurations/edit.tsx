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
import { Textarea } from "@/components/ui/textarea";
import {
  useCreateSantaConfiguration,
  useSantaConfiguration,
  useSantaConfigurations,
  useUpdateSantaConfiguration,
  type SantaConfiguration,
  type SantaConfigurationMutation,
} from "@/hooks/use-santa";
import { fieldErrors, integerRange, optionalText, positiveIntegerArray, requiredString } from "@/lib/form-validation";
import { MAX_PAGE_SIZE } from "@/lib/pagination";

import { CLIENT_MODE_OPTIONS, CLIENT_MODE_VALUES } from "./shared";

type MediaAction = "none" | "allow" | "block" | "remount";

interface ConfigurationFormState {
  name: string;
  description: string;
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
const MEDIA_ACTION_VALUES = ["none", "allow", "block", "remount"] as const;

const configurationFormSchema = z
  .object({
    name: requiredString("Name"),
    description: z.string().trim(),
    client_mode: z.enum(CLIENT_MODE_VALUES),
    label_ids: positiveIntegerArray("Label"),
    enable_bundles: z.boolean(),
    enable_transitive_rules: z.boolean(),
    enable_all_event_upload: z.boolean(),
    full_sync_interval_seconds: integerRange("Full sync interval", 60),
    batch_size: integerRange("Batch size", 5, 100),
    allowed_path_regex: z.string().trim(),
    blocked_path_regex: z.string().trim(),
    removable_media_action: z.enum(MEDIA_ACTION_VALUES),
    removable_media_remount_flags: z.string().trim(),
    encrypted_removable_media_action: z.enum(MEDIA_ACTION_VALUES),
    encrypted_removable_media_remount_flags: z.string().trim(),
    event_detail_url: z.string().trim(),
    event_detail_text: z.string().trim(),
  })
  .superRefine((value, ctx) => {
    if (value.removable_media_action === "remount" && splitWords(value.removable_media_remount_flags).length === 0) {
      ctx.addIssue({
        code: "custom",
        message: "Remount requires at least one mount flag.",
        path: ["removable_media_remount_flags"],
      });
    }
    if (
      value.encrypted_removable_media_action === "remount" &&
      splitWords(value.encrypted_removable_media_remount_flags).length === 0
    ) {
      ctx.addIssue({
        code: "custom",
        message: "Remount requires at least one mount flag.",
        path: ["encrypted_removable_media_remount_flags"],
      });
    }
  });

// Santa client defaults sourced from upstream Santa. The form pre-fills these
// so the backend never substitutes hidden defaults.
const emptyConfigurationForm: ConfigurationFormState = {
  name: "",
  description: "",
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
  const parsed = configurationFormSchema.safeParse(form);
  const errors = fieldErrors(parsed);

  async function submit() {
    const nextParsed = configurationFormSchema.safeParse(form);
    if (!nextParsed.success) {
      setShowErrors(true);
      return;
    }
    const body = configurationBody(nextParsed.data);
    if (mode === "create") await create.mutateAsync(body);
    else await update.mutateAsync({ id: configurationId ?? 0, body });
    void navigate({ to: "/santa/configurations" });
  }

  return (
    <PageShell asChild>
      <form
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          void submit();
        }}
      >
        <PageHeader title={mode === "create" ? "New Configuration" : "Edit Configuration"} />

        <FieldGroup>
          <Field data-invalid={showErrors && errors.name ? true : undefined}>
            <FieldLabel htmlFor="santa-configuration-name" required>
              Name
            </FieldLabel>
            <Input
              id="santa-configuration-name"
              required
              aria-invalid={showErrors && errors.name ? true : undefined}
              value={form.name}
              onChange={(event) => setForm({ ...form, name: event.target.value })}
            />
            {showErrors && errors.name ? <FieldError>{errors.name}</FieldError> : null}
          </Field>
          <Field>
            <FieldLabel htmlFor="santa-configuration-description">Description</FieldLabel>
            <Textarea
              id="santa-configuration-description"
              rows={3}
              value={form.description}
              onChange={(event) => setForm({ ...form, description: event.target.value })}
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
              <SelectTrigger
                id="santa-client-mode"
                className="w-full"
                aria-invalid={showErrors && errors.client_mode ? true : undefined}
              >
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
            <FieldDescription>
              Monitor records activity. Lockdown enforces matching rules. Standalone leaves Santa outside this
              configuration.
            </FieldDescription>
          </Field>
          <BoolField
            id="santa-enable-bundles"
            label="Bundles"
            description="Enables Santa bundle scanning."
            value={form.enable_bundles}
            onChange={(enable_bundles) => setForm({ ...form, enable_bundles })}
          />
          <BoolField
            id="santa-enable-transitive-rules"
            label="Transitive Rules"
            description="Allows compiler rules to create transitive allow rules for the binaries they produce."
            value={form.enable_transitive_rules}
            onChange={(enable_transitive_rules) => setForm({ ...form, enable_transitive_rules })}
          />
          <BoolField
            id="santa-upload-all-events"
            label="Upload All Events"
            description="Uploads explicitly allowed executions as well as blocked and scope decisions."
            value={form.enable_all_event_upload}
            onChange={(enable_all_event_upload) => setForm({ ...form, enable_all_event_upload })}
          />
          <Field data-invalid={showErrors && errors.full_sync_interval_seconds ? true : undefined}>
            <FieldLabel htmlFor="santa-full-sync-interval" required>
              Full Sync Interval
            </FieldLabel>
            <Input
              id="santa-full-sync-interval"
              type="number"
              min={60}
              step={1}
              required
              aria-invalid={showErrors && errors.full_sync_interval_seconds ? true : undefined}
              inputMode="numeric"
              value={form.full_sync_interval_seconds}
              onChange={(event) => setForm({ ...form, full_sync_interval_seconds: Number(event.target.value) })}
            />
            <FieldDescription>
              Maximum time, in seconds, before Santa performs a full sync. Santa enforces a 60 second minimum.
            </FieldDescription>
            {showErrors && errors.full_sync_interval_seconds ? (
              <FieldError>{errors.full_sync_interval_seconds}</FieldError>
            ) : null}
          </Field>
          <Field data-invalid={showErrors && errors.batch_size ? true : undefined}>
            <FieldLabel htmlFor="santa-batch-size" required>
              Batch Size
            </FieldLabel>
            <Input
              id="santa-batch-size"
              type="number"
              min={5}
              max={100}
              step={1}
              required
              aria-invalid={showErrors && errors.batch_size ? true : undefined}
              inputMode="numeric"
              value={form.batch_size}
              onChange={(event) => setForm({ ...form, batch_size: Number(event.target.value) })}
            />
            <FieldDescription>
              Rules downloaded or events uploaded per request. Santa makes another request when there is more work.
            </FieldDescription>
            {showErrors && errors.batch_size ? <FieldError>{errors.batch_size}</FieldError> : null}
          </Field>
          <Field>
            <FieldLabel htmlFor="santa-allowed-path-regex">Allowed Path Regex</FieldLabel>
            <Input
              id="santa-allowed-path-regex"
              value={form.allowed_path_regex}
              onChange={(event) => setForm({ ...form, allowed_path_regex: event.target.value })}
            />
            <FieldDescription>
              Matching binaries are allowed in both modes. Santa logs these events as ALLOW_SCOPE.
            </FieldDescription>
          </Field>
          <Field>
            <FieldLabel htmlFor="santa-blocked-path-regex">Blocked Path Regex</FieldLabel>
            <Input
              id="santa-blocked-path-regex"
              value={form.blocked_path_regex}
              onChange={(event) => setForm({ ...form, blocked_path_regex: event.target.value })}
            />
            <FieldDescription>In Monitor mode, Santa blocks executables whose paths match this regex.</FieldDescription>
          </Field>
          <Field>
            <FieldLabel htmlFor="santa-event-detail-url">Event Detail URL</FieldLabel>
            <Input
              id="santa-event-detail-url"
              value={form.event_detail_url}
              onChange={(event) => setForm({ ...form, event_detail_url: event.target.value })}
            />
            <FieldDescription>Optional link Santa can show with event details.</FieldDescription>
          </Field>
          <Field>
            <FieldLabel htmlFor="santa-event-detail-text">Event Detail Text</FieldLabel>
            <Input
              id="santa-event-detail-text"
              value={form.event_detail_text}
              onChange={(event) => setForm({ ...form, event_detail_text: event.target.value })}
            />
            <FieldDescription>Text shown for the event detail link.</FieldDescription>
          </Field>
          <MediaActionField
            id="santa-removable-media"
            label="Removable Media"
            description="Controls USB mass storage. Block prevents mounting; Remount mounts again with the flags below."
            action={form.removable_media_action}
            flags={form.removable_media_remount_flags}
            flagsError={showErrors ? errors.removable_media_remount_flags : undefined}
            onActionChange={(removable_media_action) => setForm({ ...form, removable_media_action })}
            onFlagsChange={(removable_media_remount_flags) => setForm({ ...form, removable_media_remount_flags })}
          />
          <MediaActionField
            id="santa-encrypted-removable-media"
            label="Encrypted Removable Media"
            description="Controls encrypted USB mass storage separately from other removable media."
            action={form.encrypted_removable_media_action}
            flags={form.encrypted_removable_media_remount_flags}
            flagsError={showErrors ? errors.encrypted_removable_media_remount_flags : undefined}
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
            <FieldDescription>
              Labels that select hosts for this configuration. A label can only belong to one configuration.
            </FieldDescription>
          </Field>
        </FieldGroup>

        <div className="flex items-center gap-2 border-t pt-4">
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
  description,
  action,
  flags,
  flagsError,
  onActionChange,
  onFlagsChange,
}: {
  id: string;
  label: string;
  description: string;
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
      <FieldDescription>{description}</FieldDescription>
      {action === "remount" ? (
        <div className="flex flex-col gap-1">
          <FieldLabel htmlFor={`${id}-flags`} required>
            Mount Flags
          </FieldLabel>
          <Input
            id={`${id}-flags`}
            placeholder="remount flags"
            required
            aria-invalid={flagsError ? true : undefined}
            value={flags}
            onChange={(event) => onFlagsChange(event.target.value)}
          />
          <FieldDescription>Separate mount options with commas or spaces.</FieldDescription>
        </div>
      ) : null}
      {flagsError ? <FieldError>{flagsError}</FieldError> : null}
    </Field>
  );
}

function formFromConfiguration(configuration: SantaConfiguration): ConfigurationFormState {
  return {
    name: configuration.name,
    description: configuration.description,
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
    description: optionalText(form.description),
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

function splitWords(value: string) {
  return value
    .split(/[\s,]+/)
    .map((part) => part.trim())
    .filter(Boolean);
}
