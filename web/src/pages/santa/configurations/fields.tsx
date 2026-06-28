import { revalidateLogic, useForm } from "@tanstack/react-form";
import { z } from "zod";

import { FormActions } from "@/components/form-actions";
import { FormField } from "@/components/form-field";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { ScrollableTabs, ScrollableTabsList } from "@/components/layout/scrollable-tabs";
import { LabelTargetSetEditor } from "@/components/targeting/label-target-set-editor";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { TabsContent, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import type { SantaConfiguration, SantaConfigurationMutation } from "@/lib/api";
import { firstErrorMessage, integerRange, requiredString } from "@/lib/form-validation";
import {
  CLIENT_MODE_OPTIONS,
  CLIENT_MODE_VALUES,
  FILE_ACCESS_ACTION_OPTIONS,
  FILE_ACCESS_ACTION_VALUES,
  MEDIA_ACTION_OPTIONS,
  MEDIA_ACTION_VALUES,
  REMOUNT_FLAG_OPTIONS,
  REMOUNT_FLAG_VALUES,
  type SantaFileAccessAction,
  type SantaMediaAction,
  type SantaRemountFlag,
} from "@/lib/santa-configurations";
import { emptyLabelTargetSet, labelTargetSetSchema } from "@/lib/targeting";
import { nonEmpty } from "@/lib/utils";

interface ConfigurationFormState {
  name: string;
  description: string;
  client_mode: SantaConfigurationMutation["client_mode"];
  targets: NonNullable<SantaConfigurationMutation["targets"]>;
  enable_bundles: boolean;
  enable_transitive_rules: boolean;
  enable_all_event_upload: boolean;
  disable_unknown_event_upload: boolean;
  override_file_access_action: SantaFileAccessAction;
  full_sync_interval_seconds: number;
  batch_size: number;
  allowed_path_regex: string;
  blocked_path_regex: string;
  removable_media_action: SantaMediaAction;
  removable_media_remount_flags: SantaRemountFlag[];
  encrypted_removable_media_action: SantaMediaAction;
  encrypted_removable_media_remount_flags: SantaRemountFlag[];
  event_detail_url: string;
  event_detail_text: string;
}

const configurationFormSchema = z
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
    event_detail_url: z.string().trim(),
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
  });

// Santa client defaults sourced from upstream Santa. The form pre-fills these
// so the backend never substitutes hidden defaults.
export const emptyConfigurationForm: ConfigurationFormState = {
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
};

export function ConfigurationForm({
  initial,
  title,
  submitLabel,
  onSubmit,
  onCancel,
}: {
  initial: ConfigurationFormState;
  title?: string;
  submitLabel: string;
  onSubmit: (body: SantaConfigurationMutation) => Promise<void> | void;
  onCancel?: () => void;
}) {
  const form = useForm({
    defaultValues: initial,
    validationLogic: revalidateLogic(),
    validators: { onDynamic: configurationFormSchema },
    onSubmit: async ({ value }) =>
      onSubmit(configurationBody(configurationFormSchema.parse(value))),
  });

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
          {(name) => <PageHeader title={title ?? (name || "Configuration")} />}
        </form.Subscribe>

        <ScrollableTabs defaultValue="options">
          <ScrollableTabsList>
            <TabsTrigger value="options">Options</TabsTrigger>
            <TabsTrigger value="targets">Targets</TabsTrigger>
          </ScrollableTabsList>

          <TabsContent value="options">
            <FieldGroup className="max-w-3xl">
              <form.Field name="name">
                {(field) => (
                  <FormField field={field} label="Name" htmlFor="santa-configuration-name" required>
                    {(control) => (
                      <Input
                        {...control}
                        name={field.name}
                        required
                        value={field.state.value}
                        onBlur={field.handleBlur}
                        onChange={(event) => field.handleChange(event.target.value)}
                      />
                    )}
                  </FormField>
                )}
              </form.Field>
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
              <form.Field name="client_mode">
                {(field) => (
                  <FormField field={field} label="Client Mode" htmlFor="santa-client-mode">
                    {(control) => (
                      <Select
                        value={field.state.value}
                        onValueChange={(clientMode) =>
                          field.handleChange(
                            clientMode as SantaConfigurationMutation["client_mode"],
                          )
                        }
                      >
                        <SelectTrigger {...control} className="w-full">
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
                    )}
                  </FormField>
                )}
              </form.Field>
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
              <form.Field
                name="disable_unknown_event_upload"
                children={(field) => (
                  <BoolField
                    id="santa-disable-unknown-event-upload"
                    label="Disable Unknown Event Upload"
                    value={field.state.value}
                    onChange={field.handleChange}
                  />
                )}
              />
              <form.Field name="override_file_access_action">
                {(field) => (
                  <FormField
                    field={field}
                    label="File Access Override"
                    htmlFor="santa-file-access-override"
                    required
                  >
                    {(control) => (
                      <Select
                        value={field.state.value}
                        onValueChange={(value) =>
                          field.handleChange(value as SantaFileAccessAction)
                        }
                      >
                        <SelectTrigger {...control} className="w-full">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectGroup>
                            {FILE_ACCESS_ACTION_OPTIONS.map((option) => (
                              <SelectItem key={option.value} value={option.value}>
                                {option.label}
                              </SelectItem>
                            ))}
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                    )}
                  </FormField>
                )}
              </form.Field>
              <div className="grid gap-4 md:grid-cols-2">
                <form.Field name="full_sync_interval_seconds">
                  {(field) => (
                    <FormField
                      field={field}
                      label="Full Sync Interval"
                      htmlFor="santa-full-sync-interval"
                      required
                    >
                      {(control) => (
                        <Input
                          {...control}
                          name={field.name}
                          type="number"
                          min={60}
                          step={1}
                          required
                          inputMode="numeric"
                          value={field.state.value}
                          onBlur={field.handleBlur}
                          onChange={(event) => field.handleChange(Number(event.target.value))}
                        />
                      )}
                    </FormField>
                  )}
                </form.Field>
                <form.Field name="batch_size">
                  {(field) => (
                    <FormField field={field} label="Batch Size" htmlFor="santa-batch-size" required>
                      {(control) => (
                        <Input
                          {...control}
                          name={field.name}
                          type="number"
                          min={5}
                          max={100}
                          step={1}
                          required
                          inputMode="numeric"
                          value={field.state.value}
                          onBlur={field.handleBlur}
                          onChange={(event) => field.handleChange(Number(event.target.value))}
                        />
                      )}
                    </FormField>
                  )}
                </form.Field>
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
          </TabsContent>

          <TabsContent value="targets">
            <form.Field
              name="targets"
              children={(field) => (
                <LabelTargetSetEditor
                  value={field.state.value}
                  onChange={(next) => field.handleChange(next)}
                />
              )}
            />
          </TabsContent>
        </ScrollableTabs>

        <FormActions form={form} submitLabel={submitLabel} onCancel={onCancel} />
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
    <Field orientation="horizontal">
      <FieldContent>
        <FieldLabel htmlFor={id}>{label}</FieldLabel>
        {description ? <FieldDescription>{description}</FieldDescription> : null}
      </FieldContent>
      <Switch id={id} checked={value} onCheckedChange={onChange} />
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
  flags: SantaRemountFlag[];
  flagsError?: string;
  onActionChange: (value: SantaMediaAction) => void;
  onFlagsChange: (value: SantaRemountFlag[]) => void;
}) {
  return (
    <Field data-invalid={flagsError ? true : undefined}>
      <FieldLabel>{label}</FieldLabel>
      <ToggleGroup
        type="single"
        value={action}
        variant="outline"
        className="flex-wrap"
        onValueChange={(value) => {
          if (value) onActionChange(value as SantaMediaAction);
        }}
      >
        {MEDIA_ACTION_OPTIONS.map((option) => (
          <ToggleGroupItem key={option.value} value={option.value}>
            {option.label}
          </ToggleGroupItem>
        ))}
      </ToggleGroup>
      {description ? <FieldDescription>{description}</FieldDescription> : null}
      {action === "remount" ? (
        <FieldSet aria-invalid={flagsError ? true : undefined}>
          <FieldLegend variant="label">
            Mount Flags <span className="text-destructive">*</span>
          </FieldLegend>
          <FieldGroup data-slot="checkbox-group" className="grid gap-3 sm:grid-cols-2">
            {REMOUNT_FLAG_OPTIONS.map((option) => (
              <Field key={option.value} orientation="horizontal">
                <Checkbox
                  id={`${id}-flag-${option.value}`}
                  checked={flags.includes(option.value)}
                  onCheckedChange={(checked) =>
                    onFlagsChange(toggleRemountFlag(flags, option.value, checked === true))
                  }
                />
                <FieldLabel htmlFor={`${id}-flag-${option.value}`}>{option.label}</FieldLabel>
              </Field>
            ))}
          </FieldGroup>
        </FieldSet>
      ) : null}
      {flagsError ? <FieldError>{flagsError}</FieldError> : null}
    </Field>
  );
}

export function formFromConfiguration(configuration: SantaConfiguration): ConfigurationFormState {
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

function configurationBody(form: ConfigurationFormState): SantaConfigurationMutation {
  return {
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
  };
}

function removableMediaPolicyBody(action: SantaMediaAction, flags: SantaRemountFlag[]) {
  if (action === "none") return undefined;
  return { action, remount_flags: flags };
}

function toggleRemountFlag(flags: SantaRemountFlag[], flag: SantaRemountFlag, checked: boolean) {
  if (checked) return flags.includes(flag) ? flags : [...flags, flag];
  return flags.filter((value) => value !== flag);
}

function filterRemountFlags(flags: string[]) {
  return flags.filter((flag): flag is SantaRemountFlag =>
    (REMOUNT_FLAG_VALUES as readonly string[]).includes(flag),
  );
}
