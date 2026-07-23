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
import type { MunkiPackageAlert } from "@/lib/api";
import { isOneOf } from "@/lib/utils";
import {
  MUNKI_INSTALLER_TYPES,
  MUNKI_INSTALLER_TYPE_OPTIONS,
  MUNKI_INSTALLER_TYPE_VALUES,
  MUNKI_RESTART_ACTION_OPTIONS,
  MUNKI_UNINSTALL_METHOD_OPTIONS,
} from "@/pages/munki/software/munki-software";

import type { PackageEditorForm } from "./fields";
import type { PackageFormInput } from "./form-schema";

const shellExtensions = [StreamLanguage.define(shell)];

type PackageFieldNameByValue<T> = {
  [K in keyof PackageFormInput]: PackageFormInput[K] extends T ? K : never;
}[keyof PackageFormInput];
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
  description,
  required,
  type = "text",
  inputMode,
}: {
  form: PackageEditorForm;
  name: StringPackageFieldName;
  id: string;
  label: string;
  description?: string;
  required?: boolean;
  type?: string;
  inputMode?: "text" | "numeric" | "decimal" | "tel" | "search" | "email" | "url";
}) {
  return (
    <form.Field name={name}>
      {(field) => (
        <FormField
          field={field}
          label={label}
          htmlFor={id}
          required={required}
          description={description}
        >
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
  description,
}: {
  form: PackageEditorForm;
  name: StringPackageFieldName;
  id: string;
  label: string;
  description?: string;
}) {
  return (
    <form.Field name={name}>
      {(field) => (
        <FormField field={field} label={label} htmlFor={id} description={description}>
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
  description,
  minHeight = "[&_.cm-content]:min-h-40",
}: {
  form: PackageEditorForm;
  name: StringPackageFieldName;
  id: string;
  label: string;
  description?: string;
  minHeight?: string;
}) {
  return (
    <form.Field name={name}>
      {(field) => (
        <FormField field={field} label={label} htmlFor={id} description={description}>
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

export function InstallerTypeField({ form }: { form: PackageEditorForm }) {
  return (
    <form.Field name="installer_type">
      {(field) => (
        <FieldSet>
          <FieldLegend variant="label">Installer Type</FieldLegend>
          <RadioGroup
            name={field.name}
            value={field.state.value}
            className="grid gap-2 md:grid-cols-3"
            onValueChange={(value) => {
              if (isOneOf(value, MUNKI_INSTALLER_TYPE_VALUES)) field.handleChange(value);
            }}
          >
            {MUNKI_INSTALLER_TYPE_OPTIONS.map((option) => (
              <FieldLabel key={option.value} htmlFor={`munki-installer-type-${option.value}`}>
                <Field orientation="horizontal">
                  <RadioGroupItem
                    id={`munki-installer-type-${option.value}`}
                    value={option.value}
                  />
                  <FieldContent>
                    <FieldTitle>{option.label}</FieldTitle>
                    <FieldDescription>
                      {MUNKI_INSTALLER_TYPES[option.value].description}
                    </FieldDescription>
                  </FieldContent>
                </Field>
              </FieldLabel>
            ))}
          </RadioGroup>
        </FieldSet>
      )}
    </form.Field>
  );
}

export function RestartActionField({ form }: { form: PackageEditorForm }) {
  return (
    <form.Field name="restart_action">
      {(field) => (
        <FormField field={field} label="Restart Action" htmlFor="munki-package-restart-action">
          {() => (
            <Select
              items={MUNKI_RESTART_ACTION_OPTIONS}
              value={field.state.value}
              onValueChange={(value) => {
                if (value !== null) field.handleChange(value);
              }}
            >
              <SelectTrigger id="munki-package-restart-action" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {MUNKI_RESTART_ACTION_OPTIONS.map((option) => (
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

const uninstallMethodItems = [
  { value: null, label: "Select a method" },
  ...MUNKI_UNINSTALL_METHOD_OPTIONS,
];

export function UninstallMethodField({ form }: { form: PackageEditorForm }) {
  return (
    <form.Field name="uninstall_method">
      {(field) => (
        <FormField
          field={field}
          label="Uninstall Method"
          htmlFor="munki-package-uninstall-method"
          description="How Munki removes this package from a client."
        >
          {() => (
            <Select
              items={uninstallMethodItems}
              value={field.state.value || null}
              onValueChange={(value) => field.handleChange(value ?? "")}
            >
              <SelectTrigger id="munki-package-uninstall-method" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {MUNKI_UNINSTALL_METHOD_OPTIONS.map((option) => (
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
  description,
}: {
  form: PackageEditorForm;
  name: BooleanPackageFieldName;
  id: string;
  label: string;
  description?: string;
}) {
  return (
    <form.Field name={name}>
      {(field) => (
        <SwitchControl
          id={id}
          label={label}
          description={description}
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
  description,
}: {
  form: PackageEditorForm;
  name: BooleanPackageFieldName;
  id: string;
  label: string;
  description?: string;
}) {
  return (
    <form.Field name={name}>
      {(field) => (
        <CheckboxControl
          id={id}
          label={label}
          description={description}
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
        className="[&_.cm-content]:min-h-56 [&_.cm-scroller]:max-h-120 [&_.cm-scroller]:overflow-y-auto"
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
