import { useNavigate } from "@tanstack/react-router";
import { Loader2 } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { EditableMunkiIcon } from "@/components/munki/editable-munki-icon";
import { PackageEditorTabs, PackageFormActions } from "@/components/munki/software-title/package-editor-fields";
import { MutationError } from "@/components/mutation-error";
import { useUploadMunkiArtifact } from "@/hooks/munki/artifacts";
import { usePackageEditorForm } from "@/hooks/munki/package-editor-form";
import {
  useCreateMunkiPackage,
  useMunkiPackage,
  useMunkiPackages,
  useUpdateMunkiPackage,
  type MunkiPackage,
} from "@/hooks/munki/packages";
import { useMunkiSoftwareTitle, useMunkiSoftwareTitles } from "@/hooks/munki/software-titles";
import { uniqueOptions } from "@/lib/form-validation";
import { emptyPackageForm, packageFormFromPackage, packageMutationFromForm } from "@/lib/munki-package-form";
import { MAX_PAGE_SIZE } from "@/lib/pagination";

import { usePackageIDParam, useSoftwareIDParam } from "./route-params";

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
