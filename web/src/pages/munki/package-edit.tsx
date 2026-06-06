import { xml } from "@codemirror/lang-xml";
import type { Extension } from "@codemirror/state";
import { useNavigate } from "@tanstack/react-router";
import { FileArchive, Plus, Trash2 } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { z } from "zod";

import { CodeEditor } from "@/components/editor/code-editor";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { EditableMunkiIcon } from "@/components/munki/editable-munki-icon";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Field, FieldContent, FieldGroup, FieldLabel, FieldLegend, FieldSet } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useCreateMunkiArtifact, useCreateMunkiArtifactUpload } from "@/hooks/munki/artifacts";
import {
  useCreateMunkiPackage,
  useMunkiPackage,
  useMunkiPackages,
  useUpdateMunkiPackage,
  type MunkiPackage,
  type MunkiPackageMutation,
} from "@/hooks/munki/packages";
import { useMunkiSoftwareTitle, useMunkiSoftwareTitles } from "@/hooks/munki/software-titles";
import type {
  PackageAlert,
  PackageInstallerEnvironmentVariable,
  PackageInstallItem,
  PackageItemToCopy,
  PackageReceipt,
  PackageReference,
} from "@/lib/api-client/types.gen";
import { fieldErrors, requiredString } from "@/lib/form-validation";
import { MAX_PAGE_SIZE } from "@/lib/pagination";

import {
  CheckboxField,
  DatalistTextField,
  FileField,
  FormActions,
  MutationError,
  SelectField,
  TextAreaField,
  TextField,
} from "./edit-shared";
import {
  optionalText,
  runSubmit,
  uniqueOptions,
  uploadSelectedArtifact,
  usePackageIDParam,
  useSoftwareIDParam,
} from "./edit-utils";
import {
  MUNKI_INSTALL_ITEM_TYPE_OPTIONS,
  MUNKI_INSTALLER_TYPE_OPTIONS,
  MUNKI_RESTART_ACTION_OPTIONS,
  MUNKI_UNINSTALL_METHOD_OPTIONS,
  type MunkiInstallerType,
  type MunkiRestartAction,
  type MunkiUninstallMethod,
} from "./shared";

const xmlExtensions: Extension[] = [xml()];

type Architecture = "arm64" | "x86_64";
type ScriptKey =
  | "installcheck_script"
  | "uninstallcheck_script"
  | "preinstall_script"
  | "postinstall_script"
  | "preuninstall_script"
  | "postuninstall_script"
  | "uninstall_script"
  | "version_script";

interface PackageReferenceRow extends PackageReference {
  rowID: string;
}

interface StringRow {
  rowID: string;
  value: string;
}

interface InstallerEnvironmentRow extends PackageInstallerEnvironmentVariable {
  rowID: string;
}

interface InstallItemRow extends PackageInstallItem {
  rowID: string;
}

interface ReceiptRow extends PackageReceipt {
  rowID: string;
}

interface ItemToCopyRow extends PackageItemToCopy {
  rowID: string;
}

interface PackageFormState {
  name: string;
  version: string;
  display_name: string;
  description: string;
  category: string;
  developer: string;
  installer_type: MunkiInstallerType;
  uninstall_method: MunkiUninstallMethod;
  custom_uninstall_method: string;
  restart_action: MunkiRestartAction;
  minimum_munki_version: string;
  minimum_os_version: string;
  maximum_os_version: string;
  supported_architectures: Architecture[];
  blocking_applications: StringRow[];
  requires: PackageReferenceRow[];
  update_for: PackageReferenceRow[];
  eligible: boolean;
  unattended_install: boolean;
  unattended_uninstall: boolean;
  uninstallable: boolean;
  on_demand: boolean;
  precache: boolean;
  autoremove: boolean;
  apple_item: boolean;
  suppress_bundle_relocation: boolean;
  force_install_after_date: string;
  installed_size: string;
  payload_identifier: string;
  package_path: string;
  installer_choices_xml: string;
  installer_environment: InstallerEnvironmentRow[];
  installs: InstallItemRow[];
  receipts: ReceiptRow[];
  items_to_copy: ItemToCopyRow[];
  notes: string;
  installcheck_script: string;
  uninstallcheck_script: string;
  preinstall_script: string;
  postinstall_script: string;
  preuninstall_script: string;
  postuninstall_script: string;
  uninstall_script: string;
  version_script: string;
  preinstall_alert: PackageAlert;
  preuninstall_alert: PackageAlert;
}

const scriptFields: { key: ScriptKey; label: string }[] = [
  { key: "installcheck_script", label: "Install Check" },
  { key: "uninstallcheck_script", label: "Uninstall Check" },
  { key: "preinstall_script", label: "Preinstall" },
  { key: "postinstall_script", label: "Postinstall" },
  { key: "preuninstall_script", label: "Preuninstall" },
  { key: "postuninstall_script", label: "Postuninstall" },
  { key: "uninstall_script", label: "Uninstall" },
  { key: "version_script", label: "Version" },
];

const packageSchema = z.object({
  name: requiredString("Name"),
  version: requiredString("Version"),
});

export function MunkiPackageNewPage() {
  const navigate = useNavigate();
  const softwareID = useSoftwareIDParam();
  const software = useMunkiSoftwareTitle(softwareID);
  const create = useCreateMunkiPackage();
  const createUpload = useCreateMunkiArtifactUpload();
  const createArtifact = useCreateMunkiArtifact();
  const titles = useMunkiSoftwareTitles({ page_size: MAX_PAGE_SIZE, sort: "name.asc" });
  const packages = useMunkiPackages({ page_size: MAX_PAGE_SIZE, sort: "name.asc" });
  const [installerFile, setInstallerFile] = useState<File | null>(null);
  const [uninstallerFile, setUninstallerFile] = useState<File | null>(null);
  const [iconFile, setIconFile] = useState<File | null>(null);
  const [form, setForm] = useState<PackageFormState>(emptyPackageForm());
  const [showErrors, setShowErrors] = useState(false);
  const parsed = useMemo(() => packageSchema.safeParse(form), [form]);
  const errors = useMemo(() => fieldErrors(parsed), [parsed]);
  const categoryOptions = useMemo(
    () => uniqueOptions((titles.data?.items ?? []).map((item) => item.category)),
    [titles.data?.items],
  );
  const developerOptions = useMemo(
    () => uniqueOptions((titles.data?.items ?? []).map((item) => item.developer)),
    [titles.data?.items],
  );

  useEffect(() => {
    if (!software.data) return;
    setForm((current) => ({
      ...current,
      name: current.name || software.data.name,
      display_name: current.display_name || software.data.display_name,
      description: current.description || software.data.description,
      category: current.category || software.data.category,
      developer: current.developer || software.data.developer,
    }));
  }, [software.data]);

  async function submit() {
    const next = packageSchema.safeParse(form);
    if (!next.success || softwareID === null) {
      setShowErrors(true);
      return;
    }
    const installerArtifact = installerFile
      ? await uploadSelectedArtifact(installerFile, "package", createUpload.mutateAsync, createArtifact.mutateAsync)
      : null;
    const uninstallerArtifact = uninstallerFile
      ? await uploadSelectedArtifact(uninstallerFile, "package", createUpload.mutateAsync, createArtifact.mutateAsync)
      : null;
    const iconArtifact = iconFile
      ? await uploadSelectedArtifact(iconFile, "icon", createUpload.mutateAsync, createArtifact.mutateAsync)
      : null;
    await create.mutateAsync(
      packageMutationFromForm(form, softwareID, {
        installerArtifactID: installerArtifact?.id,
        uninstallerArtifactID: uninstallerArtifact?.id,
        iconArtifactID: iconArtifact?.id,
      }),
    );
    void navigate({ to: "/munki/software-titles/$softwareId", params: { softwareId: String(softwareID) } });
  }

  return (
    <PageShell asChild>
      <form noValidate onSubmit={(event) => runSubmit(event, submit)}>
        <PageHeader
          title="New Package"
          leading={
            <EditableMunkiIcon
              title="package icon"
              fallbackIconUrl={software.data?.icon_url}
              file={iconFile}
              clearable={!!iconFile}
              onFileChange={setIconFile}
              onClear={() => setIconFile(null)}
            />
          }
        />
        <MutationError
          title="Failed to Create Package"
          message={
            create.error?.message ??
            createUpload.error?.message ??
            createArtifact.error?.message ??
            software.error?.message
          }
        />
        <PackageEditorTabs
          form={form}
          errors={showErrors ? errors : {}}
          categoryOptions={categoryOptions}
          developerOptions={developerOptions}
          packageOptions={packages.data?.items ?? []}
          installerFile={installerFile}
          uninstallerFile={uninstallerFile}
          installerArtifactLocation=""
          uninstallerArtifactLocation=""
          onInstallerFileChange={setInstallerFile}
          onUninstallerFileChange={setUninstallerFile}
          onChange={setForm}
        />
        <FormActions
          pending={create.isPending || createUpload.isPending || createArtifact.isPending}
          cancelTo="/munki/software-titles/$softwareId"
          cancelParams={{ softwareId: String(softwareID ?? "") }}
        />
      </form>
    </PageShell>
  );
}

export function MunkiPackageEditPage() {
  const navigate = useNavigate();
  const softwareID = useSoftwareIDParam();
  const packageID = usePackageIDParam();
  const software = useMunkiSoftwareTitle(softwareID);
  const pkg = useMunkiPackage(packageID);
  const update = useUpdateMunkiPackage();
  const createUpload = useCreateMunkiArtifactUpload();
  const createArtifact = useCreateMunkiArtifact();
  const titles = useMunkiSoftwareTitles({ page_size: MAX_PAGE_SIZE, sort: "name.asc" });
  const packages = useMunkiPackages({ page_size: MAX_PAGE_SIZE, sort: "name.asc" });
  const [installerFile, setInstallerFile] = useState<File | null>(null);
  const [uninstallerFile, setUninstallerFile] = useState<File | null>(null);
  const [iconFile, setIconFile] = useState<File | null>(null);
  const [iconCleared, setIconCleared] = useState(false);
  const [form, setForm] = useState<PackageFormState>(emptyPackageForm());
  const [showErrors, setShowErrors] = useState(false);
  const parsed = useMemo(() => packageSchema.safeParse(form), [form]);
  const errors = useMemo(() => fieldErrors(parsed), [parsed]);
  const categoryOptions = useMemo(
    () => uniqueOptions((titles.data?.items ?? []).map((item) => item.category)),
    [titles.data?.items],
  );
  const developerOptions = useMemo(
    () => uniqueOptions((titles.data?.items ?? []).map((item) => item.developer)),
    [titles.data?.items],
  );

  useEffect(() => {
    if (!pkg.data) return;
    setForm(packageFormFromPackage(pkg.data));
    setInstallerFile(null);
    setUninstallerFile(null);
    setIconFile(null);
    setIconCleared(false);
  }, [pkg.data]);

  async function submit() {
    const next = packageSchema.safeParse(form);
    if (!next.success || softwareID === null || packageID === null) {
      setShowErrors(true);
      return;
    }
    const installerArtifact = installerFile
      ? await uploadSelectedArtifact(installerFile, "package", createUpload.mutateAsync, createArtifact.mutateAsync)
      : null;
    const uninstallerArtifact = uninstallerFile
      ? await uploadSelectedArtifact(uninstallerFile, "package", createUpload.mutateAsync, createArtifact.mutateAsync)
      : null;
    const iconArtifact = iconFile
      ? await uploadSelectedArtifact(iconFile, "icon", createUpload.mutateAsync, createArtifact.mutateAsync)
      : null;
    const body = packageMutationFromForm(form, softwareID, {
      installerArtifactID: installerArtifact?.id ?? pkg.data?.installer_artifact_id,
      uninstallerArtifactID: uninstallerArtifact?.id ?? pkg.data?.uninstaller_artifact_id,
      iconArtifactID: iconArtifact?.id ?? (iconCleared ? undefined : pkg.data?.icon_artifact_id),
    });
    await update.mutateAsync({ id: packageID, body });
    void navigate({ to: "/munki/software-titles/$softwareId", params: { softwareId: String(softwareID) } });
  }

  const packageIconURL = iconCleared || !pkg.data?.icon_artifact_id ? undefined : pkg.data.icon_url;
  const packageIconClearable = !!iconFile || (!iconCleared && !!pkg.data?.icon_artifact_id);

  return (
    <PageShell asChild>
      <form noValidate onSubmit={(event) => runSubmit(event, submit)}>
        <PageHeader
          title="Edit Package"
          leading={
            <EditableMunkiIcon
              title="package icon"
              iconUrl={packageIconURL}
              fallbackIconUrl={software.data?.icon_url}
              file={iconFile}
              clearable={packageIconClearable}
              onFileChange={(file) => {
                setIconFile(file);
                setIconCleared(false);
              }}
              onClear={() => {
                setIconFile(null);
                setIconCleared(!!pkg.data?.icon_artifact_id);
              }}
            />
          }
        />
        <MutationError
          title="Failed to Update Package"
          message={
            update.error?.message ??
            createUpload.error?.message ??
            createArtifact.error?.message ??
            pkg.error?.message ??
            software.error?.message
          }
        />
        <PackageEditorTabs
          form={form}
          errors={showErrors ? errors : {}}
          categoryOptions={categoryOptions}
          developerOptions={developerOptions}
          packageOptions={(packages.data?.items ?? []).filter((item) => item.id !== packageID)}
          installerFile={installerFile}
          uninstallerFile={uninstallerFile}
          installerArtifactLocation={pkg.data?.installer_artifact_location ?? ""}
          uninstallerArtifactLocation={pkg.data?.uninstaller_artifact_location ?? ""}
          onInstallerFileChange={setInstallerFile}
          onUninstallerFileChange={setUninstallerFile}
          onChange={setForm}
        />
        <FormActions
          pending={update.isPending || createUpload.isPending || createArtifact.isPending || pkg.isLoading}
          cancelTo="/munki/software-titles/$softwareId"
          cancelParams={{ softwareId: String(softwareID ?? "") }}
        />
      </form>
    </PageShell>
  );
}

function PackageEditorTabs({
  form,
  errors,
  categoryOptions,
  developerOptions,
  packageOptions,
  installerFile,
  uninstallerFile,
  installerArtifactLocation,
  uninstallerArtifactLocation,
  onInstallerFileChange,
  onUninstallerFileChange,
  onChange,
}: {
  form: PackageFormState;
  errors: Record<string, string | undefined>;
  categoryOptions: string[];
  developerOptions: string[];
  packageOptions: MunkiPackage[];
  installerFile: File | null;
  uninstallerFile: File | null;
  installerArtifactLocation: string;
  uninstallerArtifactLocation: string;
  onInstallerFileChange: (file: File | null) => void;
  onUninstallerFileChange: (file: File | null) => void;
  onChange: (next: PackageFormState) => void;
}) {
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
          <TextField
            id="munki-package-name"
            label="Name"
            required
            value={form.name}
            error={errors.name}
            onChange={(name) => onChange({ ...form, name })}
          />
          <FieldGroup className="grid gap-4 md:grid-cols-2">
            <TextField
              id="munki-package-version"
              label="Version"
              required
              value={form.version}
              error={errors.version}
              onChange={(version) => onChange({ ...form, version })}
            />
            <TextField
              id="munki-package-display-name"
              label="Display Name"
              value={form.display_name}
              onChange={(display_name) => onChange({ ...form, display_name })}
            />
          </FieldGroup>
          <TextAreaField
            id="munki-package-description"
            label="Description"
            value={form.description}
            onChange={(description) => onChange({ ...form, description })}
          />
          <FieldGroup className="grid gap-4 md:grid-cols-3">
            <DatalistTextField
              id="munki-package-category"
              label="Category"
              value={form.category}
              options={categoryOptions}
              onChange={(category) => onChange({ ...form, category })}
            />
            <DatalistTextField
              id="munki-package-developer"
              label="Developer"
              value={form.developer}
              options={developerOptions}
              onChange={(developer) => onChange({ ...form, developer })}
            />
            <SelectField
              id="munki-package-installer-type"
              label="Installer Type"
              value={form.installer_type}
              options={MUNKI_INSTALLER_TYPE_OPTIONS}
              onChange={(installer_type) => onChange({ ...form, installer_type })}
            />
          </FieldGroup>
          <TextAreaField
            id="munki-package-notes"
            label="Notes"
            value={form.notes}
            onChange={(notes) => onChange({ ...form, notes })}
          />
        </FieldGroup>
      </TabsContent>
      <TabsContent value="payload">
        <FieldGroup>
          <FieldSet>
            <FieldLegend>Artifacts</FieldLegend>
            <FieldGroup>
              <FileField
                id="munki-package-installer-file"
                label="Installer"
                description={installerArtifactLocation || "No installer artifact selected."}
                icon={<FileArchive className="size-4" />}
                file={installerFile}
                onChange={onInstallerFileChange}
              />
              <FileField
                id="munki-package-uninstaller-file"
                label="Uninstaller"
                description={uninstallerArtifactLocation || "No uninstaller artifact selected."}
                icon={<FileArchive className="size-4" />}
                file={uninstallerFile}
                onChange={onUninstallerFileChange}
              />
            </FieldGroup>
          </FieldSet>
          <FieldSet>
            <FieldLegend>Installer</FieldLegend>
            <FieldGroup className="grid gap-4 md:grid-cols-2">
              <TextField
                id="munki-package-payload-identifier"
                label="Payload Identifier"
                value={form.payload_identifier}
                onChange={(payload_identifier) => onChange({ ...form, payload_identifier })}
              />
              <TextField
                id="munki-package-package-path"
                label="Package Path"
                value={form.package_path}
                onChange={(package_path) => onChange({ ...form, package_path })}
              />
            </FieldGroup>
            <FieldGroup className="grid gap-4 md:grid-cols-2">
              <NumberField
                id="munki-package-installed-size"
                label="Installed Size"
                value={form.installed_size}
                onChange={(installed_size) => onChange({ ...form, installed_size })}
              />
              <DateTimeField
                id="munki-package-force-install-after"
                label="Force Install After"
                value={form.force_install_after_date}
                onChange={(force_install_after_date) => onChange({ ...form, force_install_after_date })}
              />
            </FieldGroup>
            <XMLField
              value={form.installer_choices_xml}
              onChange={(installer_choices_xml) => onChange({ ...form, installer_choices_xml })}
            />
            <InstallerEnvironmentEditor
              rows={form.installer_environment}
              onChange={(installer_environment) => onChange({ ...form, installer_environment })}
            />
          </FieldSet>
        </FieldGroup>
      </TabsContent>
      <TabsContent value="requirements">
        <FieldGroup>
          <FieldSet>
            <FieldLegend>Compatibility</FieldLegend>
            <TextField
              id="munki-package-minimum-munki-version"
              label="Minimum Munki Version"
              value={form.minimum_munki_version}
              onChange={(minimum_munki_version) => onChange({ ...form, minimum_munki_version })}
            />
            <FieldGroup className="grid gap-4 md:grid-cols-2">
              <TextField
                id="munki-package-minimum-os"
                label="Minimum OS"
                value={form.minimum_os_version}
                onChange={(minimum_os_version) => onChange({ ...form, minimum_os_version })}
              />
              <TextField
                id="munki-package-maximum-os"
                label="Maximum OS"
                value={form.maximum_os_version}
                onChange={(maximum_os_version) => onChange({ ...form, maximum_os_version })}
              />
            </FieldGroup>
            <ArchitectureEditor
              values={form.supported_architectures}
              onChange={(supported_architectures) => onChange({ ...form, supported_architectures })}
            />
          </FieldSet>
          <StringArrayEditor
            legend="Blocking Applications"
            addLabel="Application"
            rows={form.blocking_applications}
            onChange={(blocking_applications) => onChange({ ...form, blocking_applications })}
          />
          <PackageReferenceEditor
            legend="Requires"
            rows={form.requires}
            packageOptions={packageOptions}
            onChange={(requires) => onChange({ ...form, requires })}
          />
          <PackageReferenceEditor
            legend="Update For"
            rows={form.update_for}
            packageOptions={packageOptions}
            onChange={(update_for) => onChange({ ...form, update_for })}
          />
        </FieldGroup>
      </TabsContent>
      <TabsContent value="evidence">
        <FieldGroup>
          <InstallItemsEditor rows={form.installs} onChange={(installs) => onChange({ ...form, installs })} />
          <ReceiptsEditor rows={form.receipts} onChange={(receipts) => onChange({ ...form, receipts })} />
          <ItemsToCopyEditor
            rows={form.items_to_copy}
            onChange={(items_to_copy) => onChange({ ...form, items_to_copy })}
          />
        </FieldGroup>
      </TabsContent>
      <TabsContent value="scripts">
        <ScriptTabs values={form} onChange={(key, value) => onChange({ ...form, [key]: value })} />
      </TabsContent>
      <TabsContent value="alerts">
        <FieldGroup>
          <AlertEditor
            id="munki-package-preinstall-alert"
            legend="Preinstall Alert"
            alert={form.preinstall_alert}
            onChange={(preinstall_alert) => onChange({ ...form, preinstall_alert })}
          />
          <AlertEditor
            id="munki-package-preuninstall-alert"
            legend="Preuninstall Alert"
            alert={form.preuninstall_alert}
            onChange={(preuninstall_alert) => onChange({ ...form, preuninstall_alert })}
          />
          <FieldSet>
            <FieldLegend>Behavior</FieldLegend>
            <CheckboxField
              id="munki-package-eligible"
              label="Available for assignment"
              checked={form.eligible}
              onChange={(eligible) => onChange({ ...form, eligible })}
            />
            <FieldGroup className="grid gap-4 md:grid-cols-3">
              <CheckboxField
                id="munki-package-unattended-install"
                label="Unattended install"
                checked={form.unattended_install}
                onChange={(unattended_install) => onChange({ ...form, unattended_install })}
              />
              <CheckboxField
                id="munki-package-unattended-uninstall"
                label="Unattended uninstall"
                checked={form.unattended_uninstall}
                onChange={(unattended_uninstall) => onChange({ ...form, unattended_uninstall })}
              />
              <CheckboxField
                id="munki-package-uninstallable"
                label="Uninstallable"
                checked={form.uninstallable}
                onChange={(uninstallable) => onChange({ ...form, uninstallable })}
              />
            </FieldGroup>
            <FieldGroup className="grid gap-4 md:grid-cols-2">
              <SelectField
                id="munki-package-uninstall-method"
                label="Uninstall Method"
                value={form.uninstall_method}
                options={MUNKI_UNINSTALL_METHOD_OPTIONS}
                onChange={(uninstall_method) => onChange({ ...form, uninstall_method })}
              />
              <SelectField
                id="munki-package-restart-action"
                label="Restart Action"
                value={form.restart_action}
                options={MUNKI_RESTART_ACTION_OPTIONS}
                onChange={(restart_action) => onChange({ ...form, restart_action })}
              />
            </FieldGroup>
            {form.uninstall_method === "custom" ? (
              <TextField
                id="munki-package-custom-uninstall-method"
                label="Custom Uninstall Method"
                value={form.custom_uninstall_method}
                onChange={(custom_uninstall_method) => onChange({ ...form, custom_uninstall_method })}
              />
            ) : null}
            <FieldGroup className="grid gap-4 md:grid-cols-3">
              <CheckboxField
                id="munki-package-on-demand"
                label="On demand"
                checked={form.on_demand}
                onChange={(on_demand) => onChange({ ...form, on_demand })}
              />
              <CheckboxField
                id="munki-package-precache"
                label="Precache"
                checked={form.precache}
                onChange={(precache) => onChange({ ...form, precache })}
              />
              <CheckboxField
                id="munki-package-autoremove"
                label="Autoremove"
                checked={form.autoremove}
                onChange={(autoremove) => onChange({ ...form, autoremove })}
              />
              <CheckboxField
                id="munki-package-apple-item"
                label="Apple item"
                checked={form.apple_item}
                onChange={(apple_item) => onChange({ ...form, apple_item })}
              />
              <CheckboxField
                id="munki-package-suppress-bundle-relocation"
                label="Suppress bundle relocation"
                checked={form.suppress_bundle_relocation}
                onChange={(suppress_bundle_relocation) => onChange({ ...form, suppress_bundle_relocation })}
              />
            </FieldGroup>
          </FieldSet>
        </FieldGroup>
      </TabsContent>
    </Tabs>
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
        <CheckboxField
          id="munki-package-arch-arm64"
          label="Apple silicon"
          checked={values.includes("arm64")}
          onChange={(checked) => onChange(toggleArray(values, "arm64", checked))}
        />
        <CheckboxField
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
      <div className="space-y-2">
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
      <div className="space-y-3">
        {rows.map((row, index) => (
          <div key={row.rowID} className="grid gap-2 md:grid-cols-[minmax(0,14rem)_minmax(0,1fr)_auto]">
            <Select
              value={row.package_id ? String(row.package_id) : "literal"}
              onValueChange={(value) => {
                const next =
                  value === "literal"
                    ? { ...row, package_id: undefined }
                    : { ...row, package_id: Number(value), name: "" };
                onChange(replaceAt(rows, index, next));
              }}
            >
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value="literal">Literal name</SelectItem>
                  {packageOptions.map((option) => (
                    <SelectItem key={option.id} value={String(option.id)}>
                      {packageLabel(option)}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
            <Input
              value={row.package_id ? packageReferenceLabel(row, packageOptions) : (row.name ?? "")}
              disabled={!!row.package_id}
              onChange={(event) => onChange(replaceAt(rows, index, { ...row, name: event.target.value }))}
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
      <div className="space-y-2">
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
      <div className="space-y-4">
        {rows.map((row, index) => (
          <div key={row.rowID} className="space-y-3 rounded-md border p-3">
            <div className="grid gap-2 md:grid-cols-[10rem_minmax(0,1fr)_auto]">
              <SelectField
                id={`munki-install-item-type-${row.rowID}`}
                label="Type"
                value={row.type}
                options={MUNKI_INSTALL_ITEM_TYPE_OPTIONS}
                onChange={(type) => onChange(replaceAt(rows, index, { ...row, type }))}
              />
              <TextField
                id={`munki-install-item-path-${row.rowID}`}
                label="Path"
                value={row.path}
                onChange={(path) => onChange(replaceAt(rows, index, { ...row, path }))}
              />
              <div className="flex items-end">
                <IconButton label="Remove" onClick={() => onChange(removeAt(rows, index))}>
                  <Trash2 />
                </IconButton>
              </div>
            </div>
            <FieldGroup className="grid gap-3 md:grid-cols-3">
              <TextField
                id={`munki-install-item-bundle-id-${row.rowID}`}
                label="Bundle ID"
                value={row.bundle_identifier ?? ""}
                onChange={(bundle_identifier) => onChange(replaceAt(rows, index, { ...row, bundle_identifier }))}
              />
              <TextField
                id={`munki-install-item-short-version-${row.rowID}`}
                label="Short Version"
                value={row.bundle_short_version ?? ""}
                onChange={(bundle_short_version) => onChange(replaceAt(rows, index, { ...row, bundle_short_version }))}
              />
              <TextField
                id={`munki-install-item-version-${row.rowID}`}
                label="Bundle Version"
                value={row.bundle_version ?? ""}
                onChange={(bundle_version) => onChange(replaceAt(rows, index, { ...row, bundle_version }))}
              />
              <TextField
                id={`munki-install-item-comparison-${row.rowID}`}
                label="Comparison Key"
                value={row.version_comparison_key ?? ""}
                onChange={(version_comparison_key) =>
                  onChange(replaceAt(rows, index, { ...row, version_comparison_key }))
                }
              />
              <TextField
                id={`munki-install-item-md5-${row.rowID}`}
                label="MD5"
                value={row.md5checksum ?? ""}
                onChange={(md5checksum) => onChange(replaceAt(rows, index, { ...row, md5checksum }))}
              />
              <TextField
                id={`munki-install-item-min-os-${row.rowID}`}
                label="Minimum OS"
                value={row.minimum_os_version ?? ""}
                onChange={(minimum_os_version) => onChange(replaceAt(rows, index, { ...row, minimum_os_version }))}
              />
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
      <div className="space-y-2">
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
      <div className="space-y-4">
        {rows.map((row, index) => (
          <div key={row.rowID} className="space-y-3 rounded-md border p-3">
            <div className="grid gap-2 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto]">
              <TextField
                id={`munki-copy-source-${row.rowID}`}
                label="Source"
                value={row.source_item}
                onChange={(source_item) => onChange(replaceAt(rows, index, { ...row, source_item }))}
              />
              <TextField
                id={`munki-copy-destination-${row.rowID}`}
                label="Destination"
                value={row.destination_path}
                onChange={(destination_path) => onChange(replaceAt(rows, index, { ...row, destination_path }))}
              />
              <div className="flex items-end">
                <IconButton label="Remove" onClick={() => onChange(removeAt(rows, index))}>
                  <Trash2 />
                </IconButton>
              </div>
            </div>
            <FieldGroup className="grid gap-3 md:grid-cols-4">
              <TextField
                id={`munki-copy-destination-item-${row.rowID}`}
                label="Destination Item"
                value={row.destination_item ?? ""}
                onChange={(destination_item) => onChange(replaceAt(rows, index, { ...row, destination_item }))}
              />
              <TextField
                id={`munki-copy-user-${row.rowID}`}
                label="User"
                value={row.user ?? ""}
                onChange={(user) => onChange(replaceAt(rows, index, { ...row, user }))}
              />
              <TextField
                id={`munki-copy-group-${row.rowID}`}
                label="Group"
                value={row.group ?? ""}
                onChange={(group) => onChange(replaceAt(rows, index, { ...row, group }))}
              />
              <TextField
                id={`munki-copy-mode-${row.rowID}`}
                label="Mode"
                value={row.mode ?? ""}
                onChange={(mode) => onChange(replaceAt(rows, index, { ...row, mode }))}
              />
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
      <CheckboxField
        id={`${id}-enabled`}
        label="Enabled"
        checked={alert.enabled}
        onChange={(enabled) => onChange({ ...alert, enabled })}
      />
      {alert.enabled ? (
        <FieldGroup className="grid gap-4 md:grid-cols-2">
          <TextField
            id={`${id}-title`}
            label="Title"
            value={alert.title ?? ""}
            onChange={(title) => onChange({ ...alert, title })}
          />
          <TextField
            id={`${id}-ok`}
            label="OK Label"
            value={alert.ok_label ?? ""}
            onChange={(ok_label) => onChange({ ...alert, ok_label })}
          />
          <TextField
            id={`${id}-cancel`}
            label="Cancel Label"
            value={alert.cancel_label ?? ""}
            onChange={(cancel_label) => onChange({ ...alert, cancel_label })}
          />
          <TextAreaField
            id={`${id}-detail`}
            label="Detail"
            value={alert.detail ?? ""}
            onChange={(detail) => onChange({ ...alert, detail })}
          />
        </FieldGroup>
      ) : null}
    </FieldSet>
  );
}

function IconButton({ label, children, onClick }: { label: string; children: React.ReactNode; onClick: () => void }) {
  return (
    <Button type="button" variant="ghost" size="icon-sm" aria-label={label} title={label} onClick={onClick}>
      {children}
    </Button>
  );
}

function packageMutationFromForm(
  form: PackageFormState,
  softwareID: number,
  artifacts: {
    installerArtifactID?: number;
    uninstallerArtifactID?: number;
    iconArtifactID?: number;
  },
): MunkiPackageMutation {
  return {
    software_id: softwareID,
    name: form.name,
    version: form.version,
    display_name: optionalText(form.display_name),
    description: optionalText(form.description),
    category: optionalText(form.category),
    developer: optionalText(form.developer),
    installer_type: form.installer_type,
    uninstall_method: form.uninstall_method,
    custom_uninstall_method: optionalText(form.custom_uninstall_method),
    restart_action: form.restart_action === "None" ? undefined : form.restart_action,
    minimum_munki_version: optionalText(form.minimum_munki_version),
    minimum_os_version: optionalText(form.minimum_os_version),
    maximum_os_version: optionalText(form.maximum_os_version),
    supported_architectures: form.supported_architectures,
    blocking_applications: cleanStringRows(form.blocking_applications),
    requires: cleanPackageReferences(form.requires),
    update_for: cleanPackageReferences(form.update_for),
    eligible: form.eligible,
    unattended_install: form.unattended_install,
    unattended_uninstall: form.unattended_uninstall,
    uninstallable: form.uninstallable,
    on_demand: form.on_demand,
    precache: form.precache,
    autoremove: form.autoremove,
    apple_item: form.apple_item,
    suppress_bundle_relocation: form.suppress_bundle_relocation,
    force_install_after_date: dateTimeLocalToISO(form.force_install_after_date),
    installed_size: numberOrUndefined(form.installed_size),
    payload_identifier: optionalText(form.payload_identifier),
    package_path: optionalText(form.package_path),
    installer_choices_xml: optionalText(form.installer_choices_xml),
    installer_environment: cleanInstallerEnvironment(form.installer_environment),
    installs: cleanInstallItems(form.installs),
    receipts: cleanReceipts(form.receipts),
    items_to_copy: cleanItemsToCopy(form.items_to_copy),
    notes: optionalText(form.notes),
    installcheck_script: optionalText(form.installcheck_script),
    uninstallcheck_script: optionalText(form.uninstallcheck_script),
    preinstall_script: optionalText(form.preinstall_script),
    postinstall_script: optionalText(form.postinstall_script),
    preuninstall_script: optionalText(form.preuninstall_script),
    postuninstall_script: optionalText(form.postuninstall_script),
    uninstall_script: optionalText(form.uninstall_script),
    version_script: optionalText(form.version_script),
    preinstall_alert: cleanAlert(form.preinstall_alert),
    preuninstall_alert: cleanAlert(form.preuninstall_alert),
    installer_artifact_id: artifacts.installerArtifactID,
    uninstaller_artifact_id: artifacts.uninstallerArtifactID,
    icon_artifact_id: artifacts.iconArtifactID,
  };
}

function emptyPackageForm(): PackageFormState {
  return {
    name: "",
    version: "",
    display_name: "",
    description: "",
    category: "",
    developer: "",
    installer_type: "pkg",
    uninstall_method: "none",
    custom_uninstall_method: "",
    restart_action: "None",
    minimum_munki_version: "",
    minimum_os_version: "",
    maximum_os_version: "",
    supported_architectures: [],
    blocking_applications: [],
    requires: [],
    update_for: [],
    eligible: true,
    unattended_install: true,
    unattended_uninstall: true,
    uninstallable: false,
    on_demand: false,
    precache: false,
    autoremove: false,
    apple_item: false,
    suppress_bundle_relocation: false,
    force_install_after_date: "",
    installed_size: "",
    payload_identifier: "",
    package_path: "",
    installer_choices_xml: "",
    installer_environment: [],
    installs: [],
    receipts: [],
    items_to_copy: [],
    notes: "",
    installcheck_script: "",
    uninstallcheck_script: "",
    preinstall_script: "",
    postinstall_script: "",
    preuninstall_script: "",
    postuninstall_script: "",
    uninstall_script: "",
    version_script: "",
    preinstall_alert: emptyAlert(),
    preuninstall_alert: emptyAlert(),
  };
}

function packageFormFromPackage(pkg: MunkiPackage): PackageFormState {
  return {
    name: pkg.name,
    version: pkg.version,
    display_name: pkg.display_name,
    description: pkg.description,
    category: pkg.category,
    developer: pkg.developer,
    installer_type: pkg.installer_type,
    uninstall_method: pkg.uninstall_method,
    custom_uninstall_method: pkg.custom_uninstall_method,
    restart_action: pkg.restart_action ?? "None",
    minimum_munki_version: pkg.minimum_munki_version,
    minimum_os_version: pkg.minimum_os_version,
    maximum_os_version: pkg.maximum_os_version,
    supported_architectures: (pkg.supported_architectures ?? []).filter(isArchitecture),
    blocking_applications: stringRows(pkg.blocking_applications ?? []),
    requires: packageReferenceRows(pkg.requires ?? []),
    update_for: packageReferenceRows(pkg.update_for ?? []),
    eligible: pkg.eligible,
    unattended_install: pkg.unattended_install,
    unattended_uninstall: pkg.unattended_uninstall,
    uninstallable: pkg.uninstallable,
    on_demand: pkg.on_demand,
    precache: pkg.precache,
    autoremove: pkg.autoremove,
    apple_item: pkg.apple_item,
    suppress_bundle_relocation: pkg.suppress_bundle_relocation,
    force_install_after_date: isoToDateTimeLocal(pkg.force_install_after_date),
    installed_size: pkg.installed_size > 0 ? String(pkg.installed_size) : "",
    payload_identifier: pkg.payload_identifier,
    package_path: pkg.package_path,
    installer_choices_xml: pkg.installer_choices_xml,
    installer_environment: installerEnvironmentRows(pkg.installer_environment ?? []),
    installs: installItemRows(pkg.installs ?? []),
    receipts: receiptRows(pkg.receipts ?? []),
    items_to_copy: itemToCopyRows(pkg.items_to_copy ?? []),
    notes: pkg.notes,
    installcheck_script: pkg.installcheck_script,
    uninstallcheck_script: pkg.uninstallcheck_script,
    preinstall_script: pkg.preinstall_script,
    postinstall_script: pkg.postinstall_script,
    preuninstall_script: pkg.preuninstall_script,
    postuninstall_script: pkg.postuninstall_script,
    uninstall_script: pkg.uninstall_script,
    version_script: pkg.version_script,
    preinstall_alert: pkg.preinstall_alert,
    preuninstall_alert: pkg.preuninstall_alert,
  };
}

function emptyAlert(): PackageAlert {
  return { enabled: false };
}

function emptyStringRow(): StringRow {
  return { rowID: rowID(), value: "" };
}

function emptyPackageReferenceRow(): PackageReferenceRow {
  return { rowID: rowID(), name: "" };
}

function emptyInstallerEnvironmentRow(): InstallerEnvironmentRow {
  return { rowID: rowID(), name: "", value: "" };
}

function emptyInstallItemRow(): InstallItemRow {
  return { rowID: rowID(), type: "application", path: "" };
}

function emptyReceiptRow(): ReceiptRow {
  return { rowID: rowID(), package_id: "" };
}

function emptyItemToCopyRow(): ItemToCopyRow {
  return { rowID: rowID(), source_item: "", destination_path: "" };
}

function packageReferenceRows(values: PackageReference[]): PackageReferenceRow[] {
  return values.map((value) => ({ ...value, rowID: rowID() }));
}

function stringRows(values: string[]): StringRow[] {
  return values.map((value) => ({ rowID: rowID(), value }));
}

function installerEnvironmentRows(values: PackageInstallerEnvironmentVariable[]): InstallerEnvironmentRow[] {
  return values.map((value) => ({ ...value, rowID: rowID() }));
}

function installItemRows(values: PackageInstallItem[]): InstallItemRow[] {
  return values.map((value) => ({ ...value, rowID: rowID() }));
}

function receiptRows(values: PackageReceipt[]): ReceiptRow[] {
  return values.map((value) => ({ ...value, rowID: rowID() }));
}

function itemToCopyRows(values: PackageItemToCopy[]): ItemToCopyRow[] {
  return values.map((value) => ({ ...value, rowID: rowID() }));
}

function cleanPackageReferences(rows: PackageReferenceRow[]): PackageReference[] {
  const out: PackageReference[] = [];
  for (const row of rows) {
    if (row.package_id) {
      out.push({ package_id: row.package_id });
      continue;
    }
    const name = row.name?.trim();
    if (name) out.push({ name });
  }
  return out;
}

function cleanInstallerEnvironment(rows: InstallerEnvironmentRow[]): PackageInstallerEnvironmentVariable[] {
  return rows.flatMap((row) => {
    const name = row.name.trim();
    return name ? [{ name, value: row.value }] : [];
  });
}

function cleanInstallItems(rows: InstallItemRow[]): PackageInstallItem[] {
  return rows.flatMap((row) => {
    const path = row.path.trim();
    return path ? [{ ...stripRowID(row), path }] : [];
  });
}

function cleanReceipts(rows: ReceiptRow[]): PackageReceipt[] {
  return rows.flatMap((row) => {
    const packageID = row.package_id.trim();
    return packageID
      ? [{ package_id: packageID, version: optionalText(row.version ?? ""), optional: row.optional }]
      : [];
  });
}

function cleanItemsToCopy(rows: ItemToCopyRow[]): PackageItemToCopy[] {
  return rows.flatMap((row) => {
    const sourceItem = row.source_item.trim();
    const destinationPath = row.destination_path.trim();
    return sourceItem || destinationPath
      ? [{ ...stripRowID(row), source_item: sourceItem, destination_path: destinationPath }]
      : [];
  });
}

function cleanAlert(alert: PackageAlert): PackageAlert {
  if (!alert.enabled) return { enabled: false };
  return {
    enabled: true,
    title: optionalText(alert.title ?? ""),
    detail: optionalText(alert.detail ?? ""),
    ok_label: optionalText(alert.ok_label ?? ""),
    cancel_label: optionalText(alert.cancel_label ?? ""),
  };
}

function cleanStringRows(values: StringRow[]) {
  return values.map((row) => row.value.trim()).filter(Boolean);
}

function numberOrUndefined(value: string) {
  if (value.trim() === "") return undefined;
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : undefined;
}

function dateTimeLocalToISO(value: string) {
  if (value.trim() === "") return undefined;
  const date = new Date(value);
  return Number.isNaN(date.valueOf()) ? undefined : date.toISOString();
}

function isoToDateTimeLocal(value?: string) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.valueOf())) return "";
  const offset = date.getTimezoneOffset() * 60_000;
  return new Date(date.valueOf() - offset).toISOString().slice(0, 16);
}

function stripRowID<T extends { rowID: string }>(row: T): Omit<T, "rowID"> {
  const { rowID: _rowID, ...rest } = row;
  return rest;
}

function replaceAt<T>(rows: T[], index: number, row: T) {
  return rows.map((value, rowIndex) => (rowIndex === index ? row : value));
}

function removeAt<T>(rows: T[], index: number) {
  return rows.filter((_, rowIndex) => rowIndex !== index);
}

function toggleArray<T>(values: T[], value: T, enabled: boolean) {
  if (enabled) return Array.from(new Set([...values, value]));
  return values.filter((item) => item !== value);
}

function packageLabel(pkg: MunkiPackage) {
  return `${pkg.name} ${pkg.version}`;
}

function packageReferenceLabel(row: PackageReferenceRow, packages: MunkiPackage[]) {
  const pkg = packages.find((item) => item.id === row.package_id);
  if (pkg) return packageLabel(pkg);
  if (row.package_name) return `${row.package_name} ${row.package_version ?? ""}`.trim();
  return "";
}

function isArchitecture(value: string): value is Architecture {
  return value === "arm64" || value === "x86_64";
}

function rowID() {
  return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}
