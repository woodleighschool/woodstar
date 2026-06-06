import { useForm } from "@tanstack/react-form";
import { useNavigate, useParams } from "@tanstack/react-router";
import { Loader2 } from "lucide-react";
import { z } from "zod";

import { MutableResourceTabs } from "@/components/layout/mutable-resource-tabs";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { LabelScopeEditor } from "@/components/targeting/label-scope-editor";
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
  useUpdateSantaConfiguration,
  type SantaConfiguration,
  type SantaConfigurationMutation,
} from "@/hooks/use-santa";
import { firstErrorMessage, integerRange, optionalText, requiredString } from "@/lib/form-validation";

import {
  CLIENT_MODE_OPTIONS,
  CLIENT_MODE_VALUES,
  MEDIA_ACTION_OPTIONS,
  MEDIA_ACTION_VALUES,
  type SantaMediaAction,
} from "./shared";

interface ConfigurationFormState {
  name: string;
  description: string;
  client_mode: SantaConfigurationMutation["client_mode"];
  targets: NonNullable<SantaConfigurationMutation["targets"]>;
  enable_bundles: boolean;
  enable_transitive_rules: boolean;
  enable_all_event_upload: boolean;
  full_sync_interval_seconds: number;
  batch_size: number;
  allowed_path_regex: string;
  blocked_path_regex: string;
  removable_media_action: SantaMediaAction;
  removable_media_remount_flags: string;
  encrypted_removable_media_action: SantaMediaAction;
  encrypted_removable_media_remount_flags: string;
  event_detail_url: string;
  event_detail_text: string;
}

const configurationFormSchema = z
  .object({
    name: requiredString("Name"),
    description: z.string().trim(),
    client_mode: z.enum(CLIENT_MODE_VALUES),
    targets: z.array(
      z.object({
        label_id: z.number().int("Label selection is invalid."),
        effect: z.enum(["include", "exclude"]),
      }),
    ),
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
  targets: [],
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

export function SantaConfigurationResourcePage({ mode }: { mode: "create" | "edit" }) {
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
  const form = useForm({
    defaultValues: initial,
    validators: {
      onSubmit: configurationFormSchema,
    },
    onSubmit: async ({ value }) => {
      const body = configurationBody(configurationFormSchema.parse(value));
      const saved =
        mode === "create"
          ? await create.mutateAsync(body)
          : await update.mutateAsync({ id: configurationId ?? 0, body });
      void navigate({ to: "/santa/configurations/$configurationId", params: { configurationId: String(saved.id) } });
    },
  });
  const pending = create.isPending || update.isPending;

  return (
    <PageShell asChild>
      <form
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit();
        }}
      >
        <form.Subscribe selector={(state) => state.values.name}>
          {(name) => <PageHeader title={mode === "create" ? "New Configuration" : name || "Configuration"} />}
        </form.Subscribe>

        <MutableResourceTabs
          tabs={[
            {
              value: "options",
              label: "Options",
              content: (
                <FieldGroup className="max-w-3xl">
                  <form.Field
                    name="name"
                    children={(field) => {
                      const error = firstErrorMessage(field.state.meta.errors);
                      return (
                        <Field data-invalid={error ? true : undefined}>
                          <FieldLabel htmlFor="santa-configuration-name" required>
                            Name
                          </FieldLabel>
                          <Input
                            id="santa-configuration-name"
                            name={field.name}
                            required
                            aria-invalid={error ? true : undefined}
                            value={field.state.value}
                            onBlur={field.handleBlur}
                            onChange={(event) => field.handleChange(event.target.value)}
                          />
                          {error ? <FieldError>{error}</FieldError> : null}
                        </Field>
                      );
                    }}
                  />
                  <form.Field
                    name="description"
                    children={(field) => (
                      <Field>
                        <FieldLabel htmlFor="santa-configuration-description">Description</FieldLabel>
                        <Textarea
                          id="santa-configuration-description"
                          name={field.name}
                          rows={3}
                          value={field.state.value}
                          onBlur={field.handleBlur}
                          onChange={(event) => field.handleChange(event.target.value)}
                        />
                      </Field>
                    )}
                  />
                  <form.Field
                    name="client_mode"
                    children={(field) => {
                      const error = firstErrorMessage(field.state.meta.errors);
                      return (
                        <Field data-invalid={error ? true : undefined}>
                          <FieldLabel htmlFor="santa-client-mode">Client Mode</FieldLabel>
                          <Select
                            value={field.state.value}
                            onValueChange={(clientMode) =>
                              field.handleChange(clientMode as SantaConfigurationMutation["client_mode"])
                            }
                          >
                            <SelectTrigger
                              id="santa-client-mode"
                              className="w-full"
                              aria-invalid={error ? true : undefined}
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
                          {error ? <FieldError>{error}</FieldError> : null}
                        </Field>
                      );
                    }}
                  />
                  <form.Field
                    name="enable_bundles"
                    children={(field) => (
                      <BoolField
                        id="santa-enable-bundles"
                        label="Bundles"
                        value={field.state.value}
                        onChange={field.handleChange}
                      />
                    )}
                  />
                  <form.Field
                    name="enable_transitive_rules"
                    children={(field) => (
                      <BoolField
                        id="santa-enable-transitive-rules"
                        label="Transitive Rules"
                        value={field.state.value}
                        onChange={field.handleChange}
                      />
                    )}
                  />
                  <form.Field
                    name="enable_all_event_upload"
                    children={(field) => (
                      <BoolField
                        id="santa-upload-all-events"
                        label="Upload All Events"
                        value={field.state.value}
                        onChange={field.handleChange}
                      />
                    )}
                  />
                  <div className="grid gap-4 md:grid-cols-2">
                    <form.Field
                      name="full_sync_interval_seconds"
                      children={(field) => {
                        const error = firstErrorMessage(field.state.meta.errors);
                        return (
                          <Field data-invalid={error ? true : undefined}>
                            <FieldLabel htmlFor="santa-full-sync-interval" required>
                              Full Sync Interval
                            </FieldLabel>
                            <Input
                              id="santa-full-sync-interval"
                              name={field.name}
                              type="number"
                              min={60}
                              step={1}
                              required
                              aria-invalid={error ? true : undefined}
                              inputMode="numeric"
                              value={field.state.value}
                              onBlur={field.handleBlur}
                              onChange={(event) => field.handleChange(Number(event.target.value))}
                            />
                            {error ? <FieldError>{error}</FieldError> : null}
                          </Field>
                        );
                      }}
                    />
                    <form.Field
                      name="batch_size"
                      children={(field) => {
                        const error = firstErrorMessage(field.state.meta.errors);
                        return (
                          <Field data-invalid={error ? true : undefined}>
                            <FieldLabel htmlFor="santa-batch-size" required>
                              Batch Size
                            </FieldLabel>
                            <Input
                              id="santa-batch-size"
                              name={field.name}
                              type="number"
                              min={5}
                              max={100}
                              step={1}
                              required
                              aria-invalid={error ? true : undefined}
                              inputMode="numeric"
                              value={field.state.value}
                              onBlur={field.handleBlur}
                              onChange={(event) => field.handleChange(Number(event.target.value))}
                            />
                            {error ? <FieldError>{error}</FieldError> : null}
                          </Field>
                        );
                      }}
                    />
                  </div>
                  <form.Field
                    name="allowed_path_regex"
                    children={(field) => (
                      <Field>
                        <FieldLabel htmlFor="santa-allowed-path-regex">Allowed Path Regex</FieldLabel>
                        <Input
                          id="santa-allowed-path-regex"
                          name={field.name}
                          value={field.state.value}
                          onBlur={field.handleBlur}
                          onChange={(event) => field.handleChange(event.target.value)}
                        />
                      </Field>
                    )}
                  />
                  <form.Field
                    name="blocked_path_regex"
                    children={(field) => (
                      <Field>
                        <FieldLabel htmlFor="santa-blocked-path-regex">Blocked Path Regex</FieldLabel>
                        <Input
                          id="santa-blocked-path-regex"
                          name={field.name}
                          value={field.state.value}
                          onBlur={field.handleBlur}
                          onChange={(event) => field.handleChange(event.target.value)}
                        />
                      </Field>
                    )}
                  />
                  <div className="grid gap-4 md:grid-cols-2">
                    <form.Field
                      name="event_detail_url"
                      children={(field) => (
                        <Field>
                          <FieldLabel htmlFor="santa-event-detail-url">Event Detail URL</FieldLabel>
                          <Input
                            id="santa-event-detail-url"
                            name={field.name}
                            value={field.state.value}
                            onBlur={field.handleBlur}
                            onChange={(event) => field.handleChange(event.target.value)}
                          />
                        </Field>
                      )}
                    />
                    <form.Field
                      name="event_detail_text"
                      children={(field) => (
                        <Field>
                          <FieldLabel htmlFor="santa-event-detail-text">Event Detail Text</FieldLabel>
                          <Input
                            id="santa-event-detail-text"
                            name={field.name}
                            value={field.state.value}
                            onBlur={field.handleBlur}
                            onChange={(event) => field.handleChange(event.target.value)}
                          />
                        </Field>
                      )}
                    />
                  </div>
                  <form.Field
                    name="removable_media_action"
                    children={(actionField) => (
                      <form.Field
                        name="removable_media_remount_flags"
                        children={(flagsField) => (
                          <MediaActionField
                            id="santa-removable-media"
                            label="Removable Media"
                            action={actionField.state.value}
                            flags={flagsField.state.value}
                            flagsError={firstErrorMessage(flagsField.state.meta.errors)}
                            onActionChange={actionField.handleChange}
                            onFlagsChange={flagsField.handleChange}
                          />
                        )}
                      />
                    )}
                  />
                  <form.Field
                    name="encrypted_removable_media_action"
                    children={(actionField) => (
                      <form.Field
                        name="encrypted_removable_media_remount_flags"
                        children={(flagsField) => (
                          <MediaActionField
                            id="santa-encrypted-removable-media"
                            label="Encrypted Removable Media"
                            action={actionField.state.value}
                            flags={flagsField.state.value}
                            flagsError={firstErrorMessage(flagsField.state.meta.errors)}
                            onActionChange={actionField.handleChange}
                            onFlagsChange={flagsField.handleChange}
                          />
                        )}
                      />
                    )}
                  />
                </FieldGroup>
              ),
            },
            {
              value: "scope",
              label: "Scope",
              content: (
                <form.Field
                  name="targets"
                  children={(field) => <LabelScopeEditor value={field.state.value} onChange={field.handleChange} />}
                />
              ),
            },
          ]}
        />

        <div className="flex items-center gap-2 border-t pt-4">
          <Button type="submit" size="sm" disabled={pending}>
            {pending ? <Loader2 data-icon="inline-start" className="animate-spin" /> : null}
            Save
          </Button>
          {mode === "create" ? (
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => void navigate({ to: "/santa/configurations" })}
            >
              Cancel
            </Button>
          ) : null}
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
  description?: string;
  value: boolean;
  onChange: (value: boolean) => void;
}) {
  return (
    <Field>
      <div className="flex items-center gap-2">
        <Checkbox id={id} checked={value} onCheckedChange={(next) => onChange(next === true)} />
        <FieldLabel htmlFor={id}>{label}</FieldLabel>
      </div>
      {description ? <FieldDescription>{description}</FieldDescription> : null}
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
  description?: string;
  action: SantaMediaAction;
  flags: string;
  flagsError?: string;
  onActionChange: (value: SantaMediaAction) => void;
  onFlagsChange: (value: string) => void;
}) {
  return (
    <Field data-invalid={flagsError ? true : undefined}>
      <FieldLabel htmlFor={`${id}-action`}>{label}</FieldLabel>
      <Select value={action} onValueChange={(value) => onActionChange(value as SantaMediaAction)}>
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
      {description ? <FieldDescription>{description}</FieldDescription> : null}
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
    targets: configuration.targets ?? [],
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
    targets: form.targets,
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

function removableMediaPolicyBody(action: SantaMediaAction, flags: string) {
  if (action === "none") return undefined;
  return { action, remount_flags: splitWords(flags) };
}

function splitWords(value: string) {
  return value
    .split(/[\s,]+/)
    .map((part) => part.trim())
    .filter(Boolean);
}
