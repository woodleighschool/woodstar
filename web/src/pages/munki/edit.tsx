import { Link, useNavigate, useParams } from "@tanstack/react-router";
import { FileArchive, ImageIcon, Loader2 } from "lucide-react";
import { useEffect, useMemo, useState, type ReactNode, type SyntheticEvent } from "react";
import { z } from "zod";

import { LabelPicker } from "@/components/labels/label-picker";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
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
import { Textarea } from "@/components/ui/textarea";
import {
  useCreateMunkiArtifact,
  useCreateMunkiArtifactUpload,
  useCreateMunkiDeployment,
  useCreateMunkiPackage,
  useCreateMunkiSoftwareTitle,
  useMunkiSoftwareTitle,
  useMunkiSoftwareTitles,
  type MunkiArtifact,
  type MunkiArtifactMutation,
  type MunkiArtifactUploadMutation,
  type MunkiDeploymentMutation,
  type MunkiPackageMutation,
  type MunkiSoftwareTitleMutation,
} from "@/hooks/use-munki";
import { fieldErrors, requiredString } from "@/lib/form-validation";
import { MAX_PAGE_SIZE } from "@/lib/pagination";

type Intent = MunkiDeploymentMutation["intent"];
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

const intentOptions: { value: Intent; label: string }[] = [
  { value: "ensure_installed", label: "Install and update" },
  { value: "ensure_absent", label: "Remove" },
  { value: "update_if_present", label: "Update if present" },
  { value: "optional", label: "Self Service" },
  { value: "featured", label: "Featured" },
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
  minimum_os_version: z.string().trim(),
  maximum_os_version: z.string().trim(),
  supported_architectures: z.array(z.enum(["arm64", "x86_64"])),
  eligible: z.boolean(),
  unattended_install: z.boolean(),
  unattended_uninstall: z.boolean(),
  uninstallable: z.boolean(),
  on_demand: z.boolean(),
  precache: z.boolean(),
});

const deploymentSchema = z
  .object({
    package_id: z.coerce.number().int().positive("Package is required."),
    intent: z.enum(["ensure_installed", "ensure_absent", "update_if_present", "optional", "featured"]),
    all_hosts: z.boolean(),
    include_label_ids: z.array(z.number().int().positive()),
    exclude_label_ids: z.array(z.number().int().positive()),
  })
  .superRefine((value, ctx) => {
    if (!value.all_hosts && value.include_label_ids.length === 0) {
      ctx.addIssue({ code: "custom", message: "Select at least one target label.", path: ["include_label_ids"] });
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
  minimum_os_version: string;
  maximum_os_version: string;
  supported_architectures: Array<"arm64" | "x86_64">;
  eligible: boolean;
  unattended_install: boolean;
  unattended_uninstall: boolean;
  uninstallable: boolean;
  on_demand: boolean;
  precache: boolean;
}

interface DeploymentFormState {
  package_id: string;
  intent: Intent;
  all_hosts: boolean;
  include_label_ids: number[];
  exclude_label_ids: number[];
}

export function MunkiSoftwareTitleNewPage() {
  const navigate = useNavigate();
  const create = useCreateMunkiSoftwareTitle();
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
  const [showErrors, setShowErrors] = useState(false);
  const parsed = useMemo(() => softwareTitleSchema.safeParse(form), [form]);
  const errors = useMemo(() => fieldErrors(parsed), [parsed]);

  async function submit() {
    const next = softwareTitleSchema.safeParse(form);
    if (!next.success) {
      setShowErrors(true);
      return;
    }
    const body: MunkiSoftwareTitleMutation = next.data;
    const title = await create.mutateAsync(body);
    void navigate({ to: "/munki/software-titles/$softwareId", params: { softwareId: String(title.id) } });
  }

  return (
    <PageShell asChild>
      <form noValidate onSubmit={(event) => runSubmit(event, submit)}>
        <PageHeader
          title="New Software"
          description="Create the software title admins target. Add package versions and deployments after the title exists."
        />
        <MutationError title="Failed to Create Software" message={create.error?.message} />
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
          <FormActions pending={create.isPending} cancelTo="/munki/software-titles" />
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
    minimum_os_version: "",
    maximum_os_version: "",
    supported_architectures: [],
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
      minimum_os_version: optionalText(next.data.minimum_os_version),
      maximum_os_version: optionalText(next.data.maximum_os_version),
      supported_architectures:
        next.data.supported_architectures.length > 0 ? next.data.supported_architectures : undefined,
      on_demand: next.data.on_demand,
      precache: next.data.precache,
      installer_artifact_id: installerArtifact?.id,
      icon_artifact_id: iconArtifact?.id,
      icon_name: iconArtifact?.location,
      icon_hash: iconArtifact?.sha256,
    };
    await create.mutateAsync(body);
    void navigate({ to: "/munki/software-titles/$softwareId", params: { softwareId: String(softwareId) } });
  }

  return (
    <PageShell asChild>
      <form noValidate onSubmit={(event) => runSubmit(event, submit)}>
        <PageHeader
          title="New Package"
          description="Add a typed pkginfo package with optional installer and icon files."
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
        <FieldGroup className="max-w-3xl">
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
              description="Rendered into pkginfo and shown when choosing a deployment package."
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
          <FieldSet>
            <FieldLegend>Artifacts</FieldLegend>
            <FieldDescription>
              Files upload to Munki storage first. Woodstar stores the artifact reference and renders the stable Munki
              URL.
            </FieldDescription>
            <FieldGroup className="grid gap-4 md:grid-cols-2">
              <FileField
                id="munki-package-installer-file"
                label="Installer"
                description="Optional package, disk image, profile, or metadata payload used by this pkginfo."
                icon={<FileArchive className="size-4" />}
                file={installerFile}
                onChange={setInstallerFile}
              />
              <FileField
                id="munki-package-icon-file"
                label="Icon"
                description="Optional app icon. If unset, Woodstar shows a package icon in the admin UI."
                accept="image/png,image/jpeg,image/webp,image/icns,.icns"
                icon={<ImageIcon className="size-4" />}
                file={iconFile}
                onChange={setIconFile}
              />
            </FieldGroup>
          </FieldSet>
          <FieldSet>
            <FieldLegend>Package Behavior</FieldLegend>
            <FieldDescription>
              These values are rendered into pkginfo. They do not inspect the installer bytes.
            </FieldDescription>
            <CheckboxField
              id="munki-package-eligible"
              label="Available for deployment"
              description="Ineligible packages stay stored but are skipped when manifests and catalogs are rendered."
              checked={form.eligible}
              onChange={(eligible) => setForm({ ...form, eligible })}
            />
            <FieldGroup className="grid gap-4 md:grid-cols-3">
              <CheckboxField
                id="munki-package-unattended-install"
                label="Unattended install"
                description="Allows Munki to install this item without Self Service interaction."
                checked={form.unattended_install}
                onChange={(unattended_install) => setForm({ ...form, unattended_install })}
              />
              <CheckboxField
                id="munki-package-unattended-uninstall"
                label="Unattended uninstall"
                description="Allows Munki to remove this item without Self Service interaction."
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
              <TextField
                id="munki-package-minimum-os"
                label="Minimum OS"
                value={form.minimum_os_version}
                placeholder="14.0"
                onChange={(minimum_os_version) => setForm({ ...form, minimum_os_version })}
              />
              <TextField
                id="munki-package-maximum-os"
                label="Maximum OS"
                value={form.maximum_os_version}
                placeholder="15.99"
                onChange={(maximum_os_version) => setForm({ ...form, maximum_os_version })}
              />
            </FieldGroup>
            <Field>
              <FieldLabel>Supported Architectures</FieldLabel>
              <div className="grid gap-3 md:grid-cols-2">
                <CheckboxField
                  id="munki-package-arch-arm64"
                  label="Apple silicon"
                  checked={form.supported_architectures.includes("arm64")}
                  onChange={(checked) =>
                    setForm({ ...form, supported_architectures: toggleArch(form, "arm64", checked) })
                  }
                />
                <CheckboxField
                  id="munki-package-arch-x86"
                  label="Intel"
                  checked={form.supported_architectures.includes("x86_64")}
                  onChange={(checked) =>
                    setForm({ ...form, supported_architectures: toggleArch(form, "x86_64", checked) })
                  }
                />
              </div>
              <FieldDescription>Leave both unchecked when the item applies to every supported Mac.</FieldDescription>
            </Field>
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
          <FormActions
            pending={create.isPending || createUpload.isPending || createArtifact.isPending}
            cancelTo="/munki/software-titles/$softwareId"
            cancelParams={{ softwareId: String(softwareId ?? "") }}
          />
        </FieldGroup>
      </form>
    </PageShell>
  );
}

export function MunkiDeploymentNewPage() {
  const navigate = useNavigate();
  const softwareId = useSoftwareIDParam();
  const software = useMunkiSoftwareTitle(softwareId);
  const create = useCreateMunkiDeployment();
  const packages = software.data?.packages ?? [];
  const [form, setForm] = useState<DeploymentFormState>({
    package_id: "",
    intent: "ensure_installed",
    all_hosts: true,
    include_label_ids: [],
    exclude_label_ids: [],
  });
  const [showErrors, setShowErrors] = useState(false);
  const parsed = useMemo(
    () =>
      deploymentSchema.safeParse({
        ...form,
        package_id: form.package_id,
      }),
    [form],
  );
  const errors = useMemo(() => fieldErrors(parsed), [parsed]);

  useEffect(() => {
    const firstPackage = software.data?.packages?.[0];
    if (!firstPackage) return;
    setForm((current) => (current.package_id ? current : { ...current, package_id: String(firstPackage.id) }));
  }, [software.data?.packages]);

  async function submit() {
    const next = deploymentSchema.safeParse(form);
    if (!next.success || softwareId === null) {
      setShowErrors(true);
      return;
    }
    const body: MunkiDeploymentMutation = {
      package_id: next.data.package_id,
      intent: next.data.intent,
      all_hosts: next.data.all_hosts,
      include_label_ids: next.data.all_hosts ? [] : next.data.include_label_ids,
      exclude_label_ids: next.data.exclude_label_ids,
    };
    await create.mutateAsync(body);
    void navigate({ to: "/munki/software-titles/$softwareId", params: { softwareId: String(softwareId) } });
  }

  return (
    <PageShell asChild>
      <form noValidate onSubmit={(event) => runSubmit(event, submit)}>
        <PageHeader
          title="New Deployment"
          description="Attach a package version to a Munki intent and a label scope. Deployment order is edited on the software page."
        />
        <MutationError title="Failed to Create Deployment" message={create.error?.message ?? software.error?.message} />
        <FieldGroup className="max-w-3xl">
          <Field data-invalid={showErrors && errors.package_id ? true : undefined}>
            <FieldLabel htmlFor="munki-deployment-package" required>
              Package
            </FieldLabel>
            <Select value={form.package_id} onValueChange={(package_id) => setForm({ ...form, package_id })}>
              <SelectTrigger id="munki-deployment-package" className="w-full">
                <SelectValue placeholder={software.isLoading ? "Loading..." : "Select Package"} />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {packages.map((pkg) => (
                    <SelectItem key={pkg.id} value={String(pkg.id)}>
                      {pkg.version}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
            <FieldDescription>
              The chosen package is rendered only when it is still available for deployment.
            </FieldDescription>
            {showErrors && errors.package_id ? <FieldError>{errors.package_id}</FieldError> : null}
          </Field>

          <Field>
            <FieldLabel htmlFor="munki-deployment-intent" required>
              Intent
            </FieldLabel>
            <Select value={form.intent} onValueChange={(intent) => setForm({ ...form, intent: intent as Intent })}>
              <SelectTrigger id="munki-deployment-intent" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {intentOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
            <FieldDescription>
              Install and update writes managed_installs. Remove writes managed_uninstalls. Update if present writes
              managed_updates. Self Service writes optional_installs. Featured writes optional_installs and
              featured_items.
            </FieldDescription>
          </Field>

          <CheckboxField
            id="munki-deployment-all-hosts"
            label="All devices"
            description="Targets every known host unless an excluded label removes it."
            checked={form.all_hosts}
            onChange={(all_hosts) => setForm({ ...form, all_hosts })}
          />

          {form.all_hosts ? null : (
            <Field data-invalid={showErrors && errors.include_label_ids ? true : undefined}>
              <FieldLabel required>Target Labels</FieldLabel>
              <LabelPicker
                value={form.include_label_ids}
                selectionMode="multiple"
                includeBuiltins
                unavailableLabelIDs={form.exclude_label_ids}
                invalid={showErrors && errors.include_label_ids ? true : undefined}
                onChange={(include_label_ids) => setForm({ ...form, include_label_ids })}
              />
              <FieldDescription>When All devices is off, a host must match at least one target label.</FieldDescription>
              {showErrors && errors.include_label_ids ? <FieldError>{errors.include_label_ids}</FieldError> : null}
            </Field>
          )}

          <Field>
            <FieldLabel>Excluded Labels</FieldLabel>
            <LabelPicker
              value={form.exclude_label_ids}
              selectionMode="multiple"
              includeBuiltins
              unavailableLabelIDs={form.include_label_ids}
              onChange={(exclude_label_ids) => setForm({ ...form, exclude_label_ids })}
            />
            <FieldDescription>
              Matching hosts are removed from this deployment, even when they match All devices or a target label.
            </FieldDescription>
          </Field>

          <FormActions
            pending={create.isPending}
            cancelTo="/munki/software-titles/$softwareId"
            cancelParams={{ softwareId: String(softwareId ?? "") }}
          />
        </FieldGroup>
      </form>
    </PageShell>
  );
}

function useSoftwareIDParam() {
  const params = useParams({ strict: false });
  const id = Number(params.softwareId);
  return Number.isFinite(id) && id > 0 ? id : null;
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
  onChange,
}: {
  id: string;
  label: string;
  description?: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
}) {
  return (
    <Field orientation="horizontal">
      <Checkbox id={id} checked={checked} onCheckedChange={(value) => onChange(value === true)} />
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

function toggleArch(form: PackageFormState, arch: "arm64" | "x86_64", checked: boolean) {
  if (checked) return Array.from(new Set([...form.supported_architectures, arch]));
  return form.supported_architectures.filter((value) => value !== arch);
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
