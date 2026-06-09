import { xml } from "@codemirror/lang-xml";
import type { Extension } from "@codemirror/state";
import { Link } from "@tanstack/react-router";
import { FileArchive, Loader2, Plus, Trash2 } from "lucide-react";
import type { ReactNode } from "react";

import { CodeEditor } from "@/components/editor/code-editor";
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
import type { PackageEditorForm } from "@/hooks/munki/package-editor-form";
import type { MunkiPackage } from "@/hooks/munki/packages";
import type { PackageAlert } from "@/lib/api-client/types.gen";
import { firstErrorMessage, requiredString } from "@/lib/form-validation";
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
} from "@/lib/munki-package-form";
import {
  MUNKI_INSTALLER_TYPE_OPTIONS,
  MUNKI_INSTALL_ITEM_TYPE_OPTIONS,
  MUNKI_RESTART_ACTION_OPTIONS,
  MUNKI_UNINSTALL_METHOD_OPTIONS,
} from "@/lib/munki-software";

const xmlExtensions: Extension[] = [xml()];

type PackageFieldSetter = <K extends keyof PackageFormState>(field: K, value: PackageFormState[K]) => void;

export function PackageEditorTabs({
  form,
  packageOptions,
  installerFile,
  uninstallerFile,
  installerArtifactLocation,
  uninstallerArtifactLocation,
  onInstallerFileChange,
  onUninstallerFileChange,
}: {
  form: PackageEditorForm;
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
      {(values) => {
        const setField: PackageFieldSetter = (field, value) => {
          form.setFieldValue(field as never, value as never);
        };

        return (
          <Tabs defaultValue="identity" className="max-w-5xl">
            <TabsList className="flex-wrap">
              <TabsTrigger value="identity">Identity</TabsTrigger>
              <TabsTrigger value="payload">Payload</TabsTrigger>
              <TabsTrigger value="requirements">Requirements</TabsTrigger>
              <TabsTrigger value="evidence">Evidence</TabsTrigger>
              <TabsTrigger value="scripts">Scripts</TabsTrigger>
              <TabsTrigger value="alerts">Alerts</TabsTrigger>
            </TabsList>
            <TabsContent value="identity">
              <FieldGroup>
                <FieldGroup className="grid gap-4 md:grid-cols-2">
                  <form.Field
                    name="version"
                    validators={{ onSubmit: requiredString("Version") }}
                    children={(field) => {
                      const invalid = field.state.meta.isTouched && !field.state.meta.isValid;
                      const error = firstErrorMessage(field.state.meta.errors);
                      return (
                        <Field data-invalid={invalid}>
                          <FieldLabel htmlFor="munki-package-version" required>
                            Version
                          </FieldLabel>
                          <Input
                            id="munki-package-version"
                            name={field.name}
                            value={field.state.value}
                            aria-invalid={invalid}
                            onBlur={field.handleBlur}
                            onChange={(event) => field.handleChange(event.target.value)}
                          />
                          {invalid && error ? <FieldError>{error}</FieldError> : null}
                        </Field>
                      );
                    }}
                  />
                  <SelectControl
                    id="munki-package-installer-type"
                    label="Installer Type"
                    value={values.installer_type}
                    options={MUNKI_INSTALLER_TYPE_OPTIONS}
                    onChange={(installer_type) => setField("installer_type", installer_type)}
                  />
                </FieldGroup>
                <Field>
                  <FieldLabel htmlFor="munki-package-notes">Notes</FieldLabel>
                  <Textarea
                    id="munki-package-notes"
                    value={values.notes}
                    onChange={(event) => setField("notes", event.target.value)}
                  />
                </Field>
              </FieldGroup>
            </TabsContent>
            <TabsContent value="payload">
              <FieldGroup>
                {values.installer_type !== "nopkg" || values.uninstall_method === "uninstall_package" ? (
                  <FieldSet>
                    <FieldLegend>Artifacts</FieldLegend>
                    <FieldGroup>
                      {values.installer_type !== "nopkg" ? (
                        <PackageFileField
                          id="munki-package-installer-file"
                          label="Installer"
                          description={installerArtifactLocation || "No installer artifact selected."}
                          icon={<FileArchive className="size-4" />}
                          file={installerFile}
                          onChange={onInstallerFileChange}
                        />
                      ) : null}
                      {values.uninstall_method === "uninstall_package" ? (
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
                ) : null}
                <FieldSet>
                  <FieldLegend>Installer</FieldLegend>
                  {values.installer_type !== "nopkg" ? (
                    <Field>
                      <FieldLabel htmlFor="munki-package-package-path">Package Path</FieldLabel>
                      <Input
                        id="munki-package-package-path"
                        value={values.package_path}
                        onChange={(event) => setField("package_path", event.target.value)}
                      />
                    </Field>
                  ) : null}
                  <FieldGroup className="grid gap-4 md:grid-cols-2">
                    <NumberField
                      id="munki-package-installed-size"
                      label="Installed Size"
                      value={values.installed_size}
                      onChange={(installed_size) => setField("installed_size", installed_size)}
                    />
                    <DateTimeField
                      id="munki-package-force-install-after"
                      label="Force Install After"
                      value={values.force_install_after_date}
                      onChange={(force_install_after_date) =>
                        setField("force_install_after_date", force_install_after_date)
                      }
                    />
                  </FieldGroup>
                  {values.installer_type !== "nopkg" ? (
                    <>
                      <XMLField
                        value={values.installer_choices_xml}
                        onChange={(installer_choices_xml) => setField("installer_choices_xml", installer_choices_xml)}
                      />
                      <InstallerEnvironmentEditor
                        rows={values.installer_environment}
                        onChange={(installer_environment) => setField("installer_environment", installer_environment)}
                      />
                    </>
                  ) : null}
                </FieldSet>
              </FieldGroup>
            </TabsContent>
            <TabsContent value="requirements">
              <FieldGroup>
                <FieldSet>
                  <FieldLegend>Compatibility</FieldLegend>
                  <Field>
                    <FieldLabel htmlFor="munki-package-minimum-munki-version">Minimum Munki Version</FieldLabel>
                    <Input
                      id="munki-package-minimum-munki-version"
                      value={values.minimum_munki_version}
                      onChange={(event) => setField("minimum_munki_version", event.target.value)}
                    />
                  </Field>
                  <FieldGroup className="grid gap-4 md:grid-cols-2">
                    <Field>
                      <FieldLabel htmlFor="munki-package-minimum-os">Minimum OS</FieldLabel>
                      <Input
                        id="munki-package-minimum-os"
                        value={values.minimum_os_version}
                        onChange={(event) => setField("minimum_os_version", event.target.value)}
                      />
                    </Field>
                    <Field>
                      <FieldLabel htmlFor="munki-package-maximum-os">Maximum OS</FieldLabel>
                      <Input
                        id="munki-package-maximum-os"
                        value={values.maximum_os_version}
                        onChange={(event) => setField("maximum_os_version", event.target.value)}
                      />
                    </Field>
                  </FieldGroup>
                  <ArchitectureEditor
                    values={values.supported_architectures}
                    onChange={(supported_architectures) => setField("supported_architectures", supported_architectures)}
                  />
                  <Field>
                    <FieldLabel htmlFor="munki-package-installable-condition">Installable Condition</FieldLabel>
                    <Textarea
                      id="munki-package-installable-condition"
                      value={values.installable_condition}
                      onChange={(event) => setField("installable_condition", event.target.value)}
                    />
                    <FieldDescription>
                      Munki predicate that must pass before this package version is considered.
                    </FieldDescription>
                  </Field>
                </FieldSet>
                <StringArrayEditor
                  legend="Blocking Applications"
                  addLabel="Application"
                  rows={values.blocking_applications}
                  onChange={(blocking_applications) => setField("blocking_applications", blocking_applications)}
                />
                <FieldSet>
                  <FieldLegend>Blocking Application Handling</FieldLegend>
                  <CheckboxControl
                    id="munki-package-blocking-applications-manual-quit-only"
                    label="Require manual quit"
                    description="Munki will show blocking apps but will not try to quit them automatically."
                    checked={values.blocking_applications_manual_quit_only}
                    onChange={(blocking_applications_manual_quit_only) =>
                      setField("blocking_applications_manual_quit_only", blocking_applications_manual_quit_only)
                    }
                  />
                  <Field>
                    <FieldLabel htmlFor="munki-package-blocking-applications-quit-script">Quit Script</FieldLabel>
                    <Textarea
                      id="munki-package-blocking-applications-quit-script"
                      value={values.blocking_applications_quit_script}
                      onChange={(event) => setField("blocking_applications_quit_script", event.target.value)}
                    />
                    <FieldDescription>Embedded script Munki uses instead of default app termination.</FieldDescription>
                  </Field>
                </FieldSet>
                <PackageReferenceEditor
                  legend="Requires"
                  rows={values.requires}
                  packageOptions={packageOptions}
                  onChange={(requires) => setField("requires", requires)}
                />
                <PackageReferenceEditor
                  legend="Update For"
                  rows={values.update_for}
                  packageOptions={packageOptions}
                  onChange={(update_for) => setField("update_for", update_for)}
                />
              </FieldGroup>
            </TabsContent>
            <TabsContent value="evidence">
              <FieldGroup>
                <InstallItemsEditor rows={values.installs} onChange={(installs) => setField("installs", installs)} />
                <ReceiptsEditor rows={values.receipts} onChange={(receipts) => setField("receipts", receipts)} />
                {values.installer_type === "copy_from_dmg" || values.uninstall_method === "remove_copied_items" ? (
                  <ItemsToCopyEditor
                    rows={values.items_to_copy}
                    onChange={(items_to_copy) => setField("items_to_copy", items_to_copy)}
                  />
                ) : null}
              </FieldGroup>
            </TabsContent>
            <TabsContent value="scripts">
              <ScriptTabs values={values} onChange={(key, value) => setField(key, value)} />
            </TabsContent>
            <TabsContent value="alerts">
              <FieldGroup>
                <AlertEditor
                  id="munki-package-preinstall-alert"
                  legend="Preinstall Alert"
                  alert={values.preinstall_alert}
                  onChange={(preinstall_alert) => setField("preinstall_alert", preinstall_alert)}
                />
                <AlertEditor
                  id="munki-package-preuninstall-alert"
                  legend="Preuninstall Alert"
                  alert={values.preuninstall_alert}
                  onChange={(preuninstall_alert) => setField("preuninstall_alert", preuninstall_alert)}
                />
                <FieldSet>
                  <FieldLegend>Behavior</FieldLegend>
                  <CheckboxControl
                    id="munki-package-eligible"
                    label="Available for targeting"
                    checked={values.eligible}
                    onChange={(eligible) => setField("eligible", eligible)}
                  />
                  <FieldGroup className="grid gap-4 md:grid-cols-3">
                    <CheckboxControl
                      id="munki-package-unattended-install"
                      label="Unattended install"
                      checked={values.unattended_install}
                      onChange={(unattended_install) => setField("unattended_install", unattended_install)}
                    />
                    <CheckboxControl
                      id="munki-package-unattended-uninstall"
                      label="Unattended uninstall"
                      checked={values.unattended_uninstall}
                      onChange={(unattended_uninstall) => setField("unattended_uninstall", unattended_uninstall)}
                    />
                  </FieldGroup>
                  <FieldGroup className="grid gap-4 md:grid-cols-2">
                    <SelectControl
                      id="munki-package-uninstall-method"
                      label="Uninstall Method"
                      value={values.uninstall_method}
                      options={MUNKI_UNINSTALL_METHOD_OPTIONS}
                      onChange={(uninstall_method) => setField("uninstall_method", uninstall_method)}
                    />
                    <SelectControl
                      id="munki-package-restart-action"
                      label="Restart Action"
                      value={values.restart_action}
                      options={MUNKI_RESTART_ACTION_OPTIONS}
                      onChange={(restart_action) => setField("restart_action", restart_action)}
                    />
                  </FieldGroup>
                  <FieldGroup className="grid gap-4 md:grid-cols-3">
                    <CheckboxControl
                      id="munki-package-on-demand"
                      label="On demand"
                      checked={values.on_demand}
                      onChange={(on_demand) => setField("on_demand", on_demand)}
                    />
                    <CheckboxControl
                      id="munki-package-precache"
                      label="Precache"
                      checked={values.precache}
                      onChange={(precache) => setField("precache", precache)}
                    />
                    <CheckboxControl
                      id="munki-package-autoremove"
                      label="Autoremove"
                      checked={values.autoremove}
                      onChange={(autoremove) => setField("autoremove", autoremove)}
                    />
                    <CheckboxControl
                      id="munki-package-apple-item"
                      label="Apple item"
                      checked={values.apple_item}
                      onChange={(apple_item) => setField("apple_item", apple_item)}
                    />
                    <CheckboxControl
                      id="munki-package-suppress-bundle-relocation"
                      label="Suppress bundle relocation"
                      checked={values.suppress_bundle_relocation}
                      onChange={(suppress_bundle_relocation) =>
                        setField("suppress_bundle_relocation", suppress_bundle_relocation)
                      }
                    />
                  </FieldGroup>
                </FieldSet>
              </FieldGroup>
            </TabsContent>
          </Tabs>
        );
      }}
    </form.Subscribe>
  );
}

export function PackageFormActions({ pending, softwareID }: { pending: boolean; softwareID: number | null }) {
  return (
    <div className="flex items-center gap-2">
      <Button type="submit" size="sm" disabled={pending}>
        {pending ? <Loader2 data-icon="inline-start" className="animate-spin" /> : null}
        Save
      </Button>
      <Button asChild type="button" variant="outline" size="sm">
        <Link to="/munki/software/$softwareId" params={{ softwareId: String(softwareID ?? "") }}>
          Cancel
        </Link>
      </Button>
    </div>
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

function SelectControl<T extends string>({
  id,
  label,
  value,
  options,
  onChange,
}: {
  id: string;
  label: string;
  value: T;
  options: Array<{ value: T; label: string }>;
  onChange: (value: T) => void;
}) {
  return (
    <Field>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <Select value={value} onValueChange={(next) => onChange(next as T)}>
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

function NumberField({
  id,
  label,
  value,
  onChange,
}: {
  id: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <Field>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <Input
        id={id}
        type="number"
        inputMode="numeric"
        min={0}
        value={value}
        onChange={(event) => onChange(event.target.value)}
      />
    </Field>
  );
}

function DateTimeField({
  id,
  label,
  value,
  onChange,
}: {
  id: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <Field>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <Input id={id} type="datetime-local" value={value} onChange={(event) => onChange(event.target.value)} />
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
    <Field>
      <FieldLabel>Supported Architectures</FieldLabel>
      <div className="grid gap-3 md:grid-cols-2">
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
      </div>
    </Field>
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
      <div className="flex flex-col gap-2">
        {rows.map((row, index) => (
          <div key={row.rowID} className="flex gap-2">
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
  rows,
  packageOptions,
  onChange,
}: {
  legend: string;
  rows: PackageReferenceRow[];
  packageOptions: MunkiPackage[];
  onChange: (rows: PackageReferenceRow[]) => void;
}) {
  return (
    <FieldSet>
      <FieldLegend>{legend}</FieldLegend>
      <div className="flex flex-col gap-3">
        {rows.map((row, index) => (
          <div key={row.rowID} className="grid gap-2 md:grid-cols-[minmax(0,1fr)_auto]">
            <Select
              value={row.package_id ? String(row.package_id) : "select"}
              onValueChange={(value) => {
                onChange(
                  replaceAt(rows, index, { ...row, package_id: value === "select" ? undefined : Number(value) }),
                );
              }}
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
          Dependency
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
      <div className="flex flex-col gap-2">
        {rows.map((row, index) => (
          <div key={row.rowID} className="grid gap-2 md:grid-cols-[minmax(0,12rem)_minmax(0,1fr)_auto]">
            <Input
              value={row.name}
              onChange={(event) => onChange(replaceAt(rows, index, { ...row, name: event.target.value }))}
            />
            <Input
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
      <div className="flex flex-col gap-4">
        {rows.map((row, index) => (
          <div key={row.rowID} className="flex flex-col gap-3 rounded-md border p-3">
            <div className="grid gap-2 md:grid-cols-[10rem_minmax(0,1fr)_auto]">
              <SelectControl
                id={`munki-install-item-type-${row.rowID}`}
                label="Type"
                value={row.type}
                options={MUNKI_INSTALL_ITEM_TYPE_OPTIONS}
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
              <div className="flex items-end">
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

function ReceiptsEditor({ rows, onChange }: { rows: ReceiptRow[]; onChange: (rows: ReceiptRow[]) => void }) {
  return (
    <FieldSet>
      <FieldLegend>Receipts</FieldLegend>
      <div className="flex flex-col gap-2">
        {rows.map((row, index) => (
          <div key={row.rowID} className="grid gap-2 md:grid-cols-[minmax(0,1fr)_10rem_8rem_auto]">
            <Input
              value={row.package_id}
              onChange={(event) => onChange(replaceAt(rows, index, { ...row, package_id: event.target.value }))}
            />
            <Input
              value={row.version ?? ""}
              onChange={(event) => onChange(replaceAt(rows, index, { ...row, version: event.target.value }))}
            />
            <Field orientation="horizontal" className="items-center">
              <Checkbox
                checked={row.optional === true}
                onCheckedChange={(checked) => onChange(replaceAt(rows, index, { ...row, optional: checked === true }))}
              />
              <FieldContent>
                <FieldLabel>Optional</FieldLabel>
              </FieldContent>
            </Field>
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
      <FieldLegend>Items To Copy</FieldLegend>
      <div className="flex flex-col gap-4">
        {rows.map((row, index) => (
          <div key={row.rowID} className="flex flex-col gap-3 rounded-md border p-3">
            <div className="grid gap-2 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto]">
              <Field>
                <FieldLabel htmlFor={`munki-copy-source-${row.rowID}`}>Source</FieldLabel>
                <Input
                  id={`munki-copy-source-${row.rowID}`}
                  value={row.source_item}
                  onChange={(event) => onChange(replaceAt(rows, index, { ...row, source_item: event.target.value }))}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor={`munki-copy-destination-${row.rowID}`}>Destination</FieldLabel>
                <Input
                  id={`munki-copy-destination-${row.rowID}`}
                  value={row.destination_path}
                  onChange={(event) =>
                    onChange(replaceAt(rows, index, { ...row, destination_path: event.target.value }))
                  }
                />
              </Field>
              <div className="flex items-end">
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

function ScriptTabs({
  values,
  onChange,
}: {
  values: Pick<PackageFormState, ScriptKey>;
  onChange: (key: ScriptKey, value: string) => void;
}) {
  return (
    <Tabs defaultValue={scriptFields[0].key} className="gap-4">
      <TabsList className="h-auto flex-wrap justify-start">
        {scriptFields.map((field) => (
          <TabsTrigger key={field.key} value={field.key}>
            {field.label}
          </TabsTrigger>
        ))}
      </TabsList>
      {scriptFields.map((field) => (
        <TabsContent key={field.key} value={field.key}>
          <ScriptField label={field.label} value={values[field.key]} onChange={(value) => onChange(field.key, value)} />
        </TabsContent>
      ))}
    </Tabs>
  );
}

function ScriptField({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return (
    <Field>
      <FieldLabel>{label}</FieldLabel>
      <CodeEditor value={value} onChange={onChange} className="[&_.cm-content]:min-h-40" placeholder="#!/bin/zsh" />
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
          <Field>
            <FieldLabel htmlFor={`${id}-detail`}>Detail</FieldLabel>
            <Textarea
              id={`${id}-detail`}
              value={alert.detail ?? ""}
              onChange={(event) => onChange({ ...alert, detail: event.target.value })}
            />
          </Field>
        </FieldGroup>
      ) : null}
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
