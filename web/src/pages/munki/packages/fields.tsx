import { xml } from "@codemirror/lang-xml";
import { StreamLanguage } from "@codemirror/language";
import { shell } from "@codemirror/legacy-modes/mode/shell";
import type { Extension } from "@codemirror/state";
import { Link } from "@tanstack/react-router";
import { FileArchive, Plus, Trash2 } from "lucide-react";
import { useState, type ReactNode } from "react";

import { CodeEditor } from "@/components/editor/code-editor";
import { FormField } from "@/components/form-field";
import { SubmitButton } from "@/components/submit-button";
import { Button } from "@/components/ui/button";
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
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import type { MunkiPackage } from "@/hooks/use-munki-packages";
import type { PackageAlert } from "@/lib/api-client/types.gen";
import { requiredString } from "@/lib/form-validation";

import {
  MUNKI_INSTALLER_TYPE_OPTIONS,
  MUNKI_INSTALL_ITEM_TYPE_OPTIONS,
  MUNKI_RESTART_ACTION_OPTIONS,
  MUNKI_UNINSTALL_METHOD_OPTIONS,
} from "../software/munki-software";
import { type PackageEditorForm } from "./editor-form";
import {
  emptyInstallItemRow,
  emptyInstallerEnvironmentRow,
  emptyItemToCopyRow,
  emptyPackageReferenceRow,
  emptyReceiptRow,
  emptyStringRow,
  packageLabel,
  removeAt,
  replaceAt,
  scriptFields,
  toggleArray,
  type Architecture,
  type InstallItemRow,
  type InstallerEnvironmentRow,
  type ItemToCopyRow,
  type PackageFormState,
  type PackageReferenceRow,
  type ReceiptRow,
  type ScriptKey,
  type StringRow,
} from "./form-state";

const xmlExtensions: Extension[] = [xml()];
const shellExtensions: Extension[] = [StreamLanguage.define(shell)];

type PackageFieldName = keyof PackageFormState;

export type SoftwareInfo = {
  name: string;
  description: string;
  category: string;
  developer: string;
};

export function PackageEditorTabs({
  form,
  softwareInfo,
  packageOptions,
  installerFile,
  uninstallerFile,
  installerArtifactLocation,
  uninstallerArtifactLocation,
  onInstallerFileChange,
  onUninstallerFileChange,
}: {
  form: PackageEditorForm;
  softwareInfo: SoftwareInfo | null;
  packageOptions: MunkiPackage[];
  installerFile: File | null;
  uninstallerFile: File | null;
  installerArtifactLocation: string;
  uninstallerArtifactLocation: string;
  onInstallerFileChange: (file: File | null) => void;
  onUninstallerFileChange: (file: File | null) => void;
}) {
  return (
    <form.Subscribe selector={(state) => state.values}>
      {(values) => (
        <Tabs defaultValue="basic" className="max-w-6xl space-y-6">
          <TabsList className="grid h-auto w-full grid-cols-2 gap-1 md:grid-cols-4 lg:grid-cols-7">
            <TabsTrigger value="basic">Basic Info</TabsTrigger>
            <TabsTrigger value="contents">Contents</TabsTrigger>
            <TabsTrigger value="copy">Items to Copy</TabsTrigger>
            <TabsTrigger value="relationships">Relationships</TabsTrigger>
            <TabsTrigger value="installer">Installer</TabsTrigger>
            <TabsTrigger value="scripts">Scripts</TabsTrigger>
            <TabsTrigger value="advanced">Advanced</TabsTrigger>
          </TabsList>

          <TabsContent value="basic" className="mt-0">
            <BasicInfoTab form={form} softwareInfo={softwareInfo} />
          </TabsContent>

          <TabsContent value="contents" className="mt-0">
            <ContentsTab form={form} />
          </TabsContent>

          <TabsContent value="copy" className="mt-0">
            <ItemsToCopyTab form={form} />
          </TabsContent>

          <TabsContent value="relationships" className="mt-0">
            <RelationshipsTab form={form} packageOptions={packageOptions} />
          </TabsContent>

          <TabsContent value="installer" className="mt-0">
            <InstallerTab
              form={form}
              installerType={values.installer_type}
              uninstallMethod={values.uninstall_method}
              installerFile={installerFile}
              uninstallerFile={uninstallerFile}
              installerArtifactLocation={installerArtifactLocation}
              uninstallerArtifactLocation={uninstallerArtifactLocation}
              onInstallerFileChange={onInstallerFileChange}
              onUninstallerFileChange={onUninstallerFileChange}
            />
          </TabsContent>

          <TabsContent value="scripts" className="mt-0">
            <form.Subscribe
              selector={(state) => state.values}
              children={(values) => (
                <ScriptEditor values={values} onChange={(key, value) => form.setFieldValue(key, value)} />
              )}
            />
          </TabsContent>

          <TabsContent value="advanced" className="mt-0">
            <AdvancedTab form={form} />
          </TabsContent>
        </Tabs>
      )}
    </form.Subscribe>
  );
}

function BasicInfoTab({ form, softwareInfo }: { form: PackageEditorForm; softwareInfo: SoftwareInfo | null }) {
  return (
    <FieldGroup>
      <FieldSet>
        <FieldLegend>Package</FieldLegend>
        <FieldGroup className="grid gap-4 md:grid-cols-2">
          <Field>
            <FieldLabel htmlFor="munki-package-name">Name</FieldLabel>
            <Input id="munki-package-name" value={softwareInfo?.name ?? ""} disabled readOnly />
          </Field>
          <Field>
            <FieldLabel htmlFor="munki-package-category">Category</FieldLabel>
            <Input id="munki-package-category" value={softwareInfo?.category ?? ""} disabled readOnly />
          </Field>
          <Field>
            <FieldLabel htmlFor="munki-package-developer">Developer</FieldLabel>
            <Input id="munki-package-developer" value={softwareInfo?.developer ?? ""} disabled readOnly />
          </Field>
          <VersionField form={form} />
          <FormSelectField
            form={form}
            name="installer_type"
            id="munki-package-installer-type"
            label="Installer Type"
            options={MUNKI_INSTALLER_TYPE_OPTIONS}
          />
          <FormSelectField
            form={form}
            name="uninstall_method"
            id="munki-package-uninstall-method"
            label="Uninstall Method"
            options={MUNKI_UNINSTALL_METHOD_OPTIONS}
          />
          <FormSelectField
            form={form}
            name="restart_action"
            id="munki-package-restart-action"
            label="Restart Action"
            options={MUNKI_RESTART_ACTION_OPTIONS}
          />
          <FormTextField
            form={form}
            name="force_install_after_date"
            id="munki-package-force-install-after"
            label="Force Install After"
            type="datetime-local"
          />
        </FieldGroup>
        <Field>
          <FieldLabel htmlFor="munki-package-description">Description</FieldLabel>
          <Textarea id="munki-package-description" value={softwareInfo?.description ?? ""} disabled readOnly />
        </Field>
        <FormTextareaField form={form} name="notes" id="munki-package-notes" label="Notes" />
      </FieldSet>

      <FieldSet>
        <FieldLegend>Behavior</FieldLegend>
        <FieldGroup className="grid gap-4 md:grid-cols-3">
          <FormCheckboxField
            form={form}
            name="unattended_install"
            id="munki-package-unattended-install"
            label="Unattended install"
          />
          <FormCheckboxField
            form={form}
            name="unattended_uninstall"
            id="munki-package-unattended-uninstall"
            label="Unattended uninstall"
          />
          <FormCheckboxField form={form} name="on_demand" id="munki-package-on-demand" label="On demand" />
          <FormCheckboxField form={form} name="precache" id="munki-package-precache" label="Precache" />
          <FormCheckboxField form={form} name="autoremove" id="munki-package-autoremove" label="Autoremove" />
          <FormCheckboxField form={form} name="apple_item" id="munki-package-apple-item" label="Apple item" />
          <FormCheckboxField
            form={form}
            name="suppress_bundle_relocation"
            id="munki-package-suppress-bundle-relocation"
            label="Suppress bundle relocation"
          />
        </FieldGroup>
      </FieldSet>

      <FieldSet>
        <FieldLegend>Woodstar</FieldLegend>
        <FormCheckboxField form={form} name="eligible" id="munki-package-eligible" label="Available for targeting" />
      </FieldSet>
    </FieldGroup>
  );
}

function ContentsTab({ form }: { form: PackageEditorForm }) {
  return (
    <FieldGroup>
      <form.Field
        name="installs"
        children={(field) => (
          <InstallItemsEditor rows={field.state.value} onChange={(rows) => field.handleChange(rows)} />
        )}
      />
      <form.Field
        name="receipts"
        children={(field) => <ReceiptsEditor rows={field.state.value} onChange={(rows) => field.handleChange(rows)} />}
      />
    </FieldGroup>
  );
}

function ItemsToCopyTab({ form }: { form: PackageEditorForm }) {
  return (
    <form.Field
      name="items_to_copy"
      children={(field) => <ItemsToCopyEditor rows={field.state.value} onChange={(rows) => field.handleChange(rows)} />}
    />
  );
}

function RelationshipsTab({ form, packageOptions }: { form: PackageEditorForm; packageOptions: MunkiPackage[] }) {
  return (
    <FieldGroup>
      <form.Field
        name="requires"
        children={(field) => (
          <PackageReferenceEditor
            legend="Requires"
            addLabel="Requirement"
            rows={field.state.value}
            packageOptions={packageOptions}
            onChange={(rows) => field.handleChange(rows)}
          />
        )}
      />
      <form.Field
        name="update_for"
        children={(field) => (
          <PackageReferenceEditor
            legend="Update For"
            addLabel="Update Target"
            rows={field.state.value}
            packageOptions={packageOptions}
            onChange={(rows) => field.handleChange(rows)}
          />
        )}
      />
      <form.Field
        name="blocking_applications"
        children={(field) => (
          <StringArrayEditor
            legend="Blocking Applications"
            addLabel="Application"
            rows={field.state.value}
            onChange={(rows) => field.handleChange(rows)}
          />
        )}
      />
      <FieldSet>
        <FieldLegend>Blocking Application Handling</FieldLegend>
        <FieldGroup>
          <FormCheckboxField
            form={form}
            name="blocking_applications_manual_quit_only"
            id="munki-package-blocking-applications-manual-quit-only"
            label="Require manual quit"
          />
          <FormCodeField
            form={form}
            name="blocking_applications_quit_script"
            id="munki-package-blocking-applications-quit-script"
            label="Quit Script"
            minHeight="[&_.cm-content]:min-h-32"
          />
        </FieldGroup>
      </FieldSet>
      <form.Field
        name="supported_architectures"
        children={(field) => (
          <ArchitectureEditor values={field.state.value} onChange={(values) => field.handleChange(values)} />
        )}
      />
    </FieldGroup>
  );
}

function InstallerTab({
  form,
  installerType,
  uninstallMethod,
  installerFile,
  uninstallerFile,
  installerArtifactLocation,
  uninstallerArtifactLocation,
  onInstallerFileChange,
  onUninstallerFileChange,
}: {
  form: PackageEditorForm;
  installerType: PackageFormState["installer_type"];
  uninstallMethod: PackageFormState["uninstall_method"];
  installerFile: File | null;
  uninstallerFile: File | null;
  installerArtifactLocation: string;
  uninstallerArtifactLocation: string;
  onInstallerFileChange: (file: File | null) => void;
  onUninstallerFileChange: (file: File | null) => void;
}) {
  return (
    <FieldGroup>
      <FieldSet>
        <FieldLegend>Artifacts</FieldLegend>
        <FieldGroup>
          {installerType !== "nopkg" ? (
            <PackageFileField
              id="munki-package-installer-file"
              label="Installer"
              description={installerArtifactLocation || "No installer artifact selected."}
              icon={<FileArchive className="size-4" />}
              file={installerFile}
              onChange={onInstallerFileChange}
            />
          ) : null}
          {uninstallMethod === "uninstall_package" ? (
            <PackageFileField
              id="munki-package-uninstaller-file"
              label="Uninstaller"
              description={uninstallerArtifactLocation || "No uninstaller artifact selected."}
              icon={<FileArchive className="size-4" />}
              file={uninstallerFile}
              onChange={onUninstallerFileChange}
            />
          ) : null}
        </FieldGroup>
      </FieldSet>

      <FieldSet>
        <FieldLegend>Installer</FieldLegend>
        <FieldGroup>
          <FormTextField form={form} name="package_path" id="munki-package-package-path" label="Package Path" />
          <FormTextField
            form={form}
            name="installed_size"
            id="munki-package-installed-size"
            label="Installed Size"
            type="number"
            inputMode="numeric"
          />
          <form.Field
            name="installer_choices_xml"
            children={(field) => <XMLField value={field.state.value} onChange={(value) => field.handleChange(value)} />}
          />
          <form.Field
            name="installer_environment"
            children={(field) => (
              <InstallerEnvironmentEditor rows={field.state.value} onChange={(rows) => field.handleChange(rows)} />
            )}
          />
        </FieldGroup>
      </FieldSet>
    </FieldGroup>
  );
}

function ScriptEditor({
  values,
  onChange,
}: {
  values: Pick<PackageFormState, ScriptKey>;
  onChange: (key: ScriptKey, value: string) => void;
}) {
  const [active, setActive] = useState<ScriptKey>(scriptFields[0].key);
  const field = scriptFields.find((item) => item.key === active) ?? scriptFields[0];

  return (
    <FieldSet>
      <FieldLegend>Scripts</FieldLegend>

      <FieldGroup>
        <Field className="max-w-sm">
          <FieldLabel htmlFor="munki-package-script">Script</FieldLabel>
          <Select value={active} onValueChange={(value) => setActive(value as ScriptKey)}>
            <SelectTrigger id="munki-package-script" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {scriptFields.map((item) => (
                  <SelectItem key={item.key} value={item.key}>
                    <span className={values[item.key] !== "" ? "font-medium" : undefined}>{item.label}</span>
                    {values[item.key] !== "" ? (
                      <span className="bg-primary ml-auto size-1.5 shrink-0 rounded-full" aria-hidden />
                    ) : null}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
        </Field>

        <ScriptField label={field.label} value={values[field.key]} onChange={(value) => onChange(field.key, value)} />
      </FieldGroup>
    </FieldSet>
  );
}

function AdvancedTab({ form }: { form: PackageEditorForm }) {
  return (
    <FieldGroup>
      <FieldSet>
        <FieldLegend>Compatibility</FieldLegend>
        <FieldGroup className="grid gap-4 md:grid-cols-3">
          <FormTextField
            form={form}
            name="minimum_munki_version"
            id="munki-package-minimum-munki-version"
            label="Minimum Munki Version"
          />
          <FormTextField form={form} name="minimum_os_version" id="munki-package-minimum-os" label="Minimum OS" />
          <FormTextField form={form} name="maximum_os_version" id="munki-package-maximum-os" label="Maximum OS" />
        </FieldGroup>
        <FormCodeField
          form={form}
          name="installable_condition"
          id="munki-package-installable-condition"
          label="Installable Condition"
          minHeight="[&_.cm-content]:min-h-32"
        />
      </FieldSet>

      <form.Field
        name="preinstall_alert"
        children={(field) => (
          <AlertEditor
            id="munki-package-preinstall-alert"
            legend="Preinstall Alert"
            alert={field.state.value}
            onChange={(alert) => field.handleChange(alert)}
          />
        )}
      />
      <form.Field
        name="preuninstall_alert"
        children={(field) => (
          <AlertEditor
            id="munki-package-preuninstall-alert"
            legend="Preuninstall Alert"
            alert={field.state.value}
            onChange={(alert) => field.handleChange(alert)}
          />
        )}
      />
    </FieldGroup>
  );
}

export function PackageFormActions({ pending, error }: { pending: boolean; error?: string }) {
  return (
    <div className="flex flex-col gap-2">
      {error ? <FieldError>{error}</FieldError> : null}
      <div className="flex items-center gap-2">
        <SubmitButton pending={pending} size="sm">
          Save
        </SubmitButton>
        <Button asChild type="button" variant="outline" size="sm">
          <Link to="/munki/packages">Cancel</Link>
        </Button>
      </div>
    </div>
  );
}

function VersionField({ form }: { form: PackageEditorForm }) {
  return (
    <form.Field name="version" validators={{ onSubmit: requiredString("Version") }}>
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

function FormTextField({
  form,
  name,
  id,
  label,
  required,
  type = "text",
  inputMode,
}: {
  form: PackageEditorForm;
  name: PackageFieldName;
  id: string;
  label: string;
  required?: boolean;
  type?: string;
  inputMode?: "text" | "numeric" | "decimal" | "tel" | "search" | "email" | "url";
}) {
  return (
    <form.Field
      name={name as never}
      children={(field) => (
        <Field>
          <FieldLabel htmlFor={id} required={required}>
            {label}
          </FieldLabel>
          <Input
            id={id}
            name={field.name}
            type={type}
            inputMode={inputMode}
            value={typeof field.state.value === "string" ? field.state.value : ""}
            onBlur={field.handleBlur}
            onChange={(event) => field.handleChange(event.target.value as never)}
          />
        </Field>
      )}
    />
  );
}

function FormTextareaField({
  form,
  name,
  id,
  label,
}: {
  form: PackageEditorForm;
  name: PackageFieldName;
  id: string;
  label: string;
}) {
  return (
    <form.Field
      name={name as never}
      children={(field) => (
        <Field>
          <FieldLabel htmlFor={id}>{label}</FieldLabel>
          <Textarea
            id={id}
            name={field.name}
            value={typeof field.state.value === "string" ? field.state.value : ""}
            onBlur={field.handleBlur}
            onChange={(event) => field.handleChange(event.target.value as never)}
          />
        </Field>
      )}
    />
  );
}

function FormCodeField({
  form,
  name,
  id,
  label,
  minHeight = "[&_.cm-content]:min-h-40",
}: {
  form: PackageEditorForm;
  name: PackageFieldName;
  id: string;
  label: string;
  minHeight?: string;
}) {
  return (
    <form.Field
      name={name as never}
      children={(field) => (
        <Field>
          <FieldLabel htmlFor={id}>{label}</FieldLabel>
          <CodeEditor
            value={typeof field.state.value === "string" ? field.state.value : ""}
            onChange={(value) => field.handleChange(value as never)}
            className={minHeight}
          />
        </Field>
      )}
    />
  );
}

function FormSelectField<T extends string>({
  form,
  name,
  id,
  label,
  options,
}: {
  form: PackageEditorForm;
  name: PackageFieldName;
  id: string;
  label: string;
  options: Array<{ value: T; label: string }>;
}) {
  return (
    <form.Field
      name={name as never}
      children={(field) => (
        <Field>
          <FieldLabel htmlFor={id}>{label}</FieldLabel>
          <Select
            value={String((field.state.value as string | undefined) ?? "")}
            onValueChange={(next) => field.handleChange(next as never)}
          >
            <SelectTrigger id={id} className="w-full">
              <SelectValue />
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
        </Field>
      )}
    />
  );
}

function FormCheckboxField({
  form,
  name,
  id,
  label,
}: {
  form: PackageEditorForm;
  name: PackageFieldName;
  id: string;
  label: string;
}) {
  return (
    <form.Field
      name={name as never}
      children={(field) => (
        <CheckboxControl
          id={id}
          label={label}
          checked={field.state.value === true}
          onChange={(checked) => field.handleChange(checked as never)}
        />
      )}
    />
  );
}

function PackageFileField({
  id,
  label,
  description,
  accept,
  icon,
  file,
  onChange,
}: {
  id: string;
  label: string;
  description: string;
  accept?: string;
  icon: ReactNode;
  file: File | null;
  onChange: (file: File | null) => void;
}) {
  return (
    <Field>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <div className="flex items-center gap-3">
        <div className="bg-muted text-muted-foreground flex size-9 shrink-0 items-center justify-center rounded-md border">
          {icon}
        </div>
        <Input id={id} type="file" accept={accept} onChange={(event) => onChange(event.target.files?.[0] ?? null)} />
      </div>
      <FieldDescription>{file ? file.name : description}</FieldDescription>
    </Field>
  );
}

function CheckboxControl({
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
      <Checkbox id={id} checked={checked} disabled={disabled} onCheckedChange={(value) => onChange(value === true)} />
      <FieldContent>
        <FieldLabel htmlFor={id}>{label}</FieldLabel>
        {description ? <FieldDescription>{description}</FieldDescription> : null}
      </FieldContent>
    </Field>
  );
}

function XMLField({ value, onChange }: { value: string; onChange: (value: string) => void }) {
  return (
    <Field>
      <FieldLabel>Installer Choices XML</FieldLabel>
      <CodeEditor
        value={value}
        onChange={onChange}
        extensions={xmlExtensions}
        lineNumbers={false}
        className="[&_.cm-content]:min-h-28"
      />
    </Field>
  );
}

function ArchitectureEditor({
  values,
  onChange,
}: {
  values: Architecture[];
  onChange: (values: Architecture[]) => void;
}) {
  return (
    <FieldSet>
      <FieldLegend>Supported Architectures</FieldLegend>
      <FieldGroup className="grid gap-4 md:grid-cols-2">
        <CheckboxControl
          id="munki-package-arch-arm64"
          label="Apple silicon"
          checked={values.includes("arm64")}
          onChange={(checked) => onChange(toggleArray(values, "arm64", checked))}
        />
        <CheckboxControl
          id="munki-package-arch-x86"
          label="Intel"
          checked={values.includes("x86_64")}
          onChange={(checked) => onChange(toggleArray(values, "x86_64", checked))}
        />
      </FieldGroup>
    </FieldSet>
  );
}

function StringArrayEditor({
  legend,
  addLabel,
  rows,
  onChange,
}: {
  legend: string;
  addLabel: string;
  rows: StringRow[];
  onChange: (rows: StringRow[]) => void;
}) {
  return (
    <FieldSet>
      <FieldLegend>{legend}</FieldLegend>
      <div className="space-y-2">
        {rows.map((row, index) => (
          <div key={row.rowID} className="grid gap-2 md:grid-cols-[minmax(0,1fr)_auto]">
            <Input
              value={row.value}
              onChange={(event) => onChange(replaceAt(rows, index, { ...row, value: event.target.value }))}
            />
            <IconButton label="Remove" onClick={() => onChange(removeAt(rows, index))}>
              <Trash2 />
            </IconButton>
          </div>
        ))}
        <Button type="button" variant="outline" size="sm" onClick={() => onChange([...rows, emptyStringRow()])}>
          <Plus data-icon="inline-start" />
          {addLabel}
        </Button>
      </div>
    </FieldSet>
  );
}

function PackageReferenceEditor({
  legend,
  addLabel,
  rows,
  packageOptions,
  onChange,
}: {
  legend: string;
  addLabel: string;
  rows: PackageReferenceRow[];
  packageOptions: MunkiPackage[];
  onChange: (rows: PackageReferenceRow[]) => void;
}) {
  return (
    <FieldSet>
      <FieldLegend>{legend}</FieldLegend>
      <div className="space-y-2">
        {rows.map((row, index) => (
          <div key={row.rowID} className="grid gap-2 md:grid-cols-[minmax(0,1fr)_auto]">
            <Select
              value={row.package_id ? String(row.package_id) : "select"}
              onValueChange={(value) =>
                onChange(replaceAt(rows, index, { ...row, package_id: value === "select" ? undefined : Number(value) }))
              }
            >
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value="select">Select package</SelectItem>
                  {packageOptions.map((option) => (
                    <SelectItem key={option.id} value={String(option.id)}>
                      {packageLabel(option)}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
            <IconButton label="Remove" onClick={() => onChange(removeAt(rows, index))}>
              <Trash2 />
            </IconButton>
          </div>
        ))}
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => onChange([...rows, emptyPackageReferenceRow()])}
        >
          <Plus data-icon="inline-start" />
          {addLabel}
        </Button>
      </div>
    </FieldSet>
  );
}

function InstallerEnvironmentEditor({
  rows,
  onChange,
}: {
  rows: InstallerEnvironmentRow[];
  onChange: (rows: InstallerEnvironmentRow[]) => void;
}) {
  return (
    <FieldSet>
      <FieldLegend>Installer Environment</FieldLegend>
      <div className="space-y-2">
        {rows.map((row, index) => (
          <div key={row.rowID} className="grid gap-2 md:grid-cols-[minmax(0,12rem)_minmax(0,1fr)_auto]">
            <Input
              aria-label="Name"
              value={row.name}
              onChange={(event) => onChange(replaceAt(rows, index, { ...row, name: event.target.value }))}
            />
            <Input
              aria-label="Value"
              value={row.value}
              onChange={(event) => onChange(replaceAt(rows, index, { ...row, value: event.target.value }))}
            />
            <IconButton label="Remove" onClick={() => onChange(removeAt(rows, index))}>
              <Trash2 />
            </IconButton>
          </div>
        ))}
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => onChange([...rows, emptyInstallerEnvironmentRow()])}
        >
          <Plus data-icon="inline-start" />
          Variable
        </Button>
      </div>
    </FieldSet>
  );
}

function InstallItemsEditor({
  rows,
  onChange,
}: {
  rows: InstallItemRow[];
  onChange: (rows: InstallItemRow[]) => void;
}) {
  return (
    <FieldSet>
      <FieldLegend>Installs</FieldLegend>
      <div className="space-y-4">
        {rows.map((row, index) => (
          <div key={row.rowID} className="space-y-3 rounded-md border p-3">
            <div className="grid gap-3 md:grid-cols-[10rem_minmax(0,1fr)_auto]">
              <InstallItemTypeField
                id={`munki-install-item-type-${row.rowID}`}
                value={row.type}
                onChange={(type) => onChange(replaceAt(rows, index, { ...row, type }))}
              />
              <Field>
                <FieldLabel htmlFor={`munki-install-item-path-${row.rowID}`}>Path</FieldLabel>
                <Input
                  id={`munki-install-item-path-${row.rowID}`}
                  value={row.path}
                  onChange={(event) => onChange(replaceAt(rows, index, { ...row, path: event.target.value }))}
                />
              </Field>
              <div className="flex items-end justify-end">
                <IconButton label="Remove" onClick={() => onChange(removeAt(rows, index))}>
                  <Trash2 />
                </IconButton>
              </div>
            </div>
            <FieldGroup className="grid gap-3 md:grid-cols-3">
              <Field>
                <FieldLabel htmlFor={`munki-install-item-bundle-id-${row.rowID}`}>Bundle ID</FieldLabel>
                <Input
                  id={`munki-install-item-bundle-id-${row.rowID}`}
                  value={row.bundle_identifier ?? ""}
                  onChange={(event) =>
                    onChange(replaceAt(rows, index, { ...row, bundle_identifier: event.target.value }))
                  }
                />
              </Field>
              <Field>
                <FieldLabel htmlFor={`munki-install-item-short-version-${row.rowID}`}>Short Version</FieldLabel>
                <Input
                  id={`munki-install-item-short-version-${row.rowID}`}
                  value={row.bundle_short_version ?? ""}
                  onChange={(event) =>
                    onChange(replaceAt(rows, index, { ...row, bundle_short_version: event.target.value }))
                  }
                />
              </Field>
              <Field>
                <FieldLabel htmlFor={`munki-install-item-version-${row.rowID}`}>Bundle Version</FieldLabel>
                <Input
                  id={`munki-install-item-version-${row.rowID}`}
                  value={row.bundle_version ?? ""}
                  onChange={(event) => onChange(replaceAt(rows, index, { ...row, bundle_version: event.target.value }))}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor={`munki-install-item-comparison-${row.rowID}`}>Comparison Key</FieldLabel>
                <Input
                  id={`munki-install-item-comparison-${row.rowID}`}
                  value={row.version_comparison_key ?? ""}
                  onChange={(event) =>
                    onChange(replaceAt(rows, index, { ...row, version_comparison_key: event.target.value }))
                  }
                />
              </Field>
              <Field>
                <FieldLabel htmlFor={`munki-install-item-md5-${row.rowID}`}>MD5</FieldLabel>
                <Input
                  id={`munki-install-item-md5-${row.rowID}`}
                  value={row.md5checksum ?? ""}
                  onChange={(event) => onChange(replaceAt(rows, index, { ...row, md5checksum: event.target.value }))}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor={`munki-install-item-min-os-${row.rowID}`}>Minimum OS</FieldLabel>
                <Input
                  id={`munki-install-item-min-os-${row.rowID}`}
                  value={row.minimum_os_version ?? ""}
                  onChange={(event) =>
                    onChange(replaceAt(rows, index, { ...row, minimum_os_version: event.target.value }))
                  }
                />
              </Field>
            </FieldGroup>
          </div>
        ))}
        <Button type="button" variant="outline" size="sm" onClick={() => onChange([...rows, emptyInstallItemRow()])}>
          <Plus data-icon="inline-start" />
          Install Item
        </Button>
      </div>
    </FieldSet>
  );
}

function InstallItemTypeField({
  id,
  value,
  onChange,
}: {
  id: string;
  value: InstallItemRow["type"];
  onChange: (value: InstallItemRow["type"]) => void;
}) {
  return (
    <Field>
      <FieldLabel htmlFor={id}>Type</FieldLabel>
      <Select value={value} onValueChange={(next) => onChange(next as InstallItemRow["type"])}>
        <SelectTrigger id={id} className="w-full">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            {MUNKI_INSTALL_ITEM_TYPE_OPTIONS.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
    </Field>
  );
}

function ReceiptsEditor({ rows, onChange }: { rows: ReceiptRow[]; onChange: (rows: ReceiptRow[]) => void }) {
  return (
    <FieldSet>
      <FieldLegend>Receipts</FieldLegend>
      <div className="space-y-2">
        {rows.map((row, index) => (
          <div key={row.rowID} className="grid gap-2 md:grid-cols-[minmax(0,1fr)_10rem_8rem_auto]">
            <Input
              aria-label="Package ID"
              placeholder="Package ID"
              value={row.package_id}
              onChange={(event) => onChange(replaceAt(rows, index, { ...row, package_id: event.target.value }))}
            />
            <Input
              aria-label="Version"
              placeholder="Version"
              value={row.version ?? ""}
              onChange={(event) => onChange(replaceAt(rows, index, { ...row, version: event.target.value }))}
            />
            <CheckboxControl
              id={`munki-receipt-optional-${row.rowID}`}
              label="Optional"
              checked={row.optional === true}
              onChange={(checked) => onChange(replaceAt(rows, index, { ...row, optional: checked }))}
            />
            <IconButton label="Remove" onClick={() => onChange(removeAt(rows, index))}>
              <Trash2 />
            </IconButton>
          </div>
        ))}
        <Button type="button" variant="outline" size="sm" onClick={() => onChange([...rows, emptyReceiptRow()])}>
          <Plus data-icon="inline-start" />
          Receipt
        </Button>
      </div>
    </FieldSet>
  );
}

function ItemsToCopyEditor({ rows, onChange }: { rows: ItemToCopyRow[]; onChange: (rows: ItemToCopyRow[]) => void }) {
  return (
    <FieldSet>
      <FieldLegend>Items to Copy</FieldLegend>
      <div className="space-y-4">
        {rows.map((row, index) => (
          <div key={row.rowID} className="space-y-3 rounded-md border p-3">
            <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto]">
              <Field>
                <FieldLabel htmlFor={`munki-copy-source-${row.rowID}`}>Source Item</FieldLabel>
                <Input
                  id={`munki-copy-source-${row.rowID}`}
                  value={row.source_item}
                  onChange={(event) => onChange(replaceAt(rows, index, { ...row, source_item: event.target.value }))}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor={`munki-copy-destination-${row.rowID}`}>Destination Path</FieldLabel>
                <Input
                  id={`munki-copy-destination-${row.rowID}`}
                  value={row.destination_path}
                  onChange={(event) =>
                    onChange(replaceAt(rows, index, { ...row, destination_path: event.target.value }))
                  }
                />
              </Field>
              <div className="flex items-end justify-end">
                <IconButton label="Remove" onClick={() => onChange(removeAt(rows, index))}>
                  <Trash2 />
                </IconButton>
              </div>
            </div>
            <FieldGroup className="grid gap-3 md:grid-cols-4">
              <Field>
                <FieldLabel htmlFor={`munki-copy-destination-item-${row.rowID}`}>Destination Item</FieldLabel>
                <Input
                  id={`munki-copy-destination-item-${row.rowID}`}
                  value={row.destination_item ?? ""}
                  onChange={(event) =>
                    onChange(replaceAt(rows, index, { ...row, destination_item: event.target.value }))
                  }
                />
              </Field>
              <Field>
                <FieldLabel htmlFor={`munki-copy-user-${row.rowID}`}>User</FieldLabel>
                <Input
                  id={`munki-copy-user-${row.rowID}`}
                  value={row.user ?? ""}
                  onChange={(event) => onChange(replaceAt(rows, index, { ...row, user: event.target.value }))}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor={`munki-copy-group-${row.rowID}`}>Group</FieldLabel>
                <Input
                  id={`munki-copy-group-${row.rowID}`}
                  value={row.group ?? ""}
                  onChange={(event) => onChange(replaceAt(rows, index, { ...row, group: event.target.value }))}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor={`munki-copy-mode-${row.rowID}`}>Mode</FieldLabel>
                <Input
                  id={`munki-copy-mode-${row.rowID}`}
                  value={row.mode ?? ""}
                  onChange={(event) => onChange(replaceAt(rows, index, { ...row, mode: event.target.value }))}
                />
              </Field>
            </FieldGroup>
          </div>
        ))}
        <Button type="button" variant="outline" size="sm" onClick={() => onChange([...rows, emptyItemToCopyRow()])}>
          <Plus data-icon="inline-start" />
          Copy Item
        </Button>
      </div>
    </FieldSet>
  );
}

function ScriptField({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return (
    <Field>
      <FieldLabel>{label}</FieldLabel>
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

function AlertEditor({
  id,
  legend,
  alert,
  onChange,
}: {
  id: string;
  legend: string;
  alert: PackageAlert;
  onChange: (alert: PackageAlert) => void;
}) {
  return (
    <FieldSet>
      <FieldLegend>{legend}</FieldLegend>
      <FieldGroup>
        <CheckboxControl
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

function IconButton({ label, children, onClick }: { label: string; children: ReactNode; onClick: () => void }) {
  return (
    <Button type="button" variant="ghost" size="icon-sm" title={label} onClick={onClick}>
      {children}
    </Button>
  );
}
