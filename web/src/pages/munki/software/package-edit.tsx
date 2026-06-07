import { useNavigate } from "@tanstack/react-router";
import { Loader2 } from "lucide-react";
import { useMemo, useState } from "react";

import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { EditableMunkiIcon } from "@/components/munki/editable-munki-icon";
import { PackageEditorTabs, PackageFormActions } from "@/components/munki/software/package-editor-fields";
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
import { useMunkiSoftwareDetail } from "@/hooks/munki/software";
import {
  emptyPackageForm,
  packageFormFromPackage,
  packageMutationFromForm,
  packageSubmitPreflightError,
} from "@/lib/munki-package-form";
import { MAX_PAGE_SIZE } from "@/lib/pagination";

import { usePackageIDParam, useSoftwareIDParam } from "./route-params";

export function MunkiPackageNewPage() {
  const navigate = useNavigate();
  const softwareID = useSoftwareIDParam();
  const software = useMunkiSoftwareDetail(softwareID);
  const create = useCreateMunkiPackage();
  const packageUpload = useUploadMunkiArtifact("package");
  const iconUpload = useUploadMunkiArtifact("icon");
  const packages = useMunkiPackages({ page_size: MAX_PAGE_SIZE, sort: "name.asc" });
  const [installerFile, setInstallerFile] = useState<File | null>(null);
  const [uninstallerFile, setUninstallerFile] = useState<File | null>(null);
  const [iconFile, setIconFile] = useState<File | null>(null);
  const [preflightError, setPreflightError] = useState<string | undefined>();
  const form = usePackageEditorForm(emptyPackageForm(), async (value) => {
    if (softwareID === null) return;
    setPreflightError(undefined);
    const validationError = packageSubmitPreflightError(value, {
      hasInstallerArtifact: !!installerFile,
      hasUninstallerArtifact: !!uninstallerFile,
    });
    if (validationError) {
      setPreflightError(validationError);
      return;
    }
    const installerArtifact =
      value.installer_type !== "nopkg" && installerFile ? await packageUpload.upload(installerFile) : null;
    const uninstallerArtifact =
      value.uninstall_method === "uninstall_package" && uninstallerFile
        ? await packageUpload.upload(uninstallerFile)
        : null;
    const iconArtifact = iconFile ? await iconUpload.upload(iconFile) : null;
    await create.mutateAsync(
      packageMutationFromForm(value, softwareID, {
        installerArtifactID: installerArtifact?.id,
        uninstallerArtifactID: uninstallerArtifact?.id,
        iconArtifactID: iconArtifact?.id,
      }),
    );
    void navigate({ to: "/munki/software/$softwareId", params: { softwareId: String(softwareID) } });
  });
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
            preflightError ??
            create.error?.message ??
            packageUpload.error?.message ??
            iconUpload.error?.message ??
            software.error?.message
          }
        />
        <PackageEditorTabs
          form={form}
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
  const software = useMunkiSoftwareDetail(softwareID);
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
  const packages = useMunkiPackages({ page_size: MAX_PAGE_SIZE, sort: "name.asc" });
  const [installerFile, setInstallerFile] = useState<File | null>(null);
  const [uninstallerFile, setUninstallerFile] = useState<File | null>(null);
  const [iconFile, setIconFile] = useState<File | null>(null);
  const [iconCleared, setIconCleared] = useState(false);
  const [preflightError, setPreflightError] = useState<string | undefined>();
  const initial = useMemo(() => packageFormFromPackage(pkg), [pkg]);
  const form = usePackageEditorForm(initial, async (value) => {
    setPreflightError(undefined);
    const validationError = packageSubmitPreflightError(value, {
      hasInstallerArtifact: !!installerFile || !!pkg.installer_artifact_id,
      hasUninstallerArtifact: !!uninstallerFile || !!pkg.uninstaller_artifact_id,
    });
    if (validationError) {
      setPreflightError(validationError);
      return;
    }
    const installerArtifact =
      value.installer_type !== "nopkg" && installerFile ? await packageUpload.upload(installerFile) : null;
    const uninstallerArtifact =
      value.uninstall_method === "uninstall_package" && uninstallerFile
        ? await packageUpload.upload(uninstallerFile)
        : null;
    const iconArtifact = iconFile ? await iconUpload.upload(iconFile) : null;
    const body = packageMutationFromForm(value, softwareID, {
      installerArtifactID: installerArtifact?.id ?? pkg.installer_artifact_id,
      uninstallerArtifactID: uninstallerArtifact?.id ?? pkg.uninstaller_artifact_id,
      iconArtifactID: iconArtifact?.id ?? (iconCleared ? undefined : pkg.icon_artifact_id),
    });
    await update.mutateAsync({ id: packageID, body });
    void navigate({ to: "/munki/software/$softwareId", params: { softwareId: String(softwareID) } });
  });
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
          message={
            preflightError ??
            update.error?.message ??
            packageUpload.error?.message ??
            iconUpload.error?.message ??
            softwareError
          }
        />
        <PackageEditorTabs
          form={form}
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
