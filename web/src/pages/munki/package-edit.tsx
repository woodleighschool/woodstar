import { useNavigate } from "@tanstack/react-router";
import { FileArchive } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { z } from "zod";

import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { EditableMunkiIcon } from "@/components/munki/editable-munki-icon";
import { Field, FieldDescription, FieldGroup, FieldLabel, FieldLegend, FieldSet } from "@/components/ui/field";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { useCreateMunkiArtifact, useCreateMunkiArtifactUpload } from "@/hooks/munki/artifacts";
import {
  useCreateMunkiPackage,
  useMunkiPackage,
  useUpdateMunkiPackage,
  type MunkiPackageMutation,
} from "@/hooks/munki/packages";
import { useMunkiSoftwareTitle, useMunkiSoftwareTitles } from "@/hooks/munki/software-titles";
import { fieldErrors, requiredString } from "@/lib/form-validation";
import { MAX_PAGE_SIZE } from "@/lib/pagination";

import {
  CheckboxField,
  DatalistTextField,
  FileField,
  FormActions,
  MutationError,
  SelectField,
  StringListField,
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

type InstallerType = NonNullable<MunkiPackageMutation["installer_type"]>;
type RestartAction = NonNullable<MunkiPackageMutation["restart_action"]>;

const installerTypeOptions: { value: InstallerType; label: string; description: string }[] = [
  { value: "pkg", label: "Package", description: "Ordinary pkg or mpkg item; omitted from rendered installer_type." },
  { value: "nopkg", label: "No package", description: "Metadata-only item with installcheck logic." },
  { value: "profile", label: "Profile", description: "Install a configuration profile." },
  {
    value: "apple_update_metadata",
    label: "Apple update metadata",
    description: "Apple software update metadata item.",
  },
];

const restartActionOptions: { value: RestartAction; label: string }[] = [
  { value: "None", label: "None" },
  { value: "RequireLogout", label: "Require logout" },
  { value: "RecommendRestart", label: "Recommend restart" },
  { value: "RequireRestart", label: "Require restart" },
  { value: "RequireShutdown", label: "Require shutdown" },
];

const packageSchema = z.object({
  name: requiredString("Name"),
  version: requiredString("Version"),
  display_name: z.string().trim(),
  description: z.string().trim(),
  category: z.string().trim(),
  developer: z.string().trim(),
  installer_type: z.enum(["pkg", "nopkg", "profile", "apple_update_metadata"]),
  uninstall_method: z.string().trim(),
  restart_action: z.enum(["None", "RequireLogout", "RecommendRestart", "RequireRestart", "RequireShutdown"]),
  minimum_munki_version: z.string().trim(),
  minimum_os_version: z.string().trim(),
  maximum_os_version: z.string().trim(),
  supported_architectures: z.array(z.enum(["arm64", "x86_64"])),
  blocking_applications: z.string().transform(parseStringList),
  requires: z.string().transform(parseStringList),
  update_for: z.string().transform(parseStringList),
  eligible: z.boolean(),
  unattended_install: z.boolean(),
  unattended_uninstall: z.boolean(),
  uninstallable: z.boolean(),
  on_demand: z.boolean(),
  precache: z.boolean(),
});

interface PackageFormState {
  name: string;
  version: string;
  display_name: string;
  description: string;
  category: string;
  developer: string;
  installer_type: InstallerType;
  uninstall_method: string;
  restart_action: RestartAction;
  minimum_munki_version: string;
  minimum_os_version: string;
  maximum_os_version: string;
  supported_architectures: Array<"arm64" | "x86_64">;
  blocking_applications: string;
  requires: string;
  update_for: string;
  eligible: boolean;
  unattended_install: boolean;
  unattended_uninstall: boolean;
  uninstallable: boolean;
  on_demand: boolean;
  precache: boolean;
}

export function MunkiPackageNewPage() {
  const navigate = useNavigate();
  const softwareId = useSoftwareIDParam();
  const software = useMunkiSoftwareTitle(softwareId);
  const create = useCreateMunkiPackage();
  const createUpload = useCreateMunkiArtifactUpload();
  const createArtifact = useCreateMunkiArtifact();
  const titles = useMunkiSoftwareTitles({ page_size: MAX_PAGE_SIZE, sort: "name.asc" });
  const categoryOptions = useMemo(
    () => uniqueOptions((titles.data?.items ?? []).map((item) => item.category)),
    [titles.data?.items],
  );
  const developerOptions = useMemo(
    () => uniqueOptions((titles.data?.items ?? []).map((item) => item.developer)),
    [titles.data?.items],
  );
  const [installerFile, setInstallerFile] = useState<File | null>(null);
  const [iconFile, setIconFile] = useState<File | null>(null);
  const [form, setForm] = useState<PackageFormState>({
    name: "",
    version: "",
    display_name: "",
    description: "",
    category: "",
    developer: "",
    installer_type: "pkg",
    uninstall_method: "",
    restart_action: "None",
    minimum_munki_version: "",
    minimum_os_version: "",
    maximum_os_version: "",
    supported_architectures: [],
    blocking_applications: "",
    requires: "",
    update_for: "",
    eligible: true,
    unattended_install: true,
    unattended_uninstall: true,
    uninstallable: false,
    on_demand: false,
    precache: false,
  });
  const [showErrors, setShowErrors] = useState(false);
  const parsed = useMemo(() => packageSchema.safeParse(form), [form]);
  const errors = useMemo(() => fieldErrors(parsed), [parsed]);

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
    if (!next.success || softwareId === null) {
      setShowErrors(true);
      return;
    }
    const installerArtifact = installerFile
      ? await uploadSelectedArtifact(installerFile, "package", createUpload.mutateAsync, createArtifact.mutateAsync)
      : null;
    const iconArtifact = iconFile
      ? await uploadSelectedArtifact(iconFile, "icon", createUpload.mutateAsync, createArtifact.mutateAsync)
      : null;
    const body: MunkiPackageMutation = {
      software_id: softwareId,
      name: next.data.name,
      version: next.data.version,
      display_name: optionalText(next.data.display_name),
      description: optionalText(next.data.description),
      category: optionalText(next.data.category),
      developer: optionalText(next.data.developer),
      eligible: next.data.eligible,
      installer_type: next.data.installer_type,
      unattended_install: next.data.unattended_install,
      unattended_uninstall: next.data.unattended_uninstall,
      uninstallable: next.data.uninstallable,
      uninstall_method: optionalText(next.data.uninstall_method),
      restart_action: next.data.restart_action === "None" ? undefined : next.data.restart_action,
      minimum_munki_version: optionalText(next.data.minimum_munki_version),
      minimum_os_version: optionalText(next.data.minimum_os_version),
      maximum_os_version: optionalText(next.data.maximum_os_version),
      supported_architectures:
        next.data.supported_architectures.length > 0 ? next.data.supported_architectures : undefined,
      blocking_applications: next.data.blocking_applications,
      requires: next.data.requires,
      update_for: next.data.update_for,
      on_demand: next.data.on_demand,
      precache: next.data.precache,
      installer_artifact_id: installerArtifact?.id,
      icon_artifact_id: iconArtifact?.id,
    };
    await create.mutateAsync(body);
    void navigate({ to: "/munki/software-titles/$softwareId", params: { softwareId: String(softwareId) } });
  }

  return (
    <PageShell asChild>
      <form noValidate onSubmit={(event) => runSubmit(event, submit)}>
        <PageHeader
          title="New Package"
          description="Add a typed pkginfo package with an optional installer file."
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
        <Tabs defaultValue="basic" className="max-w-3xl">
          <TabsList>
            <TabsTrigger value="basic">Basic</TabsTrigger>
            <TabsTrigger value="contents">Contents</TabsTrigger>
            <TabsTrigger value="requirements">Requirements</TabsTrigger>
            <TabsTrigger value="installation">Installation</TabsTrigger>
            <TabsTrigger value="advanced">Advanced</TabsTrigger>
          </TabsList>
          <TabsContent value="basic">
            <FieldGroup>
              <TextField
                id="munki-package-name"
                label="Name"
                required
                description="Stable Munki item name used in manifests. Keep it consistent across versions."
                value={form.name}
                error={showErrors ? errors.name : undefined}
                placeholder={software.data?.name}
                onChange={(name) => setForm({ ...form, name })}
              />
              <FieldGroup className="grid gap-4 md:grid-cols-2">
                <TextField
                  id="munki-package-version"
                  label="Version"
                  required
                  description="Rendered into pkginfo and shown when choosing an assignment package."
                  value={form.version}
                  error={showErrors ? errors.version : undefined}
                  onChange={(version) => setForm({ ...form, version })}
                />
                <TextField
                  id="munki-package-display-name"
                  label="Display Name"
                  value={form.display_name}
                  placeholder={software.data?.display_name}
                  onChange={(display_name) => setForm({ ...form, display_name })}
                />
              </FieldGroup>
              <TextAreaField
                id="munki-package-description"
                label="Description"
                value={form.description}
                placeholder={software.data?.description}
                onChange={(description) => setForm({ ...form, description })}
              />
              <FieldGroup className="grid gap-4 md:grid-cols-3">
                <DatalistTextField
                  id="munki-package-category"
                  label="Category"
                  value={form.category}
                  placeholder={software.data?.category}
                  options={categoryOptions}
                  onChange={(category) => setForm({ ...form, category })}
                />
                <DatalistTextField
                  id="munki-package-developer"
                  label="Developer"
                  value={form.developer}
                  placeholder={software.data?.developer}
                  options={developerOptions}
                  onChange={(developer) => setForm({ ...form, developer })}
                />
                <SelectField
                  id="munki-package-installer-type"
                  label="Installer Type"
                  description="Ordinary packages use Package. Woodstar omits that value from rendered pkginfo because Munki does not use installer_type for normal pkg installs."
                  value={form.installer_type}
                  options={installerTypeOptions}
                  onChange={(installer_type) => setForm({ ...form, installer_type })}
                />
              </FieldGroup>
            </FieldGroup>
          </TabsContent>
          <TabsContent value="contents">
            <FieldSet>
              <FieldLegend>Artifacts</FieldLegend>
              <FieldDescription>
                Files upload to Munki storage first. Woodstar stores the artifact reference and renders the stable Munki
                URL.
              </FieldDescription>
              <FieldGroup>
                <FileField
                  id="munki-package-installer-file"
                  label="Installer"
                  description="Optional package, disk image, profile, or metadata payload used by this pkginfo."
                  icon={<FileArchive className="size-4" />}
                  file={installerFile}
                  onChange={setInstallerFile}
                />
              </FieldGroup>
            </FieldSet>
          </TabsContent>
          <TabsContent value="requirements">
            <PackageRequirementsFields form={form} onChange={setForm} />
          </TabsContent>
          <TabsContent value="installation">
            <FieldSet>
              <FieldLegend>Package Behavior</FieldLegend>
              <FieldDescription>
                These values are rendered into pkginfo. They do not inspect the installer bytes.
              </FieldDescription>
              <CheckboxField
                id="munki-package-eligible"
                label="Available for assignment"
                description="Ineligible packages stay stored but are skipped when manifests and catalogs are rendered."
                checked={form.eligible}
                onChange={(eligible) => setForm({ ...form, eligible })}
              />
              <FieldGroup className="grid gap-4 md:grid-cols-3">
                <CheckboxField
                  id="munki-package-unattended-install"
                  label="Unattended install"
                  description="Allows Munki to install this item without MSC interaction."
                  checked={form.unattended_install}
                  onChange={(unattended_install) => setForm({ ...form, unattended_install })}
                />
                <CheckboxField
                  id="munki-package-unattended-uninstall"
                  label="Unattended uninstall"
                  description="Allows Munki to remove this item without MSC interaction."
                  checked={form.unattended_uninstall}
                  onChange={(unattended_uninstall) => setForm({ ...form, unattended_uninstall })}
                />
                <CheckboxField
                  id="munki-package-uninstallable"
                  label="Uninstallable"
                  description="Marks the item as removable when Munki has a valid uninstall method."
                  checked={form.uninstallable}
                  onChange={(uninstallable) => setForm({ ...form, uninstallable })}
                />
              </FieldGroup>
              <FieldGroup className="grid gap-4 md:grid-cols-2">
                <TextField
                  id="munki-package-uninstall-method"
                  label="Uninstall Method"
                  description="Munki uninstall_method value, when the item supports removal."
                  value={form.uninstall_method}
                  onChange={(uninstall_method) => setForm({ ...form, uninstall_method })}
                />
                <SelectField
                  id="munki-package-restart-action"
                  label="Restart Action"
                  value={form.restart_action}
                  options={restartActionOptions}
                  onChange={(restart_action) => setForm({ ...form, restart_action })}
                />
              </FieldGroup>
              <FieldGroup className="grid gap-4 md:grid-cols-2">
                <CheckboxField
                  id="munki-package-on-demand"
                  label="On demand"
                  description="Marks the item as available only when explicitly requested by Munki."
                  checked={form.on_demand}
                  onChange={(on_demand) => setForm({ ...form, on_demand })}
                />
                <CheckboxField
                  id="munki-package-precache"
                  label="Precache"
                  description="Allows Munki to cache the installer before it is needed."
                  checked={form.precache}
                  onChange={(precache) => setForm({ ...form, precache })}
                />
              </FieldGroup>
            </FieldSet>
          </TabsContent>
          <TabsContent value="advanced">
            <FieldSet>
              <FieldLegend>Rendered Pkginfo</FieldLegend>
              <FieldDescription>
                Woodstar renders pkginfo from typed fields after the package is saved. Scripts and alerts will get typed
                fields in a later slice.
              </FieldDescription>
            </FieldSet>
          </TabsContent>
          <FormActions
            pending={create.isPending || createUpload.isPending || createArtifact.isPending}
            cancelTo="/munki/software-titles/$softwareId"
            cancelParams={{ softwareId: String(softwareId ?? "") }}
          />
        </Tabs>
      </form>
    </PageShell>
  );
}

export function MunkiPackageEditPage() {
  const navigate = useNavigate();
  const softwareId = useSoftwareIDParam();
  const packageId = usePackageIDParam();
  const software = useMunkiSoftwareTitle(softwareId);
  const pkg = useMunkiPackage(packageId);
  const update = useUpdateMunkiPackage();
  const createUpload = useCreateMunkiArtifactUpload();
  const createArtifact = useCreateMunkiArtifact();
  const titles = useMunkiSoftwareTitles({ page_size: MAX_PAGE_SIZE, sort: "name.asc" });
  const categoryOptions = useMemo(
    () => uniqueOptions((titles.data?.items ?? []).map((item) => item.category)),
    [titles.data?.items],
  );
  const developerOptions = useMemo(
    () => uniqueOptions((titles.data?.items ?? []).map((item) => item.developer)),
    [titles.data?.items],
  );
  const [installerFile, setInstallerFile] = useState<File | null>(null);
  const [iconFile, setIconFile] = useState<File | null>(null);
  const [iconCleared, setIconCleared] = useState(false);
  const [form, setForm] = useState<PackageFormState>({
    name: "",
    version: "",
    display_name: "",
    description: "",
    category: "",
    developer: "",
    installer_type: "pkg",
    uninstall_method: "",
    restart_action: "None",
    minimum_munki_version: "",
    minimum_os_version: "",
    maximum_os_version: "",
    supported_architectures: [],
    blocking_applications: "",
    requires: "",
    update_for: "",
    eligible: true,
    unattended_install: true,
    unattended_uninstall: true,
    uninstallable: false,
    on_demand: false,
    precache: false,
  });
  const [showErrors, setShowErrors] = useState(false);
  const parsed = useMemo(() => packageSchema.safeParse(form), [form]);
  const errors = useMemo(() => fieldErrors(parsed), [parsed]);

  useEffect(() => {
    if (!pkg.data) return;
    setForm({
      name: pkg.data.name,
      version: pkg.data.version,
      display_name: pkg.data.display_name,
      description: pkg.data.description,
      category: pkg.data.category,
      developer: pkg.data.developer,
      installer_type: pkg.data.installer_type,
      uninstall_method: pkg.data.uninstall_method,
      restart_action: pkg.data.restart_action ?? "None",
      minimum_munki_version: pkg.data.minimum_munki_version,
      minimum_os_version: pkg.data.minimum_os_version,
      maximum_os_version: pkg.data.maximum_os_version,
      supported_architectures: (pkg.data.supported_architectures ?? []).filter(isSupportedArch),
      blocking_applications: formatStringList(pkg.data.blocking_applications ?? []),
      requires: formatStringList(pkg.data.requires ?? []),
      update_for: formatStringList(pkg.data.update_for ?? []),
      eligible: pkg.data.eligible,
      unattended_install: pkg.data.unattended_install,
      unattended_uninstall: pkg.data.unattended_uninstall,
      uninstallable: pkg.data.uninstallable,
      on_demand: pkg.data.on_demand,
      precache: pkg.data.precache,
    });
    setIconFile(null);
    setIconCleared(false);
  }, [pkg.data]);

  async function submit() {
    const next = packageSchema.safeParse(form);
    if (!next.success || softwareId === null || packageId === null) {
      setShowErrors(true);
      return;
    }
    const installerArtifact = installerFile
      ? await uploadSelectedArtifact(installerFile, "package", createUpload.mutateAsync, createArtifact.mutateAsync)
      : null;
    const iconArtifact = iconFile
      ? await uploadSelectedArtifact(iconFile, "icon", createUpload.mutateAsync, createArtifact.mutateAsync)
      : null;
    const body: MunkiPackageMutation = {
      software_id: softwareId,
      name: next.data.name,
      version: next.data.version,
      display_name: optionalText(next.data.display_name),
      description: optionalText(next.data.description),
      category: optionalText(next.data.category),
      developer: optionalText(next.data.developer),
      eligible: next.data.eligible,
      installer_type: next.data.installer_type,
      unattended_install: next.data.unattended_install,
      unattended_uninstall: next.data.unattended_uninstall,
      uninstallable: next.data.uninstallable,
      uninstall_method: optionalText(next.data.uninstall_method),
      restart_action: next.data.restart_action === "None" ? undefined : next.data.restart_action,
      minimum_munki_version: optionalText(next.data.minimum_munki_version),
      minimum_os_version: optionalText(next.data.minimum_os_version),
      maximum_os_version: optionalText(next.data.maximum_os_version),
      supported_architectures:
        next.data.supported_architectures.length > 0 ? next.data.supported_architectures : undefined,
      blocking_applications: next.data.blocking_applications,
      requires: next.data.requires,
      update_for: next.data.update_for,
      on_demand: next.data.on_demand,
      precache: next.data.precache,
      installer_artifact_id: installerArtifact?.id ?? pkg.data?.installer_artifact_id,
      icon_artifact_id: iconArtifact?.id ?? (iconCleared ? undefined : pkg.data?.icon_artifact_id),
    };
    await update.mutateAsync({ id: packageId, body });
    void navigate({ to: "/munki/software-titles/$softwareId", params: { softwareId: String(softwareId) } });
  }

  const packageIconURL = iconCleared || !pkg.data?.icon_artifact_id ? undefined : pkg.data.icon_url;
  const packageIconClearable = !!iconFile || (!iconCleared && !!pkg.data?.icon_artifact_id);

  return (
    <PageShell asChild>
      <form noValidate onSubmit={(event) => runSubmit(event, submit)}>
        <PageHeader
          title="Edit Package"
          description="Edit the typed pkginfo fields Woodstar renders into Munki catalogs."
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
        <Tabs defaultValue="basic" className="max-w-3xl">
          <TabsList>
            <TabsTrigger value="basic">Basic</TabsTrigger>
            <TabsTrigger value="contents">Contents</TabsTrigger>
            <TabsTrigger value="requirements">Requirements</TabsTrigger>
            <TabsTrigger value="installation">Installation</TabsTrigger>
            <TabsTrigger value="advanced">Advanced</TabsTrigger>
          </TabsList>
          <TabsContent value="basic">
            <FieldGroup>
              <TextField
                id="munki-package-name"
                label="Name"
                required
                description="Stable Munki item name. Keep it consistent across versions when this should update in place."
                value={form.name}
                error={showErrors ? errors.name : undefined}
                placeholder={software.data?.name}
                onChange={(name) => setForm({ ...form, name })}
              />
              <FieldGroup className="grid gap-4 md:grid-cols-2">
                <TextField
                  id="munki-package-version"
                  label="Version"
                  required
                  value={form.version}
                  error={showErrors ? errors.version : undefined}
                  onChange={(version) => setForm({ ...form, version })}
                />
                <TextField
                  id="munki-package-display-name"
                  label="Display Name"
                  value={form.display_name}
                  placeholder={software.data?.display_name}
                  onChange={(display_name) => setForm({ ...form, display_name })}
                />
              </FieldGroup>
              <TextAreaField
                id="munki-package-description"
                label="Description"
                value={form.description}
                placeholder={software.data?.description}
                onChange={(description) => setForm({ ...form, description })}
              />
              <FieldGroup className="grid gap-4 md:grid-cols-3">
                <DatalistTextField
                  id="munki-package-category"
                  label="Category"
                  value={form.category}
                  placeholder={software.data?.category}
                  options={categoryOptions}
                  onChange={(category) => setForm({ ...form, category })}
                />
                <DatalistTextField
                  id="munki-package-developer"
                  label="Developer"
                  value={form.developer}
                  placeholder={software.data?.developer}
                  options={developerOptions}
                  onChange={(developer) => setForm({ ...form, developer })}
                />
                <SelectField
                  id="munki-package-installer-type"
                  label="Installer Type"
                  value={form.installer_type}
                  options={installerTypeOptions}
                  onChange={(installer_type) => setForm({ ...form, installer_type })}
                />
              </FieldGroup>
            </FieldGroup>
          </TabsContent>
          <TabsContent value="contents">
            <FieldSet>
              <FieldLegend>Artifacts</FieldLegend>
              <FieldDescription>
                Replacing a file uploads a new artifact and keeps the existing artifact if no replacement is selected.
              </FieldDescription>
              <FieldGroup>
                <FileField
                  id="munki-package-installer-file"
                  label="Installer"
                  description={pkg.data?.installer_artifact_location ?? "No installer artifact selected."}
                  icon={<FileArchive className="size-4" />}
                  file={installerFile}
                  onChange={setInstallerFile}
                />
              </FieldGroup>
            </FieldSet>
          </TabsContent>
          <TabsContent value="requirements">
            <PackageRequirementsFields form={form} onChange={setForm} />
          </TabsContent>
          <TabsContent value="installation">
            <FieldSet>
              <FieldLegend>Package Behavior</FieldLegend>
              <FieldDescription>These values are rendered into pkginfo.</FieldDescription>
              <CheckboxField
                id="munki-package-eligible"
                label="Available for assignment"
                description="Ineligible packages stay stored but are skipped when manifests and catalogs are rendered."
                checked={form.eligible}
                onChange={(eligible) => setForm({ ...form, eligible })}
              />
              <FieldGroup className="grid gap-4 md:grid-cols-3">
                <CheckboxField
                  id="munki-package-unattended-install"
                  label="Unattended install"
                  checked={form.unattended_install}
                  onChange={(unattended_install) => setForm({ ...form, unattended_install })}
                />
                <CheckboxField
                  id="munki-package-unattended-uninstall"
                  label="Unattended uninstall"
                  checked={form.unattended_uninstall}
                  onChange={(unattended_uninstall) => setForm({ ...form, unattended_uninstall })}
                />
                <CheckboxField
                  id="munki-package-uninstallable"
                  label="Uninstallable"
                  checked={form.uninstallable}
                  onChange={(uninstallable) => setForm({ ...form, uninstallable })}
                />
              </FieldGroup>
              <FieldGroup className="grid gap-4 md:grid-cols-2">
                <TextField
                  id="munki-package-uninstall-method"
                  label="Uninstall Method"
                  value={form.uninstall_method}
                  onChange={(uninstall_method) => setForm({ ...form, uninstall_method })}
                />
                <SelectField
                  id="munki-package-restart-action"
                  label="Restart Action"
                  value={form.restart_action}
                  options={restartActionOptions}
                  onChange={(restart_action) => setForm({ ...form, restart_action })}
                />
              </FieldGroup>
              <FieldGroup className="grid gap-4 md:grid-cols-2">
                <CheckboxField
                  id="munki-package-on-demand"
                  label="On demand"
                  checked={form.on_demand}
                  onChange={(on_demand) => setForm({ ...form, on_demand })}
                />
                <CheckboxField
                  id="munki-package-precache"
                  label="Precache"
                  checked={form.precache}
                  onChange={(precache) => setForm({ ...form, precache })}
                />
              </FieldGroup>
            </FieldSet>
          </TabsContent>
          <TabsContent value="advanced">
            <FieldSet>
              <FieldLegend>Rendered Pkginfo</FieldLegend>
              <FieldDescription>
                This is generated from typed fields. Scripts and alerts stay out of this slice until Woodstar stores
                those Munki fields explicitly.
              </FieldDescription>
              {pkg.data?.pkginfo ? (
                <Textarea
                  value={JSON.stringify(pkg.data.pkginfo, null, 2)}
                  readOnly
                  className="min-h-72 font-mono text-xs"
                />
              ) : null}
            </FieldSet>
          </TabsContent>
          <FormActions
            pending={update.isPending || createUpload.isPending || createArtifact.isPending || pkg.isLoading}
            cancelTo="/munki/software-titles/$softwareId"
            cancelParams={{ softwareId: String(softwareId ?? "") }}
          />
        </Tabs>
      </form>
    </PageShell>
  );
}

function PackageRequirementsFields({
  form,
  onChange,
}: {
  form: PackageFormState;
  onChange: (next: PackageFormState) => void;
}) {
  return (
    <FieldSet>
      <FieldLegend>Requirements</FieldLegend>
      <FieldDescription>
        These map to Munki pkginfo requirement fields. Leave lists empty when the package has no prerequisite, updater,
        or blocking-app rule.
      </FieldDescription>
      <TextField
        id="munki-package-minimum-munki-version"
        label="Minimum Munki Version"
        value={form.minimum_munki_version}
        placeholder="7.0"
        onChange={(minimum_munki_version) => onChange({ ...form, minimum_munki_version })}
      />
      <FieldGroup className="grid gap-4 md:grid-cols-2">
        <TextField
          id="munki-package-minimum-os"
          label="Minimum OS"
          value={form.minimum_os_version}
          placeholder="14.0"
          onChange={(minimum_os_version) => onChange({ ...form, minimum_os_version })}
        />
        <TextField
          id="munki-package-maximum-os"
          label="Maximum OS"
          value={form.maximum_os_version}
          placeholder="15.99"
          onChange={(maximum_os_version) => onChange({ ...form, maximum_os_version })}
        />
      </FieldGroup>
      <Field>
        <FieldLabel>Supported Architectures</FieldLabel>
        <div className="grid gap-3 md:grid-cols-2">
          <CheckboxField
            id="munki-package-arch-arm64"
            label="Apple silicon"
            checked={form.supported_architectures.includes("arm64")}
            onChange={(checked) => onChange({ ...form, supported_architectures: toggleArch(form, "arm64", checked) })}
          />
          <CheckboxField
            id="munki-package-arch-x86"
            label="Intel"
            checked={form.supported_architectures.includes("x86_64")}
            onChange={(checked) => onChange({ ...form, supported_architectures: toggleArch(form, "x86_64", checked) })}
          />
        </div>
        <FieldDescription>Leave both unchecked when the item applies to every supported Mac.</FieldDescription>
      </Field>
      <StringListField
        id="munki-package-blocking-applications"
        label="Blocking Applications"
        description="Application names Munki should check before install or removal."
        value={form.blocking_applications}
        onChange={(blocking_applications) => onChange({ ...form, blocking_applications })}
      />
      <StringListField
        id="munki-package-requires"
        label="Requires"
        description="Munki item names that must be installed before this item."
        value={form.requires}
        onChange={(requires) => onChange({ ...form, requires })}
      />
      <StringListField
        id="munki-package-update-for"
        label="Update For"
        description="Munki item names this item updates when already present."
        value={form.update_for}
        onChange={(update_for) => onChange({ ...form, update_for })}
      />
    </FieldSet>
  );
}

function cleanStringArray(values: string[]) {
  return values.map((value) => value.trim()).filter(Boolean);
}

function parseStringList(value: string) {
  return cleanStringArray(value.split(","));
}

function formatStringList(values: string[]) {
  return values.join(", ");
}

function toggleArch(form: PackageFormState, arch: "arm64" | "x86_64", checked: boolean) {
  if (checked) return Array.from(new Set([...form.supported_architectures, arch]));
  return form.supported_architectures.filter((value) => value !== arch);
}

function isSupportedArch(value: string): value is "arm64" | "x86_64" {
  return value === "arm64" || value === "x86_64";
}
