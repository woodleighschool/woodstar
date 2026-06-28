import { StreamLanguage } from "@codemirror/language";
import { shell } from "@codemirror/legacy-modes/mode/shell";
import { Link } from "@tanstack/react-router";
import { FileArchive, Trash2 } from "lucide-react";
import { type ComponentProps, type ReactNode, useState } from "react";

import { ConfirmDialog } from "@/components/confirm-dialog";
import { CodeEditor } from "@/components/editor/code-editor";
import { EmptyPanel } from "@/components/empty-panel";
import { FormActions } from "@/components/form-actions";
import { FormField } from "@/components/form-field";
import { KeyValueGrid, KeyValueItem } from "@/components/key-value";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { ScrollableTabs, ScrollableTabsList } from "@/components/layout/scrollable-tabs";
import { MunkiIcon } from "@/components/munki/munki-icon";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Combobox,
  ComboboxAnchor,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxGroup,
  ComboboxGroupLabel,
  ComboboxInput,
  ComboboxItem,
  ComboboxTrigger,
} from "@/components/ui/combobox";
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
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@/components/ui/input-group";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tabs, TabsContent, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import type { MunkiPackage, MunkiPackageAlert } from "@/lib/api";
import { cn, formatBytes } from "@/lib/utils";

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
  numberOrUndefined,
  type PackageFormState,
  packageLabel,
  type PackageReferenceRow,
  type ReceiptRow,
  scriptFields,
  type ScriptKey,
  type StringRow,
  toggleArray,
} from "./form-state";

const shellExtensions = [StreamLanguage.define(shell)];

// uninstall_script lives on the Uninstall tab; the rest are general-purpose hooks.
const generalScriptFields = scriptFields.filter((script) => script.key !== "uninstall_script");

type PackageFieldNameByValue<T> = {
  [K in keyof PackageFormState]: PackageFormState[K] extends T ? K : never;
}[keyof PackageFormState];
type StringPackageFieldName = PackageFieldNameByValue<string>;
type BooleanPackageFieldName = PackageFieldNameByValue<boolean>;

export type SoftwareInfo = {
  id: number;
  name: string;
  description: string;
  category: string;
  developer: string;
  iconUrl?: string;
};

type PackageFormProps = {
  form: PackageEditorForm;
  title: string;
  submitLabel: string;
  softwareInfo: SoftwareInfo | null;
  softwareSelector?: ReactNode;
  packageOptions: MunkiPackage[];
  installerFile: File | null;
  installerMetadata?: MunkiPackage["installer_file"];
  hasInstallerObject: boolean;
  onInstallerFileChange: (file: File | null) => void;
  onDeleteInstaller?: () => Promise<void>;
  deletingInstaller: boolean;
  onCancel: () => void;
};

export function PackageForm({
  form,
  title,
  submitLabel,
  softwareInfo,
  softwareSelector,
  packageOptions,
  installerFile,
  installerMetadata,
  hasInstallerObject,
  onInstallerFileChange,
  onDeleteInstaller,
  deletingInstaller,
  onCancel,
}: PackageFormProps) {
  return (
    <PageShell asChild>
      <form
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit();
        }}
      >
        <PageHeader title={title} />
        <PackageEditorTabs
          form={form}
          softwareInfo={softwareInfo}
          softwareSelector={softwareSelector}
          packageOptions={packageOptions}
          installerFile={installerFile}
          installerMetadata={installerMetadata}
          hasInstallerObject={hasInstallerObject}
          onInstallerFileChange={onInstallerFileChange}
          onDeleteInstaller={onDeleteInstaller}
          deletingInstaller={deletingInstaller}
        />
        <FormActions form={form} submitLabel={submitLabel} onCancel={onCancel} />
      </form>
    </PageShell>
  );
}

export function PackageEditorTabs({
  form,
  softwareInfo,
  softwareSelector,
  packageOptions,
  installerFile,
  installerMetadata,
  hasInstallerObject,
  onInstallerFileChange,
  onDeleteInstaller,
  deletingInstaller,
}: {
  form: PackageEditorForm;
  softwareInfo: SoftwareInfo | null;
  softwareSelector?: ReactNode;
  packageOptions: MunkiPackage[];
  installerFile: File | null;
  installerMetadata?: MunkiPackage["installer_file"];
  hasInstallerObject: boolean;
  onInstallerFileChange: (file: File | null) => void;
  onDeleteInstaller?: () => Promise<void>;
  deletingInstaller: boolean;
}) {
  const tabs = [
    {
      value: "basic",
      label: "Basic Info",
      content: (
        <BasicInfoTab form={form} software={softwareInfo} softwareSelector={softwareSelector} />
      ),
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
          installerMetadata={installerMetadata}
          hasInstallerObject={hasInstallerObject}
          onInstallerFileChange={onInstallerFileChange}
          onDeleteInstaller={onDeleteInstaller}
          deletingInstaller={deletingInstaller}
        />
      ),
    },
    {
      value: "uninstall",
      label: "Uninstall",
      content: <UninstallTab form={form} />,
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
  ];

  return (
    <ScrollableTabs defaultValue="basic" className="max-w-6xl">
      <ScrollableTabsList>
        {tabs.map((tab) => (
          <TabsTrigger key={tab.value} value={tab.value}>
            {tab.label}
          </TabsTrigger>
        ))}
      </ScrollableTabsList>
      {tabs.map((tab) => (
        <TabsContent key={tab.value} value={tab.value}>
          {tab.content}
        </TabsContent>
      ))}
    </ScrollableTabs>
  );
}

function BasicInfoTab({
  form,
  software,
  softwareSelector,
}: {
  form: PackageEditorForm;
  software: SoftwareInfo | null;
  softwareSelector?: ReactNode;
}) {
  return (
    <FieldGroup>
      <ParentSoftwarePanel software={software} selector={softwareSelector} />

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
          <FormSwitchField
            form={form}
            name="restart_required"
            id="munki-package-restart-required"
            label="Restart required"
          />
          <form.Subscribe selector={(state) => state.values.restart_required}>
            {(restartRequired) =>
              restartRequired ? (
                <FormSelectField
                  form={form}
                  name="restart_action"
                  id="munki-package-restart-action"
                  label="Restart Action"
                  options={MUNKI_RESTART_ACTION_OPTIONS}
                />
              ) : null
            }
          </form.Subscribe>
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
        <FieldGroup data-slot="checkbox-group" className="grid gap-4 md:grid-cols-2">
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
        <FormSwitchField
          form={form}
          name="eligible"
          id="munki-package-eligible"
          label="Available for targeting"
        />
      </FieldSet>
    </FieldGroup>
  );
}

function ParentSoftwarePanel({
  software,
  selector,
}: {
  software: SoftwareInfo | null;
  selector?: ReactNode;
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-start gap-3">
        <MunkiIcon iconUrl={software?.iconUrl} size="md" loading="eager" />
        <div className="min-w-0">
          <CardDescription>Parent software</CardDescription>
          {software && !selector ? (
            <Link
              to="/munki/software/$softwareId"
              params={{ softwareId: String(software.id) }}
              className="block truncate text-sm font-medium hover:underline"
            >
              {software.name}
            </Link>
          ) : software ? (
            <p className="truncate text-sm font-medium">{software.name}</p>
          ) : (
            <p className="text-sm text-muted-foreground">Select software</p>
          )}
        </div>
      </CardHeader>

      {selector || software ? (
        <CardContent className="flex flex-col gap-4">
          {selector ? <div className="max-w-xl">{selector}</div> : null}

          {software ? (
            <KeyValueGrid>
              <KeyValueItem label="Category" value={software.category} />
              <KeyValueItem label="Developer" value={software.developer} />
              <KeyValueItem
                label="Description"
                value={software.description}
                className="sm:col-span-2"
                valueClassName="whitespace-pre-wrap"
              />
            </KeyValueGrid>
          ) : null}
        </CardContent>
      ) : null}
    </Card>
  );
}

function ContentsTab({ form }: { form: PackageEditorForm }) {
  return (
    <FieldGroup>
      <form.Field
        name="installs"
        mode="array"
        children={(field) => (
          <InstallsTable
            rows={field.state.value}
            onAdd={() => field.pushValue(emptyInstallItemRow())}
            onReplace={(index, row) => field.replaceValue(index, row)}
            onRemove={(index) => field.removeValue(index)}
          />
        )}
      />
      <form.Field
        name="receipts"
        mode="array"
        children={(field) => (
          <ReceiptsTable
            rows={field.state.value}
            onAdd={() => field.pushValue(emptyReceiptRow())}
            onReplace={(index, row) => field.replaceValue(index, row)}
            onRemove={(index) => field.removeValue(index)}
          />
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
        mode="array"
        children={(field) => (
          <PackageReferenceEditor
            legend="Requires"
            addLabel="Add requirement"
            rows={field.state.value}
            packageOptions={packageOptions}
            onAdd={() => field.pushValue(emptyPackageReferenceRow())}
            onReplace={(index, row) => field.replaceValue(index, row)}
            onRemove={(index) => field.removeValue(index)}
          />
        )}
      />
      <form.Field
        name="update_for"
        mode="array"
        children={(field) => (
          <PackageReferenceEditor
            legend="Update For"
            addLabel="Add update target"
            rows={field.state.value}
            packageOptions={packageOptions}
            onAdd={() => field.pushValue(emptyPackageReferenceRow())}
            onReplace={(index, row) => field.replaceValue(index, row)}
            onRemove={(index) => field.removeValue(index)}
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
  installerMetadata,
  hasInstallerObject,
  onInstallerFileChange,
  onDeleteInstaller,
  deletingInstaller,
}: {
  form: PackageEditorForm;
  installerFile: File | null;
  installerMetadata?: MunkiPackage["installer_file"];
  hasInstallerObject: boolean;
  onInstallerFileChange: (file: File | null) => void;
  onDeleteInstaller?: () => Promise<void>;
  deletingInstaller: boolean;
}) {
  return (
    <FieldGroup>
      <form.Subscribe selector={(state) => state.values.installer_type}>
        {(installerType) =>
          installerType === "nopkg" ? null : (
            <InstallerFileCard
              file={installerFile}
              metadata={installerMetadata}
              hasInstallerObject={hasInstallerObject}
              deleting={deletingInstaller}
              onFileChange={onInstallerFileChange}
              onDelete={onDeleteInstaller}
            />
          )
        }
      </form.Subscribe>

      <form.Field
        name="items_to_copy"
        mode="array"
        children={(field) => (
          <ItemsToCopyEditor
            rows={field.state.value}
            onAdd={() => field.pushValue(emptyItemToCopyRow())}
            onReplace={(index, row) => field.replaceValue(index, row)}
            onRemove={(index) => field.removeValue(index)}
          />
        )}
      />

      <BlockingApplicationsEditor form={form} />
      <FieldSet>
        <FieldLegend>Blocking Application Handling</FieldLegend>
        <FieldGroup>
          <FormSwitchField
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

function UninstallTab({ form }: { form: PackageEditorForm }) {
  return (
    <FieldSet>
      <FieldLegend>Uninstall</FieldLegend>
      <FieldGroup>
        <FormSwitchField
          form={form}
          name="uninstallable"
          id="munki-package-uninstallable"
          label="Uninstallable"
        />
        <form.Subscribe selector={(state) => state.values}>
          {(values) =>
            values.uninstallable ? (
              <>
                <div className="max-w-sm">
                  <FormSelectField
                    form={form}
                    name="uninstall_method"
                    id="munki-package-uninstall-method"
                    label="Uninstall Method"
                    options={MUNKI_UNINSTALL_METHOD_OPTIONS}
                  />
                </div>
                {values.uninstall_method === "uninstall_script" ? (
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
            ) : null
          }
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

  return (
    <Tabs value={active} onValueChange={(value) => setActive(value as ScriptKey)} className="gap-4">
      <ScrollableTabsList variant="default">
        {generalScriptFields.map((script) => (
          <TabsTrigger key={script.key} value={script.key}>
            {script.label}
            {values[script.key] !== "" ? (
              <span className="size-1.5 shrink-0 rounded-full bg-primary" aria-hidden />
            ) : null}
          </TabsTrigger>
        ))}
      </ScrollableTabsList>
      {generalScriptFields.map((script) => (
        <TabsContent key={script.key} value={script.key}>
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
          mode="array"
          children={(field) => (
            <InstallerEnvironmentEditor
              rows={field.state.value}
              onAdd={() => field.pushValue(emptyInstallerEnvironmentRow())}
              onReplace={(index, row) => field.replaceValue(index, row)}
              onRemove={(index) => field.removeValue(index)}
            />
          )}
        />
      </FieldSet>

      <FieldSet>
        <FieldLegend>Flags</FieldLegend>
        <FieldGroup className="grid gap-4 md:grid-cols-3">
          <FormSwitchField
            form={form}
            name="precache"
            id="munki-package-precache"
            label="Precache"
          />
          <FormSwitchField
            form={form}
            name="apple_item"
            id="munki-package-apple-item"
            label="Apple item"
          />
          <FormSwitchField
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

function VersionField({ form }: { form: PackageEditorForm }) {
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

function FormTextareaField({
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

function FormCodeField({
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

function FormSelectField<
  Name extends StringPackageFieldName,
  T extends PackageFormState[Name] & string,
>({
  form,
  name,
  id,
  label,
  options,
}: {
  form: PackageEditorForm;
  name: Name;
  id: string;
  label: string;
  options: Array<{ value: T; label: string }>;
}) {
  return (
    <form.Field name={name}>
      {(field) => (
        <FormField field={field} label={label} htmlFor={id}>
          {() => (
            <Select
              value={field.state.value}
              onValueChange={(next) =>
                field.handleChange(next as Parameters<typeof field.handleChange>[0])
              }
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
          )}
        </FormField>
      )}
    </form.Field>
  );
}

function FormSwitchField({
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

function FormCheckboxField({
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

function InstallerFileCard({
  file,
  metadata,
  hasInstallerObject,
  deleting,
  onFileChange,
  onDelete,
}: {
  file: File | null;
  metadata?: MunkiPackage["installer_file"];
  hasInstallerObject: boolean;
  deleting: boolean;
  onFileChange: (file: File | null) => void;
  onDelete?: () => Promise<void>;
}) {
  const [deleteOpen, setDeleteOpen] = useState(false);
  const canDelete = hasInstallerObject && onDelete !== undefined;

  return (
    <>
      <Card>
        <CardHeader className="flex flex-row items-center justify-between gap-3">
          <CardTitle>Installer</CardTitle>
          {canDelete ? (
            <Button
              type="button"
              variant="destructive"
              size="sm"
              disabled={deleting}
              onClick={() => setDeleteOpen(true)}
            >
              <Trash2 data-icon="inline-start" />
              Delete
            </Button>
          ) : null}
        </CardHeader>
        <CardContent>
          {metadata ? (
            <KeyValueGrid>
              <KeyValueItem label="Filename" value={metadata.filename} />
              <KeyValueItem label="Size" value={formatBytes(metadata.size_bytes)} />
              <KeyValueItem
                label="SHA-256"
                value={metadata.sha256}
                className="sm:col-span-2"
                valueClassName="font-mono text-xs break-all"
              />
            </KeyValueGrid>
          ) : hasInstallerObject ? (
            <p className="text-sm text-muted-foreground">Installer metadata is not available.</p>
          ) : (
            <PackageFileField
              id="munki-package-installer-file"
              label="Installer File"
              description="No installer file selected."
              icon={<FileArchive />}
              file={file}
              onChange={onFileChange}
            />
          )}
        </CardContent>
      </Card>

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title="Delete Installer?"
        description="This detaches the installer from the package and deletes the stored file when it is no longer referenced."
        confirmLabel="Delete"
        variant="destructive"
        pending={deleting}
        onConfirm={() => void onDelete?.()}
      />
    </>
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

function SwitchControl({
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

function BlockingApplicationsEditor({ form }: { form: PackageEditorForm }) {
  return (
    <form.Subscribe selector={(state) => state.values.blocking_applications_none}>
      {(blockingApplicationsNone) => (
        <form.Field
          name="blocking_applications"
          mode="array"
          children={(field) => (
            <FieldSet className="gap-4">
              <FieldLegend variant="label">Blocking Applications</FieldLegend>
              <FieldGroup className="gap-2">
                <form.Field
                  name="blocking_applications_none"
                  children={(noneField) => (
                    <Field orientation="horizontal" className="max-w-xl">
                      <FieldContent>
                        <FieldLabel htmlFor="munki-package-blocking-applications-none">
                          No blocking applications
                        </FieldLabel>
                        <FieldDescription>Render blocking_applications as [].</FieldDescription>
                      </FieldContent>
                      <Switch
                        id="munki-package-blocking-applications-none"
                        checked={noneField.state.value}
                        onCheckedChange={(checked) => {
                          noneField.handleChange(checked);
                          if (checked && field.state.value.length > 0) {
                            field.handleChange([]);
                          }
                        }}
                      />
                    </Field>
                  )}
                />
                {blockingApplicationsNone ? null : (
                  <>
                    <StringArrayRows
                      removeLabel="Remove application"
                      rows={field.state.value}
                      onReplace={(index, row) => field.replaceValue(index, row)}
                      onRemove={(index) => field.removeValue(index)}
                    />
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      className="w-fit"
                      onClick={() => field.pushValue(emptyStringRow())}
                    >
                      Add application
                    </Button>
                  </>
                )}
              </FieldGroup>
            </FieldSet>
          )}
        />
      )}
    </form.Subscribe>
  );
}

function StringArrayRows({
  removeLabel,
  rows,
  onReplace,
  onRemove,
}: {
  removeLabel: string;
  rows: StringRow[];
  onReplace: (index: number, row: StringRow) => void;
  onRemove: (index: number) => void;
}) {
  return (
    <>
      {rows.map((row, index) => (
        <InputGroup key={row.rowID}>
          <InputGroupInput
            aria-label="Application"
            value={row.value}
            onChange={(event) => onReplace(index, { ...row, value: event.target.value })}
          />
          <InputGroupAddon align="inline-end">
            <InputGroupButton
              type="button"
              variant="ghost"
              size="icon-xs"
              aria-label={removeLabel}
              onClick={() => onRemove(index)}
            >
              <Trash2 />
            </InputGroupButton>
          </InputGroupAddon>
        </InputGroup>
      ))}
    </>
  );
}

function PackageReferenceEditor({
  legend,
  addLabel,
  rows,
  packageOptions,
  onAdd,
  onReplace,
  onRemove,
}: {
  legend: string;
  addLabel: string;
  rows: PackageReferenceRow[];
  packageOptions: MunkiPackage[];
  onAdd: () => void;
  onReplace: (index: number, row: PackageReferenceRow) => void;
  onRemove: (index: number) => void;
}) {
  const packageGroups = packageReferenceGroups(packageOptions);

  return (
    <FieldSet className="gap-4">
      <FieldLegend variant="label">{legend}</FieldLegend>
      <FieldGroup className="gap-2">
        {rows.map((row, index) => (
          <PackageReferenceCombobox
            key={row.rowID}
            row={row}
            packageGroups={packageGroups}
            onChange={(next) => onReplace(index, next)}
            onRemove={() => onRemove(index)}
          />
        ))}
        <Button type="button" variant="outline" size="sm" className="w-fit" onClick={onAdd}>
          {addLabel}
        </Button>
      </FieldGroup>
    </FieldSet>
  );
}

function PackageReferenceCombobox({
  row,
  packageGroups,
  onChange,
  onRemove,
}: {
  row: PackageReferenceRow;
  packageGroups: ReturnType<typeof packageReferenceGroups>;
  onChange: (row: PackageReferenceRow) => void;
  onRemove: () => void;
}) {
  const [inputValue, setInputValue] = useState(packageReferenceInputValue(row));
  const selectedValue = row.package_id ? packageReferencePackageValue(row.package_id) : "";

  return (
    <Combobox
      value={selectedValue}
      inputValue={inputValue}
      onInputValueChange={setInputValue}
      onValueChange={(value) => {
        const selection = packageReferenceSelection(value, packageGroups);
        if (!selection) return;
        onChange({ rowID: row.rowID, ...selection });
        setInputValue(packageReferenceInputValue(selection));
      }}
    >
      <ComboboxAnchor className="w-full">
        <ComboboxInput placeholder="Select Package" />
        <ComboboxTrigger aria-label="Open packages" />
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          aria-label="Remove package reference"
          onClick={(event) => {
            event.stopPropagation();
            onRemove();
          }}
        >
          <Trash2 />
        </Button>
      </ComboboxAnchor>
      <ComboboxContent>
        <ComboboxEmpty>
          {packageGroups.length === 0 ? "No Packages Available." : "No Packages Found."}
        </ComboboxEmpty>
        {packageGroups.map((group) => (
          <ComboboxGroup key={group.softwareID}>
            <ComboboxGroupLabel>{group.softwareName}</ComboboxGroupLabel>
            {group.packages.map((option) => (
              <ComboboxItem
                key={option.id}
                value={packageReferencePackageValue(option.id)}
                label={packageLabel(option)}
              >
                {packageLabel(option)}
              </ComboboxItem>
            ))}
          </ComboboxGroup>
        ))}
      </ComboboxContent>
    </Combobox>
  );
}

function packageReferencePackageValue(packageID: number) {
  return `package:${packageID}`;
}

function packageReferenceInputValue(
  row: Pick<PackageReferenceRow, "software_name" | "package_version">,
) {
  if (!row.software_name) return "";
  if (!row.package_version) return row.software_name;
  return `${row.software_name} ${row.package_version}`;
}

function packageReferenceSelection(
  value: string,
  packageGroups: ReturnType<typeof packageReferenceGroups>,
) {
  if (!value.startsWith("package:")) return null;
  const packageID = Number(value.slice("package:".length));
  const pkg = packageGroups
    .flatMap((group) => group.packages)
    .find((option) => option.id === packageID);
  if (!pkg) return null;
  return {
    software_id: pkg.software_id,
    software_name: pkg.software_name,
    package_id: pkg.id,
    package_version: pkg.version,
  };
}

function packageReferenceGroups(packages: MunkiPackage[]) {
  const groups = new Map<
    number,
    { softwareID: number; softwareName: string; packages: MunkiPackage[] }
  >();
  for (const pkg of packages) {
    const group = groups.get(pkg.software_id) ?? {
      softwareID: pkg.software_id,
      softwareName: pkg.software_name,
      packages: [],
    };
    group.packages.push(pkg);
    groups.set(pkg.software_id, group);
  }
  return [...groups.values()];
}

function InstallerEnvironmentEditor({
  rows,
  onAdd,
  onReplace,
  onRemove,
}: {
  rows: InstallerEnvironmentRow[];
  onAdd: () => void;
  onReplace: (index: number, row: InstallerEnvironmentRow) => void;
  onRemove: (index: number) => void;
}) {
  return (
    <FieldSet className="gap-4">
      <FieldLegend variant="label">Installer Environment</FieldLegend>
      <FieldGroup className="gap-2">
        {rows.map((row, index) => (
          <div key={row.rowID} className="grid gap-2 md:grid-cols-[minmax(0,12rem)_minmax(0,1fr)]">
            <Input
              aria-label="Name"
              value={row.name}
              onChange={(event) => onReplace(index, { ...row, name: event.target.value })}
            />
            <InputGroup>
              <InputGroupInput
                aria-label="Value"
                value={row.value}
                onChange={(event) => onReplace(index, { ...row, value: event.target.value })}
              />
              <InputGroupAddon align="inline-end">
                <InputGroupButton
                  type="button"
                  variant="ghost"
                  size="icon-xs"
                  aria-label="Remove variable"
                  onClick={() => onRemove(index)}
                >
                  <Trash2 />
                </InputGroupButton>
              </InputGroupAddon>
            </InputGroup>
          </div>
        ))}
        <Button type="button" variant="outline" size="sm" className="w-fit" onClick={onAdd}>
          Add variable
        </Button>
      </FieldGroup>
    </FieldSet>
  );
}

function InstallsTable({
  rows,
  onAdd,
  onReplace,
  onRemove,
}: {
  rows: InstallItemRow[];
  onAdd: () => void;
  onReplace: (index: number, row: InstallItemRow) => void;
  onRemove: (index: number) => void;
}) {
  return (
    <FieldSet className="min-w-0 gap-4">
      <FieldLegend variant="label">Installs</FieldLegend>
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
                <TableHead className="min-w-[9rem]">Minimum Update</TableHead>
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
                      onChange={(event) => onReplace(index, { ...row, path: event.target.value })}
                    />
                  </TableCell>
                  <TableCell className="p-0">
                    <Select
                      value={row.type}
                      onValueChange={(next) =>
                        onReplace(index, { ...row, type: next as InstallItemRow["type"] })
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
                        onReplace(index, { ...row, bundle_name: event.target.value })
                      }
                    />
                  </TableCell>
                  <TableCell className="p-0">
                    <CellInput
                      aria-label="CFBundleIdentifier"
                      value={row.bundle_identifier ?? ""}
                      onChange={(event) =>
                        onReplace(index, { ...row, bundle_identifier: event.target.value })
                      }
                    />
                  </TableCell>
                  <TableCell className="p-0">
                    <CellInput
                      aria-label="CFBundleShortVersionString"
                      value={row.bundle_short_version ?? ""}
                      onChange={(event) =>
                        onReplace(index, {
                          ...row,
                          bundle_short_version: event.target.value,
                        })
                      }
                    />
                  </TableCell>
                  <TableCell className="p-0">
                    <CellInput
                      aria-label="CFBundleVersion"
                      value={row.bundle_version ?? ""}
                      onChange={(event) =>
                        onReplace(index, { ...row, bundle_version: event.target.value })
                      }
                    />
                  </TableCell>
                  <TableCell className="p-0">
                    <CellInput
                      aria-label="Minimum Update Version"
                      value={row.minimum_update_version ?? ""}
                      onChange={(event) =>
                        onReplace(index, {
                          ...row,
                          minimum_update_version: event.target.value,
                        })
                      }
                    />
                  </TableCell>
                  <TableCell className="w-9 p-0 pr-1 text-right">
                    <IconButton label="Remove" onClick={() => onRemove(index)}>
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
      <Button type="button" variant="outline" size="sm" className="w-fit" onClick={onAdd}>
        Add install item
      </Button>
    </FieldSet>
  );
}

function ReceiptsTable({
  rows,
  onAdd,
  onReplace,
  onRemove,
}: {
  rows: ReceiptRow[];
  onAdd: () => void;
  onReplace: (index: number, row: ReceiptRow) => void;
  onRemove: (index: number) => void;
}) {
  return (
    <FieldSet className="min-w-0 gap-4">
      <FieldLegend variant="label">Receipts</FieldLegend>
      {rows.length > 0 ? (
        <div className="overflow-hidden rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="min-w-[16rem]">Package ID</TableHead>
                <TableHead className="min-w-[9rem]">Version</TableHead>
                <TableHead className="min-w-[10rem]">Name</TableHead>
                <TableHead className="min-w-[8rem]">Installed Size</TableHead>
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
                        onReplace(index, { ...row, package_id: event.target.value })
                      }
                    />
                  </TableCell>
                  <TableCell className="p-0">
                    <CellInput
                      aria-label="Version"
                      value={row.version ?? ""}
                      onChange={(event) =>
                        onReplace(index, { ...row, version: event.target.value })
                      }
                    />
                  </TableCell>
                  <TableCell className="p-0">
                    <CellInput
                      aria-label="Name"
                      value={row.name ?? ""}
                      onChange={(event) => onReplace(index, { ...row, name: event.target.value })}
                    />
                  </TableCell>
                  <TableCell className="p-0">
                    <CellInput
                      aria-label="Installed Size"
                      type="number"
                      min={0}
                      value={row.installed_size ?? ""}
                      onChange={(event) =>
                        onReplace(index, {
                          ...row,
                          installed_size: numberOrUndefined(event.target.value),
                        })
                      }
                    />
                  </TableCell>
                  <TableCell className="text-center">
                    <Checkbox
                      aria-label="Optional"
                      checked={row.optional === true}
                      onCheckedChange={(value) =>
                        onReplace(index, { ...row, optional: value === true })
                      }
                    />
                  </TableCell>
                  <TableCell className="w-9 p-0 pr-1 text-right">
                    <IconButton label="Remove" onClick={() => onRemove(index)}>
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
      <Button type="button" variant="outline" size="sm" className="w-fit" onClick={onAdd}>
        Add receipt
      </Button>
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
  onAdd,
  onReplace,
  onRemove,
}: {
  rows: ItemToCopyRow[];
  onAdd: () => void;
  onReplace: (index: number, row: ItemToCopyRow) => void;
  onRemove: (index: number) => void;
}) {
  return (
    <FieldSet className="gap-4">
      <FieldLegend variant="label">Items to Copy</FieldLegend>
      <FieldGroup className="gap-4">
        {rows.map((row, index) => (
          <div key={row.rowID} className="flex flex-col gap-3 rounded-md border p-3">
            <div className="grid gap-3 md:grid-cols-2">
              <Field>
                <FieldLabel htmlFor={`munki-copy-source-${row.rowID}`}>Source Item</FieldLabel>
                <Input
                  id={`munki-copy-source-${row.rowID}`}
                  value={row.source_item}
                  onChange={(event) =>
                    onReplace(index, { ...row, source_item: event.target.value })
                  }
                />
              </Field>
              <Field>
                <FieldLabel htmlFor={`munki-copy-destination-${row.rowID}`}>
                  Destination Path
                </FieldLabel>
                <InputGroup>
                  <InputGroupInput
                    id={`munki-copy-destination-${row.rowID}`}
                    value={row.destination_path}
                    onChange={(event) =>
                      onReplace(index, { ...row, destination_path: event.target.value })
                    }
                  />
                  <InputGroupAddon align="inline-end">
                    <InputGroupButton
                      type="button"
                      variant="ghost"
                      size="icon-xs"
                      aria-label="Remove copy item"
                      onClick={() => onRemove(index)}
                    >
                      <Trash2 />
                    </InputGroupButton>
                  </InputGroupAddon>
                </InputGroup>
              </Field>
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
                    onReplace(index, { ...row, destination_item: event.target.value })
                  }
                />
              </Field>
              <Field>
                <FieldLabel htmlFor={`munki-copy-user-${row.rowID}`}>User</FieldLabel>
                <Input
                  id={`munki-copy-user-${row.rowID}`}
                  value={row.user ?? ""}
                  onChange={(event) => onReplace(index, { ...row, user: event.target.value })}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor={`munki-copy-group-${row.rowID}`}>Group</FieldLabel>
                <Input
                  id={`munki-copy-group-${row.rowID}`}
                  value={row.group ?? ""}
                  onChange={(event) => onReplace(index, { ...row, group: event.target.value })}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor={`munki-copy-mode-${row.rowID}`}>Mode</FieldLabel>
                <Input
                  id={`munki-copy-mode-${row.rowID}`}
                  value={row.mode ?? ""}
                  onChange={(event) => onReplace(index, { ...row, mode: event.target.value })}
                />
              </Field>
            </FieldGroup>
          </div>
        ))}
        <Button type="button" variant="outline" size="sm" className="w-fit" onClick={onAdd}>
          Add copy item
        </Button>
      </FieldGroup>
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
