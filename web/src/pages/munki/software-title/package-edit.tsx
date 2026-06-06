import { xml } from "@codemirror/lang-xml";
import type { Extension } from "@codemirror/state";
import { useForm } from "@tanstack/react-form";
import { Link, useNavigate } from "@tanstack/react-router";
import { FileArchive, Loader2, Plus, Trash2 } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { z } from "zod";

import { CodeEditor } from "@/components/editor/code-editor";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { EditableMunkiIcon } from "@/components/munki/editable-munki-icon";
import { MutationError } from "@/components/mutation-error";
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
import { FreeTextCombobox } from "@/components/ui/free-text-combobox";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { useUploadMunkiArtifact } from "@/hooks/munki/artifacts";
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
import { fieldErrors, firstErrorMessage, requiredString } from "@/lib/form-validation";
import { MAX_PAGE_SIZE } from "@/lib/pagination";

import {
  MUNKI_INSTALL_ITEM_TYPE_OPTIONS,
  MUNKI_INSTALLER_TYPE_OPTIONS,
  MUNKI_RESTART_ACTION_OPTIONS,
  MUNKI_UNINSTALL_METHOD_OPTIONS,
  type MunkiInstallerType,
  type MunkiRestartAction,
  type MunkiUninstallMethod,
} from "./shared";
import { optionalText, uniqueOptions, usePackageIDParam, useSoftwareIDParam } from "./utils";

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

const packageIdentitySchema = z.object({
  name: requiredString("Name"),
  version: requiredString("Version"),
});

function validatePackageForm({ value }: { value: PackageFormState }) {
  const result = packageIdentitySchema.safeParse(value);
  if (result.success) return undefined;
  return { fields: fieldErrors(result) };
}

function usePackageEditorForm(initial: PackageFormState, onSubmit: (value: PackageFormState) => Promise<void>) {
  return useForm({
    defaultValues: initial,
    validators: {
      onSubmit: validatePackageForm,
    },
    onSubmit: ({ value }) => {
      if (!packageIdentitySchema.safeParse(value).success) return;
      return onSubmit(value);
    },
  });
}

type PackageEditorForm = ReturnType<typeof usePackageEditorForm>;

type PackageFieldSetter = <K extends keyof PackageFormState>(field: K, value: PackageFormState[K]) => void;

export function MunkiPackageNewPage() {
  const navigate = useNavigate();
  const softwareID = useSoftwareIDParam();
  const software = useMunkiSoftwareTitle(softwareID);
  const create = useCreateMunkiPackage();
  const packageUpload = useUploadMunkiArtifact("package");
  const iconUpload = useUploadMunkiArtifact("icon");
  // Category/developer suggestions are loose helper text; MAX_PAGE_SIZE is enough for this non-managed vocabulary.
  const titles = useMunkiSoftwareTitles({ page_size: MAX_PAGE_SIZE, sort: "name.asc" });
  const packages = useMunkiPackages({ page_size: MAX_PAGE_SIZE, sort: "name.asc" });
  const [installerFile, setInstallerFile] = useState<File | null>(null);
  const [uninstallerFile, setUninstallerFile] = useState<File | null>(null);
  const [iconFile, setIconFile] = useState<File | null>(null);
  const form = usePackageEditorForm(emptyPackageForm(), async (value) => {
    if (softwareID === null) return;
    const installerArtifact = installerFile ? await packageUpload.upload(installerFile) : null;
    const uninstallerArtifact = uninstallerFile ? await packageUpload.upload(uninstallerFile) : null;
    const iconArtifact = iconFile ? await iconUpload.upload(iconFile) : null;
    await create.mutateAsync(
      packageMutationFromForm(value, softwareID, {
        installerArtifactID: installerArtifact?.id,
        uninstallerArtifactID: uninstallerArtifact?.id,
        iconArtifactID: iconArtifact?.id,
      }),
    );
    void navigate({ to: "/munki/software-titles/$softwareId", params: { softwareId: String(softwareID) } });
  });
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
    form.setFieldValue("name", (current) => current || software.data.name, { dontUpdateMeta: true });
    form.setFieldValue("display_name", (current) => current || software.data.display_name, { dontUpdateMeta: true });
    form.setFieldValue("description", (current) => current || software.data.description, { dontUpdateMeta: true });
    form.setFieldValue("category", (current) => current || software.data.category, { dontUpdateMeta: true });
    form.setFieldValue("developer", (current) => current || software.data.developer, { dontUpdateMeta: true });
  }, [form, software.data]);

  return (
    <PageShell asChild>
      <form
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit();
        }}
      >
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
            packageUpload.error?.message ??
            iconUpload.error?.message ??
            software.error?.message
          }
        />
        <PackageEditorTabs
          form={form}
          categoryOptions={categoryOptions}
          developerOptions={developerOptions}
          packageOptions={packages.data?.items ?? []}
          installerFile={installerFile}
          uninstallerFile={uninstallerFile}
          installerArtifactLocation=""
          uninstallerArtifactLocation=""
          onInstallerFileChange={setInstallerFile}
          onUninstallerFileChange={setUninstallerFile}
        />
        <PackageFormActions
          pending={create.isPending || packageUpload.isUploading || iconUpload.isUploading}
          softwareID={softwareID}
        />
      </form>
    </PageShell>
  );
}

export function MunkiPackageEditPage() {
  const softwareID = useSoftwareIDParam();
  const packageID = usePackageIDParam();
  const software = useMunkiSoftwareTitle(softwareID);
  const pkg = useMunkiPackage(packageID);

  if (softwareID === null || packageID === null) {
    return (
      <PageShell>
        <MutationError title="Failed to Load Package" message="Package route is invalid." />
      </PageShell>
    );
  }

  if (pkg.error) {
    return (
      <PageShell>
        <MutationError title="Failed to Load Package" message={pkg.error.message} />
      </PageShell>
    );
  }

  if (!pkg.data) {
    return (
      <PageShell className="text-muted-foreground flex-row items-center gap-2 text-sm">
        <Loader2 className="animate-spin" /> Loading Package...
      </PageShell>
    );
  }

  return (
    <MunkiPackageEditForm
      key={`${pkg.data.id}:${pkg.data.updated_at}`}
      softwareID={softwareID}
      packageID={packageID}
      pkg={pkg.data}
      softwareIconURL={software.data?.icon_url}
      softwareError={software.error?.message}
    />
  );
}

function MunkiPackageEditForm({
  softwareID,
  packageID,
  pkg,
  softwareIconURL,
  softwareError,
}: {
  softwareID: number;
  packageID: number;
  pkg: MunkiPackage;
  softwareIconURL?: string;
  softwareError?: string;
}) {
  const navigate = useNavigate();
  const update = useUpdateMunkiPackage();
  const packageUpload = useUploadMunkiArtifact("package");
  const iconUpload = useUploadMunkiArtifact("icon");
  // Category/developer suggestions are loose helper text; MAX_PAGE_SIZE is enough for this non-managed vocabulary.
  const titles = useMunkiSoftwareTitles({ page_size: MAX_PAGE_SIZE, sort: "name.asc" });
  const packages = useMunkiPackages({ page_size: MAX_PAGE_SIZE, sort: "name.asc" });
  const [installerFile, setInstallerFile] = useState<File | null>(null);
  const [uninstallerFile, setUninstallerFile] = useState<File | null>(null);
  const [iconFile, setIconFile] = useState<File | null>(null);
  const [iconCleared, setIconCleared] = useState(false);
  const initial = useMemo(() => packageFormFromPackage(pkg), [pkg]);
  const form = usePackageEditorForm(initial, async (value) => {
    const installerArtifact = installerFile ? await packageUpload.upload(installerFile) : null;
    const uninstallerArtifact = uninstallerFile ? await packageUpload.upload(uninstallerFile) : null;
    const iconArtifact = iconFile ? await iconUpload.upload(iconFile) : null;
    const body = packageMutationFromForm(value, softwareID, {
      installerArtifactID: installerArtifact?.id ?? pkg.installer_artifact_id,
      uninstallerArtifactID: uninstallerArtifact?.id ?? pkg.uninstaller_artifact_id,
      iconArtifactID: iconArtifact?.id ?? (iconCleared ? undefined : pkg.icon_artifact_id),
    });
    await update.mutateAsync({ id: packageID, body });
    void navigate({ to: "/munki/software-titles/$softwareId", params: { softwareId: String(softwareID) } });
  });
  const categoryOptions = useMemo(
    () => uniqueOptions((titles.data?.items ?? []).map((item) => item.category)),
    [titles.data?.items],
  );
  const developerOptions = useMemo(
    () => uniqueOptions((titles.data?.items ?? []).map((item) => item.developer)),
    [titles.data?.items],
  );

  const packageIconURL = iconCleared || !pkg.icon_artifact_id ? undefined : pkg.icon_url;
  const packageIconClearable = !!iconFile || (!iconCleared && !!pkg.icon_artifact_id);

  return (
    <PageShell asChild>
      <form
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit();
        }}
      >
        <PageHeader
          title="Edit Package"
          leading={
            <EditableMunkiIcon
              title="package icon"
              iconUrl={packageIconURL}
              fallbackIconUrl={softwareIconURL}
              file={iconFile}
              clearable={packageIconClearable}
              onFileChange={(file) => {
                setIconFile(file);
                setIconCleared(false);
              }}
              onClear={() => {
                setIconFile(null);
                setIconCleared(!!pkg.icon_artifact_id);
              }}
            />
          }
        />
        <MutationError
          title="Failed to Update Package"
          message={update.error?.message ?? packageUpload.error?.message ?? iconUpload.error?.message ?? softwareError}
        />
        <PackageEditorTabs
          form={form}
          categoryOptions={categoryOptions}
          developerOptions={developerOptions}
          packageOptions={(packages.data?.items ?? []).filter((item) => item.id !== packageID)}
          installerFile={installerFile}
          uninstallerFile={uninstallerFile}
          installerArtifactLocation={pkg.installer_artifact_location ?? ""}
          uninstallerArtifactLocation={pkg.uninstaller_artifact_location ?? ""}
          onInstallerFileChange={setInstallerFile}
          onUninstallerFileChange={setUninstallerFile}
        />
        <PackageFormActions
          pending={update.isPending || packageUpload.isUploading || iconUpload.isUploading}
          softwareID={softwareID}
        />
      </form>
    </PageShell>
  );
}

function PackageEditorTabs({
  form,
  categoryOptions,
  developerOptions,
  packageOptions,
  installerFile,
  uninstallerFile,
  installerArtifactLocation,
  uninstallerArtifactLocation,
  onInstallerFileChange,
  onUninstallerFileChange,
}: {
  form: PackageEditorForm;
  categoryOptions: string[];
  developerOptions: string[];
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
                <form.Field
                  name="name"
                  validators={{ onSubmit: requiredString("Name") }}
                  children={(field) => {
                    const invalid = field.state.meta.isTouched && !field.state.meta.isValid;
                    const error = firstErrorMessage(field.state.meta.errors);
                    return (
                      <Field data-invalid={invalid}>
                        <FieldLabel htmlFor="munki-package-name" required>
                          Name
                        </FieldLabel>
                        <Input
                          id="munki-package-name"
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
                  <Field>
                    <FieldLabel htmlFor="munki-package-display-name">Display Name</FieldLabel>
                    <Input
                      id="munki-package-display-name"
                      value={values.display_name}
                      onChange={(event) => setField("display_name", event.target.value)}
                    />
                  </Field>
                </FieldGroup>
                <Field>
                  <FieldLabel htmlFor="munki-package-description">Description</FieldLabel>
                  <Textarea
                    id="munki-package-description"
                    value={values.description}
                    onChange={(event) => setField("description", event.target.value)}
                  />
                </Field>
                <FieldGroup className="grid gap-4 md:grid-cols-3">
                  <Field>
                    <FieldLabel htmlFor="munki-package-category">Category</FieldLabel>
                    <FreeTextCombobox
                      id="munki-package-category"
                      value={values.category}
                      options={categoryOptions}
                      onChange={(category) => setField("category", category)}
                    />
                  </Field>
                  <Field>
                    <FieldLabel htmlFor="munki-package-developer">Developer</FieldLabel>
                    <FreeTextCombobox
                      id="munki-package-developer"
                      value={values.developer}
                      options={developerOptions}
                      onChange={(developer) => setField("developer", developer)}
                    />
                  </Field>
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
                <FieldSet>
                  <FieldLegend>Artifacts</FieldLegend>
                  <FieldGroup>
                    <PackageFileField
                      id="munki-package-installer-file"
                      label="Installer"
                      description={installerArtifactLocation || "No installer artifact selected."}
                      icon={<FileArchive className="size-4" />}
                      file={installerFile}
                      onChange={onInstallerFileChange}
                    />
                    <PackageFileField
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
                    <Field>
                      <FieldLabel htmlFor="munki-package-payload-identifier">Payload Identifier</FieldLabel>
                      <Input
                        id="munki-package-payload-identifier"
                        value={values.payload_identifier}
                        onChange={(event) => setField("payload_identifier", event.target.value)}
                      />
                    </Field>
                    <Field>
                      <FieldLabel htmlFor="munki-package-package-path">Package Path</FieldLabel>
                      <Input
                        id="munki-package-package-path"
                        value={values.package_path}
                        onChange={(event) => setField("package_path", event.target.value)}
                      />
                    </Field>
                  </FieldGroup>
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
                  <XMLField
                    value={values.installer_choices_xml}
                    onChange={(installer_choices_xml) => setField("installer_choices_xml", installer_choices_xml)}
                  />
                  <InstallerEnvironmentEditor
                    rows={values.installer_environment}
                    onChange={(installer_environment) => setField("installer_environment", installer_environment)}
                  />
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
                </FieldSet>
                <StringArrayEditor
                  legend="Blocking Applications"
                  addLabel="Application"
                  rows={values.blocking_applications}
                  onChange={(blocking_applications) => setField("blocking_applications", blocking_applications)}
                />
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
                <ItemsToCopyEditor
                  rows={values.items_to_copy}
                  onChange={(items_to_copy) => setField("items_to_copy", items_to_copy)}
                />
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
                    label="Available for assignment"
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
                    <CheckboxControl
                      id="munki-package-uninstallable"
                      label="Uninstallable"
                      checked={values.uninstallable}
                      onChange={(uninstallable) => setField("uninstallable", uninstallable)}
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
                  {values.uninstall_method === "custom" ? (
                    <Field>
                      <FieldLabel htmlFor="munki-package-custom-uninstall-method">Custom Uninstall Method</FieldLabel>
                      <Input
                        id="munki-package-custom-uninstall-method"
                        value={values.custom_uninstall_method}
                        onChange={(event) => setField("custom_uninstall_method", event.target.value)}
                      />
                    </Field>
                  ) : null}
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

function PackageFormActions({ pending, softwareID }: { pending: boolean; softwareID: number | null }) {
  return (
    <div className="flex items-center gap-2">
      <Button type="submit" size="sm" disabled={pending}>
        {pending ? <Loader2 data-icon="inline-start" className="animate-spin" /> : null}
        Save
      </Button>
      <Button asChild type="button" variant="outline" size="sm">
        <Link to="/munki/software-titles/$softwareId" params={{ softwareId: String(softwareID ?? "") }}>
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
  icon: React.ReactNode;
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
