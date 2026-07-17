import { StreamLanguage } from "@codemirror/language";
import { shell } from "@codemirror/legacy-modes/mode/shell";

import { CodeEditor } from "@/components/editor/code-editor";
import { FormField } from "@/components/form-field";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Field,
  FieldContent,
  FieldDescription,
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
import { Textarea } from "@/components/ui/textarea";
import type { MunkiPackageAlert } from "@/lib/api";

import type { PackageEditorForm } from "./editor-form";
import type { PackageFormState } from "./form-state";

const shellExtensions = [StreamLanguage.define(shell)];

type PackageFieldNameByValue<T> = {
  [K in keyof PackageFormState]: PackageFormState[K] extends T ? K : never;
}[keyof PackageFormState];
type StringPackageFieldName = PackageFieldNameByValue<string>;
type BooleanPackageFieldName = PackageFieldNameByValue<boolean>;

export function VersionField({ form }: { form: PackageEditorForm }) {
  return (
    <form.Field name="version">
      {(field) => (
        <FormField field={field} label="Version" htmlFor="munki-package-version" required>
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
  );
}

export function FormTextField({
  form,
  name,
  id,
  label,
  required,
  type = "text",
  inputMode,
}: {
  form: PackageEditorForm;
  name: StringPackageFieldName;
  id: string;
  label: string;
  required?: boolean;
  type?: string;
  inputMode?: "text" | "numeric" | "decimal" | "tel" | "search" | "email" | "url";
}) {
  return (
    <form.Field name={name}>
      {(field) => (
        <FormField field={field} label={label} htmlFor={id} required={required}>
          {(control) => (
            <Input
              {...control}
              id={id}
              name={field.name}
              type={type}
              inputMode={inputMode}
              value={field.state.value}
              onBlur={field.handleBlur}
              onChange={(event) => field.handleChange(event.target.value)}
            />
          )}
        </FormField>
      )}
    </form.Field>
  );
}

export function FormTextareaField({
  form,
  name,
  id,
  label,
}: {
  form: PackageEditorForm;
  name: StringPackageFieldName;
  id: string;
  label: string;
}) {
  return (
    <form.Field name={name}>
      {(field) => (
        <FormField field={field} label={label} htmlFor={id}>
          {(control) => (
            <Textarea
              {...control}
              id={id}
              name={field.name}
              value={field.state.value}
              onBlur={field.handleBlur}
              onChange={(event) => field.handleChange(event.target.value)}
            />
          )}
        </FormField>
      )}
    </form.Field>
  );
}

export function FormCodeField({
  form,
  name,
  id,
  label,
  minHeight = "[&_.cm-content]:min-h-40",
}: {
  form: PackageEditorForm;
  name: StringPackageFieldName;
  id: string;
  label: string;
  minHeight?: string;
}) {
  return (
    <form.Field name={name}>
      {(field) => (
        <FormField field={field} label={label} htmlFor={id}>
          {() => (
            <CodeEditor
              value={field.state.value}
              onChange={field.handleChange}
              className={minHeight}
            />
          )}
        </FormField>
      )}
    </form.Field>
  );
}

export function FormSelectField<
  Name extends StringPackageFieldName,
  T extends PackageFormState[Name] & string,
>({
  form,
  name,
  id,
  label,
  options,
  placeholder,
}: {
  form: PackageEditorForm;
  name: Name;
  id: string;
  label: string;
  options: Array<{ value: T; label: string }>;
  placeholder?: string;
}) {
  return (
    <form.Field name={name}>
      {(field) => (
        <FormField field={field} label={label} htmlFor={id}>
          {() => (
            <Select
              items={options}
              value={field.state.value === "" ? null : (field.state.value as unknown as T)}
              onValueChange={(next) =>
                field.handleChange(next as unknown as Parameters<typeof field.handleChange>[0])
              }
            >
              <SelectTrigger id={id} className="w-full">
                <SelectValue placeholder={placeholder} />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {options.map((option) => (
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
  );
}

export function FormSwitchField({
  form,
  name,
  id,
  label,
}: {
  form: PackageEditorForm;
  name: BooleanPackageFieldName;
  id: string;
  label: string;
}) {
  return (
    <form.Field name={name}>
      {(field) => (
        <SwitchControl
          id={id}
          label={label}
          checked={field.state.value}
          onChange={field.handleChange}
        />
      )}
    </form.Field>
  );
}

export function FormCheckboxField({
  form,
  name,
  id,
  label,
}: {
  form: PackageEditorForm;
  name: BooleanPackageFieldName;
  id: string;
  label: string;
}) {
  return (
    <form.Field name={name}>
      {(field) => (
        <CheckboxControl
          id={id}
          label={label}
          checked={field.state.value}
          onChange={field.handleChange}
        />
      )}
    </form.Field>
  );
}

export function CheckboxControl({
  id,
  label,
  description,
  checked,
  disabled,
  onChange,
}: {
  id: string;
  label: string;
  description?: string;
  checked: boolean;
  disabled?: boolean;
  onChange: (checked: boolean) => void;
}) {
  return (
    <Field orientation="horizontal" className={disabled ? "opacity-60" : undefined}>
      <Checkbox
        id={id}
        checked={checked}
        disabled={disabled}
        onCheckedChange={(value) => onChange(value)}
      />
      <FieldContent>
        <FieldLabel htmlFor={id}>{label}</FieldLabel>
        {description ? <FieldDescription>{description}</FieldDescription> : null}
      </FieldContent>
    </Field>
  );
}

export function SwitchControl({
  id,
  label,
  description,
  checked,
  disabled,
  onChange,
}: {
  id: string;
  label: string;
  description?: string;
  checked: boolean;
  disabled?: boolean;
  onChange: (checked: boolean) => void;
}) {
  return (
    <Field orientation="horizontal" className={disabled ? "opacity-60" : undefined}>
      <FieldContent>
        <FieldLabel htmlFor={id}>{label}</FieldLabel>
        {description ? <FieldDescription>{description}</FieldDescription> : null}
      </FieldContent>
      <Switch id={id} checked={checked} disabled={disabled} onCheckedChange={onChange} />
    </Field>
  );
}

export function ScriptField({
  label,
  value,
  onChange,
}: {
  label?: string;
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <Field>
      {label ? <FieldLabel>{label}</FieldLabel> : null}
      <CodeEditor
        value={value}
        onChange={onChange}
        extensions={shellExtensions}
        className="[&_.cm-content]:min-h-56 [&_.cm-scroller]:max-h-[30rem] [&_.cm-scroller]:overflow-y-auto"
        placeholder="#!/bin/zsh"
      />
    </Field>
  );
}

export function AlertEditor({
  id,
  legend,
  alert,
  onChange,
}: {
  id: string;
  legend: string;
  alert: MunkiPackageAlert;
  onChange: (alert: MunkiPackageAlert) => void;
}) {
  return (
    <FieldSet>
      <FieldLegend>{legend}</FieldLegend>
      <FieldGroup>
        <SwitchControl
          id={`${id}-enabled`}
          label="Enabled"
          checked={alert.enabled}
          onChange={(enabled) => onChange({ ...alert, enabled })}
        />
        {alert.enabled ? (
          <FieldGroup className="grid gap-4 md:grid-cols-2">
            <Field>
              <FieldLabel htmlFor={`${id}-title`}>Title</FieldLabel>
              <Input
                id={`${id}-title`}
                value={alert.title ?? ""}
                onChange={(event) => onChange({ ...alert, title: event.target.value })}
              />
            </Field>
            <Field>
              <FieldLabel htmlFor={`${id}-ok`}>OK Label</FieldLabel>
              <Input
                id={`${id}-ok`}
                value={alert.ok_label ?? ""}
                onChange={(event) => onChange({ ...alert, ok_label: event.target.value })}
              />
            </Field>
            <Field>
              <FieldLabel htmlFor={`${id}-cancel`}>Cancel Label</FieldLabel>
              <Input
                id={`${id}-cancel`}
                value={alert.cancel_label ?? ""}
                onChange={(event) => onChange({ ...alert, cancel_label: event.target.value })}
              />
            </Field>
            <Field className="md:col-span-2">
              <FieldLabel htmlFor={`${id}-detail`}>Detail</FieldLabel>
              <Textarea
                id={`${id}-detail`}
                value={alert.detail ?? ""}
                onChange={(event) => onChange({ ...alert, detail: event.target.value })}
              />
            </Field>
          </FieldGroup>
        ) : null}
      </FieldGroup>
    </FieldSet>
  );
}
