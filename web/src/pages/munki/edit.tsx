import { Link, useNavigate, useParams } from "@tanstack/react-router";
import { FileArchive, Loader2 } from "lucide-react";
import { useEffect, useMemo, useState, type ReactNode, type SyntheticEvent } from "react";
import { z } from "zod";

import { LabelPicker } from "@/components/labels/label-picker";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { EditableMunkiIcon } from "@/components/munki/editable-munki-icon";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
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
import {
  useCreateMunkiArtifact,
  useCreateMunkiArtifactUpload,
  useCreateMunkiAssignment,
  useCreateMunkiPackage,
  useCreateMunkiSoftwareTitle,
  useMunkiAssignment,
  useMunkiPackage,
  useMunkiSoftwareTitle,
  useMunkiSoftwareTitles,
  useUpdateMunkiAssignment,
  useUpdateMunkiPackage,
  useUpdateMunkiSoftwareTitle,
  type MunkiArtifact,
  type MunkiArtifactMutation,
  type MunkiArtifactUploadMutation,
  type MunkiAssignmentMutation,
  type MunkiPackageMutation,
  type MunkiSoftwareTitleMutation,
} from "@/hooks/use-munki";
import { fieldErrors, requiredString } from "@/lib/form-validation";
import { MAX_PAGE_SIZE } from "@/lib/pagination";

type AssignmentAction = NonNullable<MunkiAssignmentMutation["action"]>;
type AssignmentEffect = MunkiAssignmentMutation["effect"];
type PackageSelection = NonNullable<MunkiAssignmentMutation["package_selection"]>;
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

const actionOptions: { value: AssignmentAction; label: string; description: string }[] = [
  { value: "install", label: "Managed Installs", description: "Forces installation by writing managed_installs." },
  { value: "remove", label: "Managed Uninstalls", description: "Forces removal by writing managed_uninstalls." },
  {
    value: "update_if_present",
    label: "Managed Updates",
    description: "Updates installed items by writing managed_updates.",
  },
  {
    value: "none",
    label: "No managed section",
    description: "Only Optional Installs and Featured Items section membership is rendered.",
  },
];

const packageSelectionOptions: { value: PackageSelection; label: string; description: string }[] = [
  {
    value: "latest_eligible",
    label: "Latest compatible",
    description: "Render the Munki name and include all eligible pkginfos for the client to choose from.",
  },
  {
    value: "specific_package",
    label: "Pinned package",
    description: "Render Name--Version and include only that pkginfo candidate.",
  },
];

const softwareTitleSchema = z.object({
  name: requiredString("Name"),
  display_name: z.string().trim(),
  description: z.string().trim(),
  category: z.string().trim(),
  developer: z.string().trim(),
});

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

const assignmentSchema = z
  .object({
    priority: z.number().int("Priority must be a whole number.").positive("Priority must be at least 1."),
    effect: z.enum(["include", "exclude"]),
    label_id: z
      .string()
      .trim()
      .refine((value) => Number(value) > 0, "Label is required."),
    package_selection: z.enum(["latest_eligible", "specific_package"]),
    pinned_package_id: z.string().trim(),
    action: z.enum(["install", "remove", "update_if_present", "none"]),
    optional_install: z.boolean(),
    featured_item: z.boolean(),
  })
  .superRefine((value, ctx) => {
    if (value.effect === "exclude") return;
    if (value.package_selection === "specific_package" && !Number(value.pinned_package_id)) {
      ctx.addIssue({ code: "custom", message: "Package is required.", path: ["pinned_package_id"] });
    }
    if (value.featured_item && !value.optional_install) {
      ctx.addIssue({
        code: "custom",
        message: "Featured Items must also be Optional Installs.",
        path: ["featured_item"],
      });
    }
    if (value.action === "remove" && (value.optional_install || value.featured_item)) {
      ctx.addIssue({
        code: "custom",
        message: "Managed Uninstalls cannot also be Optional Installs or Featured Items.",
        path: ["optional_install"],
      });
    }
  });

interface SoftwareTitleFormState {
  name: string;
  display_name: string;
  description: string;
  category: string;
  developer: string;
}

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

interface AssignmentFormState {
  priority: number;
  effect: AssignmentEffect;
  label_id: string;
  package_selection: PackageSelection;
  pinned_package_id: string;
  action: AssignmentAction;
  optional_install: boolean;
  featured_item: boolean;
}

export function MunkiSoftwareTitleNewPage() {
  const navigate = useNavigate();
  const create = useCreateMunkiSoftwareTitle();
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
  const [form, setForm] = useState<SoftwareTitleFormState>({
    name: "",
    display_name: "",
    description: "",
    category: "",
    developer: "",
  });
  const [iconFile, setIconFile] = useState<File | null>(null);
  const [showErrors, setShowErrors] = useState(false);
  const parsed = useMemo(() => softwareTitleSchema.safeParse(form), [form]);
  const errors = useMemo(() => fieldErrors(parsed), [parsed]);

  async function submit() {
    const next = softwareTitleSchema.safeParse(form);
    if (!next.success) {
      setShowErrors(true);
      return;
    }
    const iconArtifact = iconFile
      ? await uploadSelectedArtifact(iconFile, "icon", createUpload.mutateAsync, createArtifact.mutateAsync)
      : null;
    const body: MunkiSoftwareTitleMutation = {
      ...next.data,
      icon_artifact_id: iconArtifact?.id,
    };
    const title = await create.mutateAsync(body);
    void navigate({ to: "/munki/software-titles/$softwareId", params: { softwareId: String(title.id) } });
  }

  return (
    <PageShell asChild>
      <form noValidate onSubmit={(event) => runSubmit(event, submit)}>
        <PageHeader
          title="New Software"
          description="Create the software title admins target. Add package versions and assignments after the title exists."
          leading={
            <EditableMunkiIcon
              title="software icon"
              file={iconFile}
              clearable={!!iconFile}
              onFileChange={setIconFile}
              onClear={() => setIconFile(null)}
            />
          }
        />
        <MutationError
          title="Failed to Create Software"
          message={create.error?.message ?? createUpload.error?.message ?? createArtifact.error?.message}
        />
        <FieldGroup className="max-w-3xl">
          <TextField
            id="munki-software-name"
            label="Name"
            required
            description="Stable Munki item name. Use Display Name for spaces, punctuation, and nicer casing."
            value={form.name}
            error={showErrors ? errors.name : undefined}
            onChange={(name) => setForm({ ...form, name })}
          />
          <TextField
            id="munki-software-display-name"
            label="Display Name"
            value={form.display_name}
            onChange={(display_name) => setForm({ ...form, display_name })}
          />
          <TextAreaField
            id="munki-software-description"
            label="Description"
            value={form.description}
            onChange={(description) => setForm({ ...form, description })}
          />
          <div className="grid gap-4 md:grid-cols-2">
            <DatalistTextField
              id="munki-software-category"
              label="Category"
              value={form.category}
              options={categoryOptions}
              onChange={(category) => setForm({ ...form, category })}
            />
            <DatalistTextField
              id="munki-software-developer"
              label="Developer"
              value={form.developer}
              options={developerOptions}
              onChange={(developer) => setForm({ ...form, developer })}
            />
          </div>
          <FormActions
            pending={create.isPending || createUpload.isPending || createArtifact.isPending}
            cancelTo="/munki/software-titles"
          />
        </FieldGroup>
      </form>
    </PageShell>
  );
}

export function MunkiSoftwareTitleEditPage() {
  const navigate = useNavigate();
  const softwareId = useSoftwareIDParam();
  const software = useMunkiSoftwareTitle(softwareId);
  const update = useUpdateMunkiSoftwareTitle();
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
  const [form, setForm] = useState<SoftwareTitleFormState>({
    name: "",
    display_name: "",
    description: "",
    category: "",
    developer: "",
  });
  const [iconFile, setIconFile] = useState<File | null>(null);
  const [iconCleared, setIconCleared] = useState(false);
  const [showErrors, setShowErrors] = useState(false);
  const parsed = useMemo(() => softwareTitleSchema.safeParse(form), [form]);
  const errors = useMemo(() => fieldErrors(parsed), [parsed]);

  useEffect(() => {
    if (!software.data) return;
    setForm({
      name: software.data.name,
      display_name: software.data.display_name,
      description: software.data.description,
      category: software.data.category,
      developer: software.data.developer,
    });
    setIconFile(null);
    setIconCleared(false);
  }, [software.data]);

  async function submit() {
    const next = softwareTitleSchema.safeParse(form);
    if (!next.success || softwareId === null) {
      setShowErrors(true);
      return;
    }
    const iconArtifact = iconFile
      ? await uploadSelectedArtifact(iconFile, "icon", createUpload.mutateAsync, createArtifact.mutateAsync)
      : null;
    const body: MunkiSoftwareTitleMutation = {
      ...next.data,
      icon_artifact_id: iconArtifact?.id ?? (iconCleared ? undefined : software.data?.icon_artifact_id),
    };
    const title = await update.mutateAsync({ id: softwareId, body });
    void navigate({ to: "/munki/software-titles/$softwareId", params: { softwareId: String(title.id) } });
  }

  return (
    <PageShell asChild>
      <form noValidate onSubmit={(event) => runSubmit(event, submit)}>
        <PageHeader
          title="Edit Software"
          description="Edit the software title admins target. Package versions inherit this icon unless they override it."
          leading={
            <EditableMunkiIcon
              title="software icon"
              iconUrl={iconCleared ? undefined : software.data?.icon_url}
              file={iconFile}
              clearable={!!iconFile || (!iconCleared && !!software.data?.icon_artifact_id)}
              onFileChange={(file) => {
                setIconFile(file);
                setIconCleared(false);
              }}
              onClear={() => {
                setIconFile(null);
                setIconCleared(!!software.data?.icon_artifact_id);
              }}
            />
          }
        />
        <MutationError
          title="Failed to Update Software"
          message={
            update.error?.message ??
            createUpload.error?.message ??
            createArtifact.error?.message ??
            software.error?.message
          }
        />
        <FieldGroup className="max-w-3xl">
          <TextField
            id="munki-software-name"
            label="Name"
            required
            description="Stable Munki item name. Use Display Name for spaces, punctuation, and nicer casing."
            value={form.name}
            error={showErrors ? errors.name : undefined}
            onChange={(name) => setForm({ ...form, name })}
          />
          <TextField
            id="munki-software-display-name"
            label="Display Name"
            value={form.display_name}
            onChange={(display_name) => setForm({ ...form, display_name })}
          />
          <TextAreaField
            id="munki-software-description"
            label="Description"
            value={form.description}
            onChange={(description) => setForm({ ...form, description })}
          />
          <div className="grid gap-4 md:grid-cols-2">
            <DatalistTextField
              id="munki-software-category"
              label="Category"
              value={form.category}
              options={categoryOptions}
              onChange={(category) => setForm({ ...form, category })}
            />
            <DatalistTextField
              id="munki-software-developer"
              label="Developer"
              value={form.developer}
              options={developerOptions}
              onChange={(developer) => setForm({ ...form, developer })}
            />
          </div>
          <FormActions
            pending={update.isPending || createUpload.isPending || createArtifact.isPending || software.isLoading}
            cancelTo="/munki/software-titles/$softwareId"
            cancelParams={{ softwareId: String(softwareId ?? "") }}
          />
        </FieldGroup>
      </form>
    </PageShell>
  );
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

export function MunkiAssignmentNewPage() {
  const navigate = useNavigate();
  const softwareId = useSoftwareIDParam();
  const software = useMunkiSoftwareTitle(softwareId);
  const create = useCreateMunkiAssignment();
  const packages = software.data?.packages ?? [];
  const [form, setForm] = useState<AssignmentFormState>({
    priority: 1,
    effect: "include",
    label_id: "",
    package_selection: "latest_eligible",
    pinned_package_id: "",
    action: "install",
    optional_install: false,
    featured_item: false,
  });
  const [showErrors, setShowErrors] = useState(false);
  const parsed = useMemo(() => assignmentSchema.safeParse(form), [form]);
  const errors = useMemo(() => fieldErrors(parsed), [parsed]);

  useEffect(() => {
    if (!software.data) return;
    setForm((current) => ({
      ...current,
      priority: current.priority === 1 ? (software.data.assignments?.length ?? 0) + 1 : current.priority,
    }));
  }, [software.data]);

  async function submit() {
    const next = assignmentSchema.safeParse(form);
    if (!next.success || softwareId === null) {
      setShowErrors(true);
      return;
    }
    const body = assignmentBody(softwareId, next.data);
    await create.mutateAsync(body);
    void navigate({ to: "/munki/software-titles/$softwareId", params: { softwareId: String(softwareId) } });
  }

  return (
    <PageShell asChild>
      <form noValidate onSubmit={(event) => runSubmit(event, submit)}>
        <PageHeader title="New Assignment" description="Priority decides which matching row wins for this software." />
        <MutationError title="Failed to Create Assignment" message={create.error?.message ?? software.error?.message} />
        <AssignmentFormFields
          form={form}
          packages={packages}
          showErrors={showErrors}
          errors={errors}
          loadingPackages={software.isLoading}
          onChange={setForm}
        />
        <FormActions
          pending={create.isPending}
          cancelTo="/munki/software-titles/$softwareId"
          cancelParams={{ softwareId: String(softwareId ?? "") }}
        />
      </form>
    </PageShell>
  );
}

export function MunkiAssignmentEditPage() {
  const navigate = useNavigate();
  const softwareId = useSoftwareIDParam();
  const assignmentId = useAssignmentIDParam();
  const software = useMunkiSoftwareTitle(softwareId);
  const assignment = useMunkiAssignment(assignmentId);
  const update = useUpdateMunkiAssignment();
  const packages = software.data?.packages ?? [];
  const [form, setForm] = useState<AssignmentFormState>({
    priority: 1,
    effect: "include",
    label_id: "",
    package_selection: "latest_eligible",
    pinned_package_id: "",
    action: "install",
    optional_install: false,
    featured_item: false,
  });
  const [showErrors, setShowErrors] = useState(false);
  const parsed = useMemo(() => assignmentSchema.safeParse(form), [form]);
  const errors = useMemo(() => fieldErrors(parsed), [parsed]);

  useEffect(() => {
    if (!assignment.data) return;
    setForm({
      priority: assignment.data.priority,
      effect: assignment.data.effect,
      label_id: String(assignment.data.label_id),
      package_selection: assignment.data.package_selection ?? "latest_eligible",
      pinned_package_id: assignment.data.pinned_package_id ? String(assignment.data.pinned_package_id) : "",
      action: assignment.data.action ?? "install",
      optional_install: assignment.data.optional_install,
      featured_item: assignment.data.featured_item,
    });
  }, [assignment.data]);

  async function submit() {
    const next = assignmentSchema.safeParse(form);
    if (!next.success || softwareId === null || assignmentId === null) {
      setShowErrors(true);
      return;
    }
    const body = assignmentBody(softwareId, next.data);
    await update.mutateAsync({ id: assignmentId, body });
    void navigate({ to: "/munki/software-titles/$softwareId", params: { softwareId: String(softwareId) } });
  }

  return (
    <PageShell asChild>
      <form noValidate onSubmit={(event) => runSubmit(event, submit)}>
        <PageHeader title="Edit Assignment" description="Priority 1 is evaluated first." />
        <MutationError
          title="Failed to Update Assignment"
          message={update.error?.message ?? assignment.error?.message ?? software.error?.message}
        />
        <AssignmentFormFields
          form={form}
          packages={packages}
          showErrors={showErrors}
          errors={errors}
          loadingPackages={software.isLoading}
          onChange={setForm}
        />
        <FormActions
          pending={update.isPending || assignment.isLoading}
          cancelTo="/munki/software-titles/$softwareId"
          cancelParams={{ softwareId: String(softwareId ?? "") }}
        />
      </form>
    </PageShell>
  );
}

function AssignmentFormFields({
  form,
  packages,
  showErrors,
  errors,
  loadingPackages,
  onChange,
}: {
  form: AssignmentFormState;
  packages: Array<{ id: number; version: string; display_name?: string; name: string }>;
  showErrors: boolean;
  errors: Record<string, string>;
  loadingPackages: boolean;
  onChange: (next: AssignmentFormState) => void;
}) {
  const include = form.effect === "include";
  return (
    <FieldGroup className="max-w-3xl">
      <Field data-invalid={showErrors && errors.priority ? true : undefined}>
        <FieldLabel htmlFor="munki-assignment-priority" required>
          Priority
        </FieldLabel>
        <Input
          id="munki-assignment-priority"
          type="number"
          min={1}
          step={1}
          required
          inputMode="numeric"
          aria-invalid={showErrors && errors.priority ? true : undefined}
          value={form.priority}
          onChange={(event) => onChange({ ...form, priority: Number(event.target.value) })}
        />
        {showErrors && errors.priority ? <FieldError>{errors.priority}</FieldError> : null}
      </Field>

      <Field data-invalid={showErrors && errors.label_id ? true : undefined}>
        <FieldLabel required>Label</FieldLabel>
        <LabelPicker
          value={form.label_id ? [Number(form.label_id)] : []}
          onChange={(labelIDs) => onChange({ ...form, label_id: labelIDs[0] ? String(labelIDs[0]) : "" })}
          selectionMode="single"
          includeBuiltins
          placeholder="Select label"
          required
          invalid={showErrors && errors.label_id ? true : undefined}
        />
        {showErrors && errors.label_id ? <FieldError>{errors.label_id}</FieldError> : null}
      </Field>

      <Field>
        <FieldLabel htmlFor="munki-assignment-effect" required>
          Effect
        </FieldLabel>
        <Select
          value={form.effect}
          onValueChange={(effect) =>
            onChange({
              ...form,
              effect: effect as AssignmentEffect,
              optional_install: effect === "exclude" ? false : form.optional_install,
              featured_item: effect === "exclude" ? false : form.featured_item,
            })
          }
        >
          <SelectTrigger id="munki-assignment-effect" className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              <SelectItem value="include">Include</SelectItem>
              <SelectItem value="exclude">Exclude</SelectItem>
            </SelectGroup>
          </SelectContent>
        </Select>
      </Field>

      {include ? (
        <>
          <Field>
            <FieldLabel htmlFor="munki-assignment-selection" required>
              Package Selection
            </FieldLabel>
            <Select
              value={form.package_selection}
              onValueChange={(package_selection) =>
                onChange({
                  ...form,
                  package_selection: package_selection as PackageSelection,
                  pinned_package_id: package_selection === "latest_eligible" ? "" : form.pinned_package_id,
                })
              }
            >
              <SelectTrigger id="munki-assignment-selection" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {packageSelectionOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
            <FieldDescription>{packageSelectionDescription(form.package_selection)}</FieldDescription>
          </Field>

          {form.package_selection === "specific_package" ? (
            <Field data-invalid={showErrors && errors.pinned_package_id ? true : undefined}>
              <FieldLabel htmlFor="munki-assignment-package" required>
                Pinned Package
              </FieldLabel>
              <Select
                value={form.pinned_package_id}
                onValueChange={(pinned_package_id) => onChange({ ...form, pinned_package_id })}
              >
                <SelectTrigger id="munki-assignment-package" className="w-full">
                  <SelectValue placeholder={loadingPackages ? "Loading..." : "Select Package"} />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    {packages.map((pkg) => (
                      <SelectItem key={pkg.id} value={String(pkg.id)}>
                        {pkg.version} · {pkg.display_name ?? pkg.name}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
              <FieldDescription>Rendered as Name--Version in the manifest.</FieldDescription>
              {showErrors && errors.pinned_package_id ? <FieldError>{errors.pinned_package_id}</FieldError> : null}
            </Field>
          ) : null}

          <Field>
            <FieldLabel htmlFor="munki-assignment-action" required>
              Managed Section
            </FieldLabel>
            <Select
              value={form.action}
              onValueChange={(action) =>
                onChange({
                  ...form,
                  action: action as AssignmentAction,
                  optional_install: action === "remove" ? false : form.optional_install,
                  featured_item: action === "remove" ? false : form.featured_item,
                })
              }
            >
              <SelectTrigger id="munki-assignment-action" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {actionOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
            <FieldDescription>{actionDescription(form.action)}</FieldDescription>
          </Field>

          <FieldSet>
            <FieldLegend>Managed Software Centre</FieldLegend>
            <FieldDescription>These write the optional_installs and featured_items manifest sections.</FieldDescription>
            <CheckboxField
              id="munki-assignment-optional-install"
              label="Optional Installs"
              description="Adds this item to optional_installs so it appears in MSC."
              checked={form.optional_install}
              disabled={form.action === "remove"}
              onChange={(optional_install) =>
                onChange({
                  ...form,
                  optional_install,
                  featured_item: optional_install ? form.featured_item : false,
                })
              }
            />
            <CheckboxField
              id="munki-assignment-featured-item"
              label="Featured Items"
              description="Also adds this item to featured_items. Munki expects featured items to also be optional installs."
              checked={form.featured_item}
              disabled={form.action === "remove"}
              onChange={(featured_item) =>
                onChange({
                  ...form,
                  optional_install: featured_item ? true : form.optional_install,
                  featured_item,
                })
              }
            />
            {showErrors && errors.optional_install ? <FieldError>{errors.optional_install}</FieldError> : null}
            {showErrors && errors.featured_item ? <FieldError>{errors.featured_item}</FieldError> : null}
          </FieldSet>
        </>
      ) : null}
    </FieldGroup>
  );
}

function assignmentBody(softwareId: number, form: AssignmentFormState): MunkiAssignmentMutation {
  const body: MunkiAssignmentMutation = {
    software_id: softwareId,
    priority: form.priority,
    effect: form.effect,
    label_id: Number(form.label_id),
  };
  if (form.effect === "exclude") {
    return body;
  }
  return {
    ...body,
    action: form.action,
    optional_install: form.optional_install,
    featured_item: form.featured_item,
    package_selection: form.package_selection,
    pinned_package_id: form.package_selection === "specific_package" ? Number(form.pinned_package_id) : undefined,
  };
}

function useSoftwareIDParam() {
  const params = useParams({ strict: false });
  const id = Number(params.softwareId);
  return Number.isFinite(id) && id > 0 ? id : null;
}

function usePackageIDParam() {
  const params = useParams({ strict: false });
  const id = Number(params.packageId);
  return Number.isFinite(id) && id > 0 ? id : null;
}

function useAssignmentIDParam() {
  const params = useParams({ strict: false });
  const id = Number(params.assignmentId);
  return Number.isFinite(id) && id > 0 ? id : null;
}

function packageSelectionDescription(value: PackageSelection) {
  return packageSelectionOptions.find((option) => option.value === value)?.description;
}

function actionDescription(value: AssignmentAction) {
  return actionOptions.find((option) => option.value === value)?.description;
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

function TextField({
  id,
  label,
  required,
  value,
  error,
  placeholder,
  description,
  onChange,
}: {
  id: string;
  label: string;
  required?: boolean;
  value: string;
  error?: string;
  placeholder?: string;
  description?: string;
  onChange: (value: string) => void;
}) {
  return (
    <Field data-invalid={error ? true : undefined}>
      <FieldLabel htmlFor={id} required={required}>
        {label}
      </FieldLabel>
      <Input id={id} value={value} placeholder={placeholder} onChange={(event) => onChange(event.target.value)} />
      {description ? <FieldDescription>{description}</FieldDescription> : null}
      {error ? <FieldError>{error}</FieldError> : null}
    </Field>
  );
}

function StringListField({
  id,
  label,
  description,
  value,
  onChange,
}: {
  id: string;
  label: string;
  description: string;
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <Field>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <Input id={id} value={value} placeholder="ItemA, ItemB" onChange={(event) => onChange(event.target.value)} />
      <FieldDescription>{description}</FieldDescription>
    </Field>
  );
}

function DatalistTextField({
  id,
  label,
  value,
  options,
  placeholder,
  onChange,
}: {
  id: string;
  label: string;
  value: string;
  options: string[];
  placeholder?: string;
  onChange: (value: string) => void;
}) {
  const listID = `${id}-options`;
  return (
    <Field>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <Input
        id={id}
        list={options.length > 0 ? listID : undefined}
        value={value}
        placeholder={placeholder}
        onChange={(event) => onChange(event.target.value)}
      />
      {options.length > 0 ? (
        <datalist id={listID}>
          {options.map((option) => (
            <option key={option} value={option} />
          ))}
        </datalist>
      ) : null}
    </Field>
  );
}

function SelectField<T extends string>({
  id,
  label,
  value,
  options,
  description,
  onChange,
}: {
  id: string;
  label: string;
  value: T;
  options: Array<{ value: T; label: string; description?: string }>;
  description?: string;
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
      {description ? <FieldDescription>{description}</FieldDescription> : null}
    </Field>
  );
}

function FileField({
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

function TextAreaField({
  id,
  label,
  value,
  placeholder,
  onChange,
}: {
  id: string;
  label: string;
  value: string;
  placeholder?: string;
  onChange: (value: string) => void;
}) {
  return (
    <Field>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <Textarea id={id} value={value} placeholder={placeholder} onChange={(event) => onChange(event.target.value)} />
    </Field>
  );
}

function CheckboxField({
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

function FormActions({
  pending,
  cancelTo,
  cancelParams,
}: {
  pending: boolean;
  cancelTo: string;
  cancelParams?: Record<string, string>;
}) {
  return (
    <div className="flex items-center gap-2">
      <Button type="submit" size="sm" disabled={pending}>
        {pending ? <Loader2 data-icon="inline-start" className="animate-spin" /> : null}
        Save
      </Button>
      <Button asChild type="button" variant="outline" size="sm">
        <Link to={cancelTo} params={cancelParams}>
          Cancel
        </Link>
      </Button>
    </div>
  );
}

function MutationError({ title, message }: { title: string; message?: string }) {
  if (!message) return null;
  return (
    <Alert variant="destructive">
      <AlertTitle>{title}</AlertTitle>
      <AlertDescription>{message}</AlertDescription>
    </Alert>
  );
}

function optionalText(value: string) {
  const trimmed = value.trim();
  return trimmed === "" ? undefined : trimmed;
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

function uniqueOptions(values: string[]) {
  return Array.from(new Set(values.map((value) => value.trim()).filter(Boolean))).sort((a, b) => a.localeCompare(b));
}

async function uploadSelectedArtifact(
  file: File,
  kind: "package" | "icon",
  createUpload: (body: MunkiArtifactUploadMutation) => Promise<{
    upload_url: string;
    headers?: Record<string, string>;
    artifact: MunkiArtifactMutation;
  }>,
  createArtifact: (body: MunkiArtifactMutation) => Promise<MunkiArtifact>,
) {
  const sha256 = await fileSHA256(file);
  const upload = await createUpload({
    kind,
    filename: file.name,
    content_type: file.type || undefined,
    size_bytes: file.size,
    sha256,
  });
  const headers = new Headers(upload.headers);
  if (file.type && !headers.has("Content-Type")) {
    headers.set("Content-Type", file.type);
  }
  const response = await fetch(upload.upload_url, {
    method: "PUT",
    headers,
    body: file,
  });
  if (!response.ok) {
    throw new Error(`Upload failed with HTTP ${response.status}`);
  }
  return createArtifact(upload.artifact);
}

async function fileSHA256(file: File) {
  const digest = await crypto.subtle.digest("SHA-256", await file.arrayBuffer());
  return Array.from(new Uint8Array(digest))
    .map((byte) => byte.toString(16).padStart(2, "0"))
    .join("");
}

function runSubmit(event: SyntheticEvent<HTMLFormElement>, submit: () => Promise<void>) {
  event.preventDefault();
  void submit();
}
