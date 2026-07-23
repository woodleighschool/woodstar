import { useState } from "react";

import { CodeEditor } from "@/components/editor/code-editor";
import { FormField } from "@/components/form-field";
import type { FormTabDefinition } from "@/components/form-tabs";
import { ScrollableTabsList } from "@/components/layout/scrollable-tabs";
import { FieldDescription, FieldGroup, FieldLegend, FieldSet } from "@/components/ui/field";
import { Tabs, TabsContent, TabsTrigger } from "@/components/ui/tabs";
import type { MunkiPackage, MunkiSoftware } from "@/lib/api";
import { assertNever } from "@/lib/utils";

import type { PackageEditorForm } from "./fields";
import {
  emptyInstallerEnvironmentRow,
  emptyInstallItemRow,
  emptyItemToCopyRow,
  emptyPackageReferenceRow,
  emptyReceiptRow,
} from "./form-adapter";
import { type PackageFormInput, scriptFields, type ScriptKey } from "./form-schema";
import {
  ArchitectureEditor,
  BlockingApplicationsEditor,
  InstallerEnvironmentEditor,
  InstallerFileField,
  InstallsTable,
  ItemsToCopyEditor,
  ReceiptsTable,
} from "./package-collection-editors";
import {
  AlertEditor,
  FormCheckboxField,
  FormCodeField,
  FormSwitchField,
  FormTextareaField,
  FormTextField,
  InstallerTypeField,
  RestartActionField,
  ScriptField,
  UninstallMethodField,
  VersionField,
} from "./package-form-controls";
import {
  PackageReferenceEditor,
  ParentSoftwareField,
  type SoftwareInfo,
  SoftwareSelector,
} from "./package-reference-editors";

// uninstall_script lives on the Uninstall tab; the rest are general-purpose hooks.
const generalScriptFields = scriptFields.filter((script) => script.key !== "uninstall_script");

export const packageFormTabs = [
  {
    value: "basic",
    label: "Basic Info",
    fields: [
      "software_id",
      "version",
      "installer_type",
      "installer_file",
      "restart_required",
      "restart_action",
      "force_install_after_date",
      "notes",
      "unattended_install",
      "unattended_uninstall",
      "on_demand",
      "autoremove",
      "uninstallable",
    ],
  },
  { value: "contents", label: "Contents", fields: ["installs", "receipts"] },
  {
    value: "requirements",
    label: "Requirements",
    fields: [
      "requires",
      "update_for",
      "minimum_munki_version",
      "minimum_os_version",
      "maximum_os_version",
      "installable_condition",
    ],
  },
  {
    value: "installation",
    label: "Installation",
    fields: [
      "items_to_copy",
      "blocking_applications_none",
      "blocking_applications",
      "blocking_applications_manual_quit_only",
      "blocking_applications_quit_script",
      "supported_architectures",
      "installer_choices_xml",
    ],
  },
  {
    value: "uninstall",
    label: "Uninstall",
    fields: ["uninstall_method", "uninstall_script"],
  },
  {
    value: "scripts",
    label: "Scripts",
    fields: generalScriptFields.map((script) => script.key),
  },
  {
    value: "alerts",
    label: "Alerts",
    fields: ["preinstall_alert", "preuninstall_alert"],
  },
  {
    value: "advanced",
    label: "Advanced",
    fields: [
      "package_path",
      "installed_size",
      "installer_environment",
      "precache",
      "apple_item",
      "suppress_bundle_relocation",
    ],
  },
] as const satisfies readonly (FormTabDefinition & { label: string })[];

export function PackageEditorTabContent({
  tab,
  form,
  softwareInfo,
  softwareOptions,
  softwareLoading,
  packageOptions,
  installerMetadata,
}: {
  tab: (typeof packageFormTabs)[number]["value"];
  form: PackageEditorForm;
  softwareInfo: SoftwareInfo | null;
  softwareOptions?: MunkiSoftware[];
  softwareLoading?: boolean;
  packageOptions: MunkiPackage[];
  installerMetadata?: MunkiPackage["installer_file"];
}) {
  switch (tab) {
    case "basic":
      return (
        <BasicInfoTab
          form={form}
          software={softwareInfo}
          softwareOptions={softwareOptions}
          softwareLoading={softwareLoading}
          installerMetadata={installerMetadata}
        />
      );
    case "contents":
      return <ContentsTab form={form} />;
    case "requirements":
      return <RequirementsTab form={form} packageOptions={packageOptions} />;
    case "installation":
      return <InstallationTab form={form} />;
    case "uninstall":
      return <UninstallTab form={form} />;
    case "scripts":
      return <ScriptsTab form={form} />;
    case "alerts":
      return <AlertsTab form={form} />;
    case "advanced":
      return <AdvancedTab form={form} />;
  }
  return assertNever(tab);
}

function BasicInfoTab({
  form,
  software,
  softwareOptions,
  softwareLoading,
  installerMetadata,
}: {
  form: PackageEditorForm;
  software: SoftwareInfo | null;
  softwareOptions?: MunkiSoftware[];
  softwareLoading?: boolean;
  installerMetadata?: MunkiPackage["installer_file"];
}) {
  return (
    <FieldGroup>
      {softwareOptions ? (
        <SoftwareSelector form={form} rows={softwareOptions} loading={softwareLoading === true} />
      ) : software ? (
        <ParentSoftwareField software={software} />
      ) : null}

      <VersionField form={form} />
      <InstallerTypeField form={form} />
      <FormCheckboxField
        form={form}
        name="on_demand"
        id="munki-package-on-demand"
        label="On demand"
        description="Use with Optional Installs for repeatable maintenance actions; Munki never considers the item installed."
      />
      <form.Subscribe selector={(state) => state.values.installer_type}>
        {(installerType) =>
          installerType === "nopkg" ? null : (
            <InstallerFileField form={form} metadata={installerMetadata} />
          )
        }
      </form.Subscribe>
      <FormSwitchField
        form={form}
        name="restart_required"
        id="munki-package-restart-required"
        label="Restart required"
      />
      <form.Subscribe selector={(state) => state.values.restart_required}>
        {(restartRequired) => (restartRequired ? <RestartActionField form={form} /> : null)}
      </form.Subscribe>
      <FormTextField
        form={form}
        name="force_install_after_date"
        id="munki-package-force-install-after"
        label="Force Install After"
        description="Client-local deadline that can force logout or restart after it passes."
        type="datetime-local"
      />
      <FormTextareaField
        form={form}
        name="notes"
        id="munki-package-notes"
        label="Notes"
        description="Admin notes excluded from Munki catalogs."
      />

      <FieldSet>
        <FieldLegend>Behavior</FieldLegend>
        <FieldDescription>
          Controls how Munki installs, removes, and retires this package.
        </FieldDescription>
        <FieldGroup data-slot="checkbox-group">
          <FormCheckboxField
            form={form}
            name="unattended_install"
            id="munki-package-unattended-install"
            label="Unattended install"
            description="Munki can install without notifying the current GUI user."
          />
          <FormCheckboxField
            form={form}
            name="unattended_uninstall"
            id="munki-package-unattended-uninstall"
            label="Unattended uninstall"
            description="Munki can uninstall without notifying the current GUI user."
          />
          <FormCheckboxField
            form={form}
            name="autoremove"
            id="munki-package-autoremove"
            label="Autoremove"
            description="Remove the item from hosts where it is not a managed install."
          />
          <FormCheckboxField
            form={form}
            name="uninstallable"
            id="munki-package-uninstallable"
            label="Uninstallable"
            description="Allow removal using the configured uninstall method."
          />
        </FieldGroup>
      </FieldSet>
    </FieldGroup>
  );
}

function ContentsTab({ form }: { form: PackageEditorForm }) {
  return (
    <FieldGroup>
      <form.Field name="installs" mode="array">
        {(field) => (
          <FormField field={field}>
            {(control) => (
              <div {...control} tabIndex={-1}>
                <InstallsTable
                  rows={field.state.value}
                  onAdd={() => field.pushValue(emptyInstallItemRow())}
                  onReplace={(index, row) => field.replaceValue(index, row)}
                  onRemove={(index) => field.removeValue(index)}
                />
              </div>
            )}
          </FormField>
        )}
      </form.Field>
      <form.Field name="receipts" mode="array">
        {(field) => (
          <FormField field={field}>
            {(control) => (
              <div {...control} tabIndex={-1}>
                <ReceiptsTable
                  rows={field.state.value}
                  onAdd={() => field.pushValue(emptyReceiptRow())}
                  onReplace={(index, row) => field.replaceValue(index, row)}
                  onRemove={(index) => field.removeValue(index)}
                />
              </div>
            )}
          </FormField>
        )}
      </form.Field>
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
      <form.Field name="requires" mode="array">
        {(field) => (
          <FormField field={field}>
            {(control) => (
              <div {...control} tabIndex={-1}>
                <PackageReferenceEditor
                  legend="Requires"
                  description="Installs these packages first."
                  addLabel="Add requirement"
                  rows={field.state.value}
                  packageOptions={packageOptions}
                  onAdd={() => field.pushValue(emptyPackageReferenceRow())}
                  onReplace={(index, row) => field.replaceValue(index, row)}
                  onRemove={(index) => field.removeValue(index)}
                />
              </div>
            )}
          </FormField>
        )}
      </form.Field>
      <form.Field name="update_for" mode="array">
        {(field) => (
          <FormField field={field}>
            {(control) => (
              <div {...control} tabIndex={-1}>
                <PackageReferenceEditor
                  legend="Update For"
                  description="Treats this package as an update for the selected packages."
                  addLabel="Add update target"
                  rows={field.state.value}
                  packageOptions={packageOptions}
                  onAdd={() => field.pushValue(emptyPackageReferenceRow())}
                  onReplace={(index, row) => field.replaceValue(index, row)}
                  onRemove={(index) => field.removeValue(index)}
                />
              </div>
            )}
          </FormField>
        )}
      </form.Field>
      <FieldSet>
        <FieldLegend>Compatibility</FieldLegend>
        <FieldDescription>Restricts installation to matching client versions.</FieldDescription>
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
          description="NSPredicate evaluated on the client; false prevents installation."
          minHeight="[&_.cm-content]:min-h-32"
        />
      </FieldSet>
    </FieldGroup>
  );
}

function InstallationTab({ form }: { form: PackageEditorForm }) {
  return (
    <FieldGroup>
      <form.Field name="items_to_copy" mode="array">
        {(field) => (
          <FormField field={field}>
            {(control) => (
              <div {...control} tabIndex={-1}>
                <ItemsToCopyEditor
                  rows={field.state.value}
                  onAdd={() => field.pushValue(emptyItemToCopyRow())}
                  onReplace={(index, row) => field.replaceValue(index, row)}
                  onRemove={(index) => field.removeValue(index)}
                />
              </div>
            )}
          </FormField>
        )}
      </form.Field>

      <BlockingApplicationsEditor form={form} />
      <FieldSet>
        <FieldLegend>Blocking Application Handling</FieldLegend>
        <FieldDescription>
          Controls how Munki handles applications that must close before installation.
        </FieldDescription>
        <FieldGroup>
          <FormCheckboxField
            form={form}
            name="blocking_applications_manual_quit_only"
            id="munki-package-blocking-applications-manual-quit-only"
            label="Require manual quit"
            description="Prevent Munki from attempting to quit blocking applications."
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

      <form.Field name="supported_architectures">
        {(field) => (
          <ArchitectureEditor
            values={field.state.value}
            onChange={(values) => field.handleChange(values)}
          />
        )}
      </form.Field>

      <form.Field name="installer_choices_xml">
        {(field) => (
          <FormField
            field={field}
            label="Installer Choices XML"
            htmlFor="installer-choices-xml"
            description="ChoiceChangesXML applied when installing an Apple metapackage."
          >
            {(control) => (
              <div {...control} tabIndex={-1}>
                <CodeEditor
                  value={field.state.value}
                  onChange={field.handleChange}
                  lineNumbers={false}
                  className="[&_.cm-content]:min-h-28"
                />
              </div>
            )}
          </FormField>
        )}
      </form.Field>
    </FieldGroup>
  );
}

function UninstallTab({ form }: { form: PackageEditorForm }) {
  return (
    <FieldGroup>
      <form.Subscribe selector={(state) => state.values.uninstall_method}>
        {(uninstallMethod) => (
          <>
            <div className="max-w-sm">
              <UninstallMethodField form={form} />
            </div>
            {uninstallMethod === "uninstall_script" ? (
              <form.Field name="uninstall_script">
                {(field) => (
                  <FormField field={field} label="Uninstall Script">
                    {(control) => (
                      <div {...control} tabIndex={-1}>
                        <ScriptField value={field.state.value} onChange={field.handleChange} />
                      </div>
                    )}
                  </FormField>
                )}
              </form.Field>
            ) : null}
          </>
        )}
      </form.Subscribe>
    </FieldGroup>
  );
}

function ScriptsTab({ form }: { form: PackageEditorForm }) {
  return (
    <form.Subscribe selector={(state) => state.values}>
      {(values) => (
        <ScriptsEditor values={values} onChange={(key, value) => form.setFieldValue(key, value)} />
      )}
    </form.Subscribe>
  );
}

function ScriptsEditor({
  values,
  onChange,
}: {
  values: Pick<PackageFormInput, ScriptKey>;
  onChange: (key: ScriptKey, value: string) => void;
}) {
  const [active, setActive] = useState<ScriptKey>(generalScriptFields[0].key);

  return (
    <Tabs
      value={active}
      onValueChange={(value) => {
        if (typeof value === "string" && isGeneralScriptKey(value)) setActive(value);
      }}
      className="max-w-3xl gap-4"
    >
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

function isGeneralScriptKey(value: string): value is ScriptKey {
  return generalScriptFields.some((script) => script.key === value);
}

function AlertsTab({ form }: { form: PackageEditorForm }) {
  return (
    <FieldGroup>
      <form.Field name="preinstall_alert">
        {(field) => (
          <AlertEditor
            id="munki-package-preinstall-alert"
            legend="Preinstall Alert"
            alert={field.state.value}
            onChange={(alert) => field.handleChange(alert)}
          />
        )}
      </form.Field>
      <form.Field name="preuninstall_alert">
        {(field) => (
          <AlertEditor
            id="munki-package-preuninstall-alert"
            legend="Preuninstall Alert"
            alert={field.state.value}
            onChange={(alert) => field.handleChange(alert)}
          />
        )}
      </form.Field>
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
            description="Package location inside the mounted disk image."
          />
          <FormTextField
            form={form}
            name="installed_size"
            id="munki-package-installed-size"
            label="Installed Size"
            description="Kilobytes used for the client free-space check."
            type="number"
            inputMode="numeric"
          />
        </FieldGroup>
        <form.Field name="installer_environment" mode="array">
          {(field) => (
            <FormField field={field}>
              {(control) => (
                <div {...control} tabIndex={-1}>
                  <InstallerEnvironmentEditor
                    rows={field.state.value}
                    onAdd={() => field.pushValue(emptyInstallerEnvironmentRow())}
                    onReplace={(index, row) => field.replaceValue(index, row)}
                    onRemove={(index) => field.removeValue(index)}
                  />
                </div>
              )}
            </FormField>
          )}
        </form.Field>
      </FieldSet>

      <FieldSet>
        <FieldLegend>Flags</FieldLegend>
        <FieldGroup data-slot="checkbox-group">
          <FormCheckboxField
            form={form}
            name="precache"
            id="munki-package-precache"
            label="Precache"
            description="Download an Optional Install before the user selects it."
          />
          <FormCheckboxField
            form={form}
            name="apple_item"
            id="munki-package-apple-item"
            label="Apple item"
            description="Treat this package as an Apple update."
          />
          <FormCheckboxField
            form={form}
            name="suppress_bundle_relocation"
            id="munki-package-suppress-bundle-relocation"
            label="Suppress bundle relocation"
            description="Prevent legacy bundle packages from updating a moved copy."
          />
        </FieldGroup>
      </FieldSet>
    </FieldGroup>
  );
}
