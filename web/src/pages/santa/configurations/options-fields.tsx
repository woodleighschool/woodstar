import { FormField } from "@/components/form-field";
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
  FieldTitle,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import {
  CLIENT_MODES,
  CLIENT_MODE_OPTIONS,
  CLIENT_MODE_VALUES,
  FILE_ACCESS_ACTIONS,
  FILE_ACCESS_ACTION_OPTIONS,
} from "@/lib/santa-configurations";
import { isOneOf } from "@/lib/utils";

import type { ConfigurationEditorForm } from "./fields";
import { ConfigurationMediaFields } from "./media-fields";

export function ConfigurationOptionsFields({ form }: { form: ConfigurationEditorForm }) {
  return (
    <FieldGroup className="max-w-3xl">
      <FieldSet>
        <FieldLegend>General</FieldLegend>
        <FieldGroup>
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
          <form.Field name="description">
            {(field) => (
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
          </form.Field>
          <form.Field name="client_mode">
            {(field) => (
              <FieldSet>
                <FieldLegend variant="label">Client Mode</FieldLegend>
                <RadioGroup
                  name={field.name}
                  value={field.state.value}
                  className="grid gap-2 md:grid-cols-3"
                  onValueChange={(value) => {
                    if (isOneOf(value, CLIENT_MODE_VALUES)) field.handleChange(value);
                  }}
                >
                  {CLIENT_MODE_OPTIONS.map((option) => (
                    <FieldLabel key={option.value} htmlFor={`santa-client-mode-${option.value}`}>
                      <Field orientation="horizontal">
                        <RadioGroupItem
                          id={`santa-client-mode-${option.value}`}
                          value={option.value}
                        />
                        <FieldContent>
                          <FieldTitle>{option.label}</FieldTitle>
                          <FieldDescription>
                            {CLIENT_MODES[option.value].description}
                          </FieldDescription>
                        </FieldContent>
                      </Field>
                    </FieldLabel>
                  ))}
                </RadioGroup>
              </FieldSet>
            )}
          </form.Field>
        </FieldGroup>
      </FieldSet>

      <FieldSet>
        <FieldLegend>Rule Evaluation</FieldLegend>
        <FieldGroup className="grid gap-4 md:grid-cols-2">
          <form.Field name="enable_bundles">
            {(field) => (
              <BoolField
                id="santa-enable-bundles"
                label="Bundle Scanning"
                description="Generate events for executables contained in a blocked application bundle."
                value={field.state.value}
                onChange={field.handleChange}
              />
            )}
          </form.Field>
          <form.Field name="enable_transitive_rules">
            {(field) => (
              <BoolField
                id="santa-enable-transitive-rules"
                label="Transitive Rules"
                description="Allow compiler rules to create allow rules for the executables they produce."
                value={field.state.value}
                onChange={field.handleChange}
              />
            )}
          </form.Field>
        </FieldGroup>
      </FieldSet>

      <FieldSet>
        <FieldLegend>Sync</FieldLegend>
        <FieldDescription>
          Controls how Santa exchanges rules and events with Woodstar.
        </FieldDescription>
        <FieldGroup>
          <div className="grid gap-4 md:grid-cols-2">
            <form.Field name="full_sync_interval_seconds">
              {(field) => (
                <FormField
                  field={field}
                  label="Full Sync Interval"
                  htmlFor="santa-full-sync-interval"
                  required
                  description="Maximum seconds between full syncs. Santa enforces a minimum of 60."
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
                <FormField
                  field={field}
                  label="Batch Size"
                  htmlFor="santa-batch-size"
                  required
                  description="Rules downloaded or events uploaded per request."
                >
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
          <div className="grid gap-4 md:grid-cols-2">
            <form.Field name="enable_all_event_upload">
              {(field) => (
                <BoolField
                  id="santa-upload-all-events"
                  label="Upload All Events"
                  description="Upload explicitly allowed executions in addition to Santa's usual event set."
                  value={field.state.value}
                  onChange={field.handleChange}
                />
              )}
            </form.Field>
            <form.Field name="disable_unknown_event_upload">
              {(field) => (
                <BoolField
                  id="santa-disable-unknown-event-upload"
                  label="Skip Unknown Events"
                  description="Do not upload unknown executions allowed by Monitor mode."
                  value={field.state.value}
                  onChange={field.handleChange}
                />
              )}
            </form.Field>
          </div>
        </FieldGroup>
      </FieldSet>

      <FieldSet>
        <FieldLegend>Execution and File Access</FieldLegend>
        <FieldGroup>
          <form.Field name="override_file_access_action">
            {(field) => (
              <FormField
                field={field}
                label="File Access Override"
                htmlFor="santa-file-access-override"
                required
                description={FILE_ACCESS_ACTIONS[field.state.value].description}
              >
                {(control) => (
                  <Select
                    items={FILE_ACCESS_ACTION_OPTIONS}
                    value={field.state.value}
                    onValueChange={(value) => {
                      if (value !== null) field.handleChange(value);
                    }}
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
          <form.Field name="allowed_path_regex">
            {(field) => (
              <Field>
                <FieldLabel htmlFor="santa-allowed-path-regex">Allowed Path Regex</FieldLabel>
                <Input
                  id="santa-allowed-path-regex"
                  name={field.name}
                  value={field.state.value}
                  onBlur={field.handleBlur}
                  onChange={(event) => field.handleChange(event.target.value)}
                />
                <FieldDescription>
                  ICU regex matched against the resolved absolute executable path. Prefer signed
                  rules where possible.
                </FieldDescription>
              </Field>
            )}
          </form.Field>
          <form.Field name="blocked_path_regex">
            {(field) => (
              <Field>
                <FieldLabel htmlFor="santa-blocked-path-regex">Blocked Path Regex</FieldLabel>
                <Input
                  id="santa-blocked-path-regex"
                  name={field.name}
                  value={field.state.value}
                  onBlur={field.handleBlur}
                  onChange={(event) => field.handleChange(event.target.value)}
                />
                <FieldDescription>
                  ICU regex matched against the resolved absolute executable path after
                  higher-priority rules.
                </FieldDescription>
              </Field>
            )}
          </form.Field>
        </FieldGroup>
      </FieldSet>

      <FieldSet>
        <FieldLegend>Block Notifications</FieldLegend>
        <FieldDescription>
          Configures the detail button shown in Santa execution block notifications.
        </FieldDescription>
        <FieldGroup className="grid gap-4 md:grid-cols-2">
          <form.Field name="event_detail_url">
            {(field) => (
              <FormField
                field={field}
                label="Event Detail URL"
                htmlFor="santa-event-detail-url"
                description="HTTPS URL template. Santa replaces event placeholders before opening it."
              >
                {(control) => (
                  <Input
                    {...control}
                    name={field.name}
                    value={field.state.value}
                    onBlur={field.handleBlur}
                    onChange={(event) => field.handleChange(event.target.value)}
                  />
                )}
              </FormField>
            )}
          </form.Field>
          <form.Field name="event_detail_text">
            {(field) => (
              <Field>
                <FieldLabel htmlFor="santa-event-detail-text">Event Detail Text</FieldLabel>
                <Input
                  id="santa-event-detail-text"
                  name={field.name}
                  value={field.state.value}
                  onBlur={field.handleBlur}
                  onChange={(event) => field.handleChange(event.target.value)}
                />
                <FieldDescription>Button label used with the event detail URL.</FieldDescription>
              </Field>
            )}
          </form.Field>
        </FieldGroup>
      </FieldSet>

      <ConfigurationMediaFields form={form} />
    </FieldGroup>
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
