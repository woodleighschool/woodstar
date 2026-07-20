import { FormField } from "@/components/form-field";
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldLabel,
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
import { Textarea } from "@/components/ui/textarea";
import { CLIENT_MODE_OPTIONS, FILE_ACCESS_ACTION_OPTIONS } from "@/lib/santa-configurations";

import type { ConfigurationEditorForm } from "./fields";
import { ConfigurationMediaFields } from "./media-fields";

export function ConfigurationOptionsFields({ form }: { form: ConfigurationEditorForm }) {
  return (
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
          <FormField field={field} label="Client Mode" htmlFor="santa-client-mode">
            {(control) => (
              <Select
                items={CLIENT_MODE_OPTIONS}
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
      <form.Field name="enable_bundles">
        {(field) => (
          <BoolField
            id="santa-enable-bundles"
            label="Bundles"
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
            value={field.state.value}
            onChange={field.handleChange}
          />
        )}
      </form.Field>
      <form.Field name="enable_all_event_upload">
        {(field) => (
          <BoolField
            id="santa-upload-all-events"
            label="Upload All Events"
            value={field.state.value}
            onChange={field.handleChange}
          />
        )}
      </form.Field>
      <form.Field name="disable_unknown_event_upload">
        {(field) => (
          <BoolField
            id="santa-disable-unknown-event-upload"
            label="Disable Unknown Event Upload"
            value={field.state.value}
            onChange={field.handleChange}
          />
        )}
      </form.Field>
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
          </Field>
        )}
      </form.Field>
      <div className="grid gap-4 md:grid-cols-2">
        <form.Field name="event_detail_url">
          {(field) => (
            <FormField field={field} label="Event Detail URL" htmlFor="santa-event-detail-url">
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
            </Field>
          )}
        </form.Field>
      </div>
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
