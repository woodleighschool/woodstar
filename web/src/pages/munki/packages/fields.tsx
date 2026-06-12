import { StreamLanguage } from "@codemirror/language";
import { shell } from "@codemirror/legacy-modes/mode/shell";
import type { Extension } from "@codemirror/state";
import { Link } from "@tanstack/react-router";
import { FileArchive, Plus, Trash2 } from "lucide-react";
import { type ComponentProps, type ReactNode, useState } from "react";

import { CodeEditor } from "@/components/editor/code-editor";
import { EmptyPanel } from "@/components/empty-panel";
import { FormField } from "@/components/form-field";
import { ScrollableTabs } from "@/components/layout/scrollable-tabs";
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
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import type { MunkiPackage } from "@/hooks/use-munki-packages";
import { useTabIndicator } from "@/hooks/use-tab-indicator";
import type { PackageAlert } from "@/lib/api";
import { requiredString } from "@/lib/form-validation";
import { cn } from "@/lib/utils";

import {
  MUNKI_INSTALL_ITEM_TYPE_OPTIONS,
  MUNKI_INSTALLER_TYPE_OPTIONS,
  MUNKI_RESTART_ACTION_OPTIONS,
  MUNKI_UNINSTALL_METHOD_OPTIONS,
} from "../software/munki-software";
import { type PackageEditorForm } from "./editor-form";
import {
  type Architecture,
  emptyInstallerEnvironmentRow,
  emptyInstallItemRow,
  emptyItemToCopyRow,
  emptyPackageReferenceRow,
  emptyReceiptRow,
  emptyStringRow,
  type InstallerEnvironmentRow,
  type InstallItemRow,
  type ItemToCopyRow,
  type PackageFormState,
  packageLabel,
  type PackageReferenceRow,
  type ReceiptRow,
  removeAt,
  replaceAt,
  scriptFields,
  type ScriptKey,
  type StringRow,
  toggleArray,
} from "./form-state";

const shellExtensions: Extension[] = [StreamLanguage.define(shell)];

// uninstall_script lives on the Uninstall tab; the rest are general-purpose hooks.
const generalScriptFields = scriptFields.filter((script) => script.key !== "uninstall_script");

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
    <ScrollableTabs
      className="max-w-6xl"
      tabs={[
        {
          value: "basic",
          label: "Basic Info",
          content: <BasicInfoTab form={form} software={softwareInfo} />,
        },
        {
          value: "contents",
          label: "Contents",
          content: <ContentsTab form={form} />,
        },
        {
          value: "requirements",
          label: "Requirements",
          content: <RequirementsTab form={form} packageOptions={packageOptions} />,
        },
        {
          value: "installation",
          label: "Installation",
          content: (
            <InstallationTab
              form={form}
              installerFile={installerFile}
              installerArtifactLocation={installerArtifactLocation}
              onInstallerFileChange={onInstallerFileChange}
            />
          ),
        },
        {
          value: "uninstall",
          label: "Uninstall",
          content: (
            <UninstallTab
              form={form}
              uninstallerFile={uninstallerFile}
              uninstallerArtifactLocation={uninstallerArtifactLocation}
              onUninstallerFileChange={onUninstallerFileChange}
            />
          ),
        },
        {
          value: "scripts",
          label: "Scripts",
          content: <ScriptsTab form={form} />,
        },
        {
          value: "alerts",
          label: "Alerts",
          content: <AlertsTab form={form} />,
        },
        {
          value: "advanced",
          label: "Advanced",
          content: <AdvancedTab form={form} />,
        },
      ]}
    />
  );
}

function BasicInfoTab({
  form,
  software,
}: {
  form: PackageEditorForm;
  software: SoftwareInfo | null;
}) {
  return (
    <FieldGroup>
      <InheritedSummary software={software} />

      <FieldSet>
        <FieldLegend>Package</FieldLegend>
        <FieldGroup className="grid gap-4 md:grid-cols-2">
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
        <FormTextareaField form={form} name="notes" id="munki-package-notes" label="Notes" />
      </FieldSet>

      <FieldSet>
        <FieldLegend>Behavior</FieldLegend>
        <FieldGroup className="grid gap-4 md:grid-cols-2">
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
          <FormCheckboxField
            form={form}
            name="on_demand"
            id="munki-package-on-demand"
            label="On demand"
          />
          <FormCheckboxField
            form={form}
            name="autoremove"
            id="munki-package-autoremove"
            label="Autoremove"
          />
        </FieldGroup>
      </FieldSet>

      <FieldSet>
        <FieldLegend>Woodstar</FieldLegend>
        <FormCheckboxField
          form={form}
          name="eligible"
          id="munki-package-eligible"
          label="Available for targeting"
        />
      </FieldSet>
    </FieldGroup>
  );
}

function InheritedSummary({ software }: { software: SoftwareInfo | null }) {
  return (
    <div className="rounded-md border bg-muted/30 p-4">
      <p className="mb-3 text-xs font-medium text-muted-foreground">
        Inherited from the parent software
      </p>
      <dl className="grid gap-x-6 gap-y-3 sm:grid-cols-2">
        <InheritedItem label="Name" value={software?.name} />
        <InheritedItem label="Category" value={software?.category} />
        <InheritedItem label="Developer" value={software?.developer} />
        <InheritedItem
          label="Description"
          value={software?.description}
          className="sm:col-span-2"
        />
      </dl>
    </div>
  );
}

function InheritedItem({
  label,
  value,
  className,
}: {
  label: string;
  value?: string;
  className?: string;
}) {
  const hasValue = value !== undefined && value !== "";
  return (
    <div className={className}>
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className="text-sm whitespace-pre-wrap">
        {hasValue ? value : <span className="text-muted-foreground">None</span>}
      </dd>
    </div>
  );
}

function ContentsTab({ form }: { form: PackageEditorForm }) {
  return (
    <FieldGroup>
      <form.Field
        name="installs"
        children={(field) => (
          <InstallsTable rows={field.state.value} onChange={(rows) => field.handleChange(rows)} />
        )}
      />
      <form.Field
        name="receipts"
        children={(field) => (
          <ReceiptsTable rows={field.state.value} onChange={(rows) => field.handleChange(rows)} />
        )}
      />
    </FieldGroup>
  );
}

function RequirementsTab({
  form,
  packageOptions,
}: {
  form: PackageEditorForm;
  packageOptions: MunkiPackage[];
}) {
  return (
    <FieldGroup>
      <form.Field
        name="requires"
        children={(field) => (
          <PackageReferenceEditor
            legend="Requires"
            addLabel="Add requirement"
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
            addLabel="Add update target"
            rows={field.state.value}
            packageOptions={packageOptions}
            onChange={(rows) => field.handleChange(rows)}
          />
        )}
      />
      <FieldSet>
        <FieldLegend>Compatibility</FieldLegend>
        <FieldGroup className="grid gap-4 md:grid-cols-3">
          <FormTextField
            form={form}
            name="minimum_munki_version"
            id="munki-package-minimum-munki-version"
            label="Minimum Munki Version"
          />
          <FormTextField
            form={form}
            name="minimum_os_version"
            id="munki-package-minimum-os"
            label="Minimum OS"
          />
          <FormTextField
            form={form}
            name="maximum_os_version"
            id="munki-package-maximum-os"
            label="Maximum OS"
          />
        </FieldGroup>
        <FormCodeField
          form={form}
          name="installable_condition"
          id="munki-package-installable-condition"
          label="Installable Condition"
          minHeight="[&_.cm-content]:min-h-32"
        />
      </FieldSet>
    </FieldGroup>
  );
}

function InstallationTab({
  form,
  installerFile,
  installerArtifactLocation,
  onInstallerFileChange,
}: {
  form: PackageEditorForm;
  installerFile: File | null;
  installerArtifactLocation: string;
  onInstallerFileChange: (file: File | null) => void;
}) {
  return (
    <FieldGroup>
      <form.Subscribe selector={(state) => state.values.installer_type}>
        {(installerType) =>
          installerType === "nopkg" ? null : (
            <FieldSet>
              <FieldLegend>Installer</FieldLegend>
              <PackageFileField
                id="munki-package-installer-file"
                label="Installer Artifact"
                description={installerArtifactLocation || "No installer artifact selected."}
                icon={<FileArchive className="size-4" />}
                file={installerFile}
                onChange={onInstallerFileChange}
              />
            </FieldSet>
          )
        }
      </form.Subscribe>

      <form.Field
        name="items_to_copy"
        children={(field) => (
          <ItemsToCopyEditor
            rows={field.state.value}
            onChange={(rows) => field.handleChange(rows)}
          />
        )}
      />

      <form.Field
        name="blocking_applications"
        children={(field) => (
          <StringArrayEditor
            legend="Blocking Applications"
            addLabel="Add application"
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
          <ArchitectureEditor
            values={field.state.value}
            onChange={(values) => field.handleChange(values)}
          />
        )}
      />

      <form.Field
        name="installer_choices_xml"
        children={(field) => (
          <InstallerChoicesField
            value={field.state.value}
            onChange={(value) => field.handleChange(value)}
          />
        )}
      />
    </FieldGroup>
  );
}

function UninstallTab({
  form,
  uninstallerFile,
  uninstallerArtifactLocation,
  onUninstallerFileChange,
}: {
  form: PackageEditorForm;
  uninstallerFile: File | null;
  uninstallerArtifactLocation: string;
  onUninstallerFileChange: (file: File | null) => void;
}) {
  return (
    <FieldSet>
      <FieldLegend>Uninstall</FieldLegend>
      <FieldGroup>
        <div className="max-w-sm">
          <FormSelectField
            form={form}
            name="uninstall_method"
            id="munki-package-uninstall-method"
            label="Uninstall Method"
            options={MUNKI_UNINSTALL_METHOD_OPTIONS}
          />
        </div>
        <form.Subscribe selector={(state) => state.values.uninstall_method}>
          {(method) => (
            <>
              {method === "uninstall_package" ? (
                <PackageFileField
                  id="munki-package-uninstaller-file"
                  label="Uninstaller Artifact"
                  description={uninstallerArtifactLocation || "No uninstaller artifact selected."}
                  icon={<FileArchive className="size-4" />}
                  file={uninstallerFile}
                  onChange={onUninstallerFileChange}
                />
              ) : null}
              {method === "uninstall_script" ? (
                <form.Field
                  name="uninstall_script"
                  children={(field) => (
                    <ScriptField
                      label="Uninstall Script"
                      value={field.state.value}
                      onChange={(value) => field.handleChange(value)}
                    />
                  )}
                />
              ) : null}
            </>
          )}
        </form.Subscribe>
      </FieldGroup>
    </FieldSet>
  );
}

function ScriptsTab({ form }: { form: PackageEditorForm }) {
  return (
    <form.Subscribe
      selector={(state) => state.values}
      children={(values) => (
        <ScriptsEditor values={values} onChange={(key, value) => form.setFieldValue(key, value)} />
      )}
    />
  );
}

function ScriptsEditor({
  values,
  onChange,
}: {
  values: Pick<PackageFormState, ScriptKey>;
  onChange: (key: ScriptKey, value: string) => void;
}) {
  const [active, setActive] = useState<ScriptKey>(generalScriptFields[0].key);
  const { listRef, box } = useTabIndicator(active);

  return (
    <Tabs value={active} onValueChange={(value) => setActive(value as ScriptKey)} className="gap-4">
      <ScrollArea orientation="horizontal">
        <TabsList ref={listRef} className="relative w-fit">
          <span
            aria-hidden
            className="pointer-events-none absolute inset-y-[3px] left-0 rounded-md bg-background shadow-sm transition-[transform,width,opacity] duration-300 ease-out dark:border dark:border-input dark:bg-input/30"
            style={{
              transform: `translateX(${box?.left ?? 0}px)`,
              width: box?.width ?? 0,
              opacity: box ? 1 : 0,
            }}
          />
          {generalScriptFields.map((script) => (
            <TabsTrigger
              key={script.key}
              value={script.key}
              className="relative z-10 bg-transparent! shadow-none! dark:border-transparent! dark:bg-transparent!"
            >
              {script.label}
              {values[script.key] !== "" ? (
                <span className="size-1.5 shrink-0 rounded-full bg-primary" aria-hidden />
              ) : null}
            </TabsTrigger>
          ))}
        </TabsList>
      </ScrollArea>
      {generalScriptFields.map((script) => (
        <TabsContent key={script.key} value={script.key} className="min-w-0">
          <ScriptField
            value={values[script.key]}
            onChange={(value) => onChange(script.key, value)}
          />
        </TabsContent>
      ))}
    </Tabs>
  );
}

function AlertsTab({ form }: { form: PackageEditorForm }) {
  return (
    <FieldGroup>
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

function AdvancedTab({ form }: { form: PackageEditorForm }) {
  return (
    <FieldGroup>
      <FieldSet>
        <FieldLegend>Installer Details</FieldLegend>
        <FieldGroup className="grid gap-4 md:grid-cols-2">
          <FormTextField
            form={form}
            name="package_path"
            id="munki-package-package-path"
            label="Package Path"
          />
          <FormTextField
            form={form}
            name="installed_size"
            id="munki-package-installed-size"
            label="Installed Size"
            type="number"
            inputMode="numeric"
          />
        </FieldGroup>
        <form.Field
          name="installer_environment"
          children={(field) => (
            <InstallerEnvironmentEditor
              rows={field.state.value}
              onChange={(rows) => field.handleChange(rows)}
            />
          )}
        />
      </FieldSet>

      <FieldSet>
        <FieldLegend>Flags</FieldLegend>
        <FieldGroup className="grid gap-4 md:grid-cols-3">
          <FormCheckboxField
            form={form}
            name="precache"
            id="munki-package-precache"
            label="Precache"
          />
          <FormCheckboxField
            form={form}
            name="apple_item"
            id="munki-package-apple-item"
            label="Apple item"
          />
          <FormCheckboxField
            form={form}
            name="suppress_bundle_relocation"
            id="munki-package-suppress-bundle-relocation"
            label="Suppress bundle relocation"
          />
        </FieldGroup>
      </FieldSet>
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
        <div className="flex size-9 shrink-0 items-center justify-center rounded-md border bg-muted text-muted-foreground">
          {icon}
        </div>
        <Input
          id={id}
          type="file"
          accept={accept}
          onChange={(event) => onChange(event.target.files?.[0] ?? null)}
        />
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
      <Checkbox
        id={id}
        checked={checked}
        disabled={disabled}
        onCheckedChange={(value) => onChange(value === true)}
      />
      <FieldContent>
        <FieldLabel htmlFor={id}>{label}</FieldLabel>
        {description ? <FieldDescription>{description}</FieldDescription> : null}
      </FieldContent>
    </Field>
  );
}

function InstallerChoicesField({
  value,
  onChange,
}: {
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <Field>
      <FieldLabel>Installer Choices XML</FieldLabel>
      <CodeEditor
        value={value}
        onChange={onChange}
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
      <CollectionHeader
        title={legend}
        addLabel={addLabel}
        onAdd={() => onChange([...rows, emptyStringRow()])}
      />
      <div className="space-y-2 empty:hidden">
        {rows.map((row, index) => (
          <div key={row.rowID} className="grid gap-2 md:grid-cols-[minmax(0,1fr)_auto]">
            <Input
              value={row.value}
              onChange={(event) =>
                onChange(replaceAt(rows, index, { ...row, value: event.target.value }))
              }
            />
            <IconButton label="Remove" onClick={() => onChange(removeAt(rows, index))}>
              <Trash2 />
            </IconButton>
          </div>
        ))}
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
      <CollectionHeader
        title={legend}
        addLabel={addLabel}
        onAdd={() => onChange([...rows, emptyPackageReferenceRow()])}
      />
      <div className="space-y-2 empty:hidden">
        {rows.map((row, index) => (
          <div key={row.rowID} className="grid gap-2 md:grid-cols-[minmax(0,1fr)_auto]">
            <Select
              value={row.package_id ? String(row.package_id) : "select"}
              onValueChange={(value) =>
                onChange(
                  replaceAt(rows, index, {
                    ...row,
                    package_id: value === "select" ? undefined : Number(value),
                  }),
                )
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
      <CollectionHeader
        title="Installer Environment"
        addLabel="Add variable"
        onAdd={() => onChange([...rows, emptyInstallerEnvironmentRow()])}
      />
      <div className="space-y-2 empty:hidden">
        {rows.map((row, index) => (
          <div
            key={row.rowID}
            className="grid gap-2 md:grid-cols-[minmax(0,12rem)_minmax(0,1fr)_auto]"
          >
            <Input
              aria-label="Name"
              value={row.name}
              onChange={(event) =>
                onChange(replaceAt(rows, index, { ...row, name: event.target.value }))
              }
            />
            <Input
              aria-label="Value"
              value={row.value}
              onChange={(event) =>
                onChange(replaceAt(rows, index, { ...row, value: event.target.value }))
              }
            />
            <IconButton label="Remove" onClick={() => onChange(removeAt(rows, index))}>
              <Trash2 />
            </IconButton>
          </div>
        ))}
      </div>
    </FieldSet>
  );
}

function InstallsTable({
  rows,
  onChange,
}: {
  rows: InstallItemRow[];
  onChange: (rows: InstallItemRow[]) => void;
}) {
  return (
    <FieldSet className="min-w-0">
      <CollectionHeader
        title="Installs"
        addLabel="Add install item"
        onAdd={() => onChange([...rows, emptyInstallItemRow()])}
      />
      {rows.length > 0 ? (
        <div className="overflow-hidden rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="min-w-[14rem]">Path</TableHead>
                <TableHead className="w-[9rem]">Type</TableHead>
                <TableHead className="min-w-[10rem]">CFBundleName</TableHead>
                <TableHead className="min-w-[12rem]">CFBundleIdentifier</TableHead>
                <TableHead className="min-w-[9rem]">CFBundleShortVersionString</TableHead>
                <TableHead className="min-w-[9rem]">CFBundleVersion</TableHead>
                <TableHead className="w-9" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row, index) => (
                <TableRow key={row.rowID} className="hover:bg-transparent">
                  <TableCell className="p-0">
                    <CellInput
                      aria-label="Path"
                      value={row.path}
                      onChange={(event) =>
                        onChange(replaceAt(rows, index, { ...row, path: event.target.value }))
                      }
                    />
                  </TableCell>
                  <TableCell className="p-0">
                    <Select
                      value={row.type}
                      onValueChange={(next) =>
                        onChange(
                          replaceAt(rows, index, { ...row, type: next as InstallItemRow["type"] }),
                        )
                      }
                    >
                      <SelectTrigger
                        aria-label="Type"
                        className="h-8 rounded-none border-0 bg-transparent px-2 shadow-none focus-visible:ring-1 focus-visible:ring-inset dark:bg-transparent"
                      >
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
                  </TableCell>
                  <TableCell className="p-0">
                    <CellInput
                      aria-label="CFBundleName"
                      value={row.bundle_name ?? ""}
                      onChange={(event) =>
                        onChange(
                          replaceAt(rows, index, { ...row, bundle_name: event.target.value }),
                        )
                      }
                    />
                  </TableCell>
                  <TableCell className="p-0">
                    <CellInput
                      aria-label="CFBundleIdentifier"
                      value={row.bundle_identifier ?? ""}
                      onChange={(event) =>
                        onChange(
                          replaceAt(rows, index, { ...row, bundle_identifier: event.target.value }),
                        )
                      }
                    />
                  </TableCell>
                  <TableCell className="p-0">
                    <CellInput
                      aria-label="CFBundleShortVersionString"
                      value={row.bundle_short_version ?? ""}
                      onChange={(event) =>
                        onChange(
                          replaceAt(rows, index, {
                            ...row,
                            bundle_short_version: event.target.value,
                          }),
                        )
                      }
                    />
                  </TableCell>
                  <TableCell className="p-0">
                    <CellInput
                      aria-label="CFBundleVersion"
                      value={row.bundle_version ?? ""}
                      onChange={(event) =>
                        onChange(
                          replaceAt(rows, index, { ...row, bundle_version: event.target.value }),
                        )
                      }
                    />
                  </TableCell>
                  <TableCell className="w-9 p-0 pr-1 text-right">
                    <IconButton label="Remove" onClick={() => onChange(removeAt(rows, index))}>
                      <Trash2 />
                    </IconButton>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : (
        <EmptyPanel>No installs</EmptyPanel>
      )}
    </FieldSet>
  );
}

function ReceiptsTable({
  rows,
  onChange,
}: {
  rows: ReceiptRow[];
  onChange: (rows: ReceiptRow[]) => void;
}) {
  return (
    <FieldSet className="min-w-0">
      <CollectionHeader
        title="Receipts"
        addLabel="Add receipt"
        onAdd={() => onChange([...rows, emptyReceiptRow()])}
      />
      {rows.length > 0 ? (
        <div className="overflow-hidden rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="min-w-[16rem]">Package ID</TableHead>
                <TableHead className="min-w-[9rem]">Version</TableHead>
                <TableHead className="w-24 text-center">Optional</TableHead>
                <TableHead className="w-9" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row, index) => (
                <TableRow key={row.rowID} className="hover:bg-transparent">
                  <TableCell className="p-0">
                    <CellInput
                      aria-label="Package ID"
                      value={row.package_id}
                      onChange={(event) =>
                        onChange(replaceAt(rows, index, { ...row, package_id: event.target.value }))
                      }
                    />
                  </TableCell>
                  <TableCell className="p-0">
                    <CellInput
                      aria-label="Version"
                      value={row.version ?? ""}
                      onChange={(event) =>
                        onChange(replaceAt(rows, index, { ...row, version: event.target.value }))
                      }
                    />
                  </TableCell>
                  <TableCell className="text-center">
                    <Checkbox
                      aria-label="Optional"
                      checked={row.optional === true}
                      onCheckedChange={(value) =>
                        onChange(replaceAt(rows, index, { ...row, optional: value === true }))
                      }
                    />
                  </TableCell>
                  <TableCell className="w-9 p-0 pr-1 text-right">
                    <IconButton label="Remove" onClick={() => onChange(removeAt(rows, index))}>
                      <Trash2 />
                    </IconButton>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : (
        <EmptyPanel>No receipts</EmptyPanel>
      )}
    </FieldSet>
  );
}

function CellInput({ className, ...props }: ComponentProps<typeof Input>) {
  return (
    <Input
      {...props}
      className={cn(
        "h-8 rounded-none border-0 bg-transparent px-2 shadow-none focus-visible:ring-1 focus-visible:ring-inset dark:bg-transparent",
        className,
      )}
    />
  );
}

function ItemsToCopyEditor({
  rows,
  onChange,
}: {
  rows: ItemToCopyRow[];
  onChange: (rows: ItemToCopyRow[]) => void;
}) {
  return (
    <FieldSet>
      <CollectionHeader
        title="Items to Copy"
        addLabel="Add copy item"
        onAdd={() => onChange([...rows, emptyItemToCopyRow()])}
      />
      <div className="space-y-4 empty:hidden">
        {rows.map((row, index) => (
          <div key={row.rowID} className="space-y-3 rounded-md border p-3">
            <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto]">
              <Field>
                <FieldLabel htmlFor={`munki-copy-source-${row.rowID}`}>Source Item</FieldLabel>
                <Input
                  id={`munki-copy-source-${row.rowID}`}
                  value={row.source_item}
                  onChange={(event) =>
                    onChange(replaceAt(rows, index, { ...row, source_item: event.target.value }))
                  }
                />
              </Field>
              <Field>
                <FieldLabel htmlFor={`munki-copy-destination-${row.rowID}`}>
                  Destination Path
                </FieldLabel>
                <Input
                  id={`munki-copy-destination-${row.rowID}`}
                  value={row.destination_path}
                  onChange={(event) =>
                    onChange(
                      replaceAt(rows, index, { ...row, destination_path: event.target.value }),
                    )
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
                <FieldLabel htmlFor={`munki-copy-destination-item-${row.rowID}`}>
                  Destination Item
                </FieldLabel>
                <Input
                  id={`munki-copy-destination-item-${row.rowID}`}
                  value={row.destination_item ?? ""}
                  onChange={(event) =>
                    onChange(
                      replaceAt(rows, index, { ...row, destination_item: event.target.value }),
                    )
                  }
                />
              </Field>
              <Field>
                <FieldLabel htmlFor={`munki-copy-user-${row.rowID}`}>User</FieldLabel>
                <Input
                  id={`munki-copy-user-${row.rowID}`}
                  value={row.user ?? ""}
                  onChange={(event) =>
                    onChange(replaceAt(rows, index, { ...row, user: event.target.value }))
                  }
                />
              </Field>
              <Field>
                <FieldLabel htmlFor={`munki-copy-group-${row.rowID}`}>Group</FieldLabel>
                <Input
                  id={`munki-copy-group-${row.rowID}`}
                  value={row.group ?? ""}
                  onChange={(event) =>
                    onChange(replaceAt(rows, index, { ...row, group: event.target.value }))
                  }
                />
              </Field>
              <Field>
                <FieldLabel htmlFor={`munki-copy-mode-${row.rowID}`}>Mode</FieldLabel>
                <Input
                  id={`munki-copy-mode-${row.rowID}`}
                  value={row.mode ?? ""}
                  onChange={(event) =>
                    onChange(replaceAt(rows, index, { ...row, mode: event.target.value }))
                  }
                />
              </Field>
            </FieldGroup>
          </div>
        ))}
      </div>
    </FieldSet>
  );
}

function ScriptField({
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

function IconButton({
  label,
  children,
  onClick,
}: {
  label: string;
  children: ReactNode;
  onClick: () => void;
}) {
  return (
    <Button type="button" variant="ghost" size="icon-sm" title={label} onClick={onClick}>
      {children}
    </Button>
  );
}

// Section header for a row collection: legend on the left, add button on the right.
function CollectionHeader({
  title,
  addLabel,
  onAdd,
}: {
  title: string;
  addLabel: string;
  onAdd: () => void;
}) {
  return (
    <FieldLegend className="mb-0 flex w-full items-center justify-between gap-3">
      <span>{title}</span>
      <IconButton label={addLabel} onClick={onAdd}>
        <Plus />
      </IconButton>
    </FieldLegend>
  );
}
