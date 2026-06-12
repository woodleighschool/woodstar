import { encodeSort, MAX_PAGE_SIZE } from "@/hooks/use-data-table-search";
import { useNavigate, useParams } from "@tanstack/react-router";
import { useMemo, useState } from "react";

import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { QueryError } from "@/components/query-error";
import { useUploadMunkiArtifact } from "@/hooks/use-munki-artifacts";
import {
  type MunkiPackage,
  useMunkiPackage,
  useMunkiPackages,
  useUpdateMunkiPackage,
} from "@/hooks/use-munki-packages";

import { usePackageEditorForm } from "./editor-form";
import { PackageEditorTabs, PackageFormActions, type SoftwareInfo } from "./fields";
import {
  packageFormFromPackage,
  packageMutationFromForm,
  packageSubmitPreflightError,
} from "./form-state";

export function MunkiPackageEditPage() {
  const params = useParams({ strict: false });
  const packageID = Number(params.packageId);
  const validPackageID = Number.isFinite(packageID) && packageID > 0 ? packageID : null;
  const pkg = useMunkiPackage(validPackageID);

  if (validPackageID === null) {
    return (
      <PageShell>
        <QueryError
          title="Failed to load package"
          error={{ message: "Package route is invalid." }}
        />
      </PageShell>
    );
  }

  if (pkg.error) {
    return (
      <PageShell>
        <QueryError
          title="Failed to load package"
          error={pkg.error}
          onRetry={() => void pkg.refetch()}
        />
      </PageShell>
    );
  }

  if (!pkg.data) return null;

  return (
    <MunkiPackageEditForm
      key={`${pkg.data.id}:${pkg.data.updated_at}`}
      packageID={validPackageID}
      pkg={pkg.data}
    />
  );
}

function MunkiPackageEditForm({ packageID, pkg }: { packageID: number; pkg: MunkiPackage }) {
  const navigate = useNavigate();
  const update = useUpdateMunkiPackage();
  const packageUpload = useUploadMunkiArtifact("package");
  const packages = useMunkiPackages({ per_page: MAX_PAGE_SIZE, sort: encodeSort("name") });
  const [installerFile, setInstallerFile] = useState<File | null>(null);
  const [uninstallerFile, setUninstallerFile] = useState<File | null>(null);
  const [preflightError, setPreflightError] = useState<string | undefined>();
  const initial = useMemo(() => packageFormFromPackage(pkg), [pkg]);
  const softwareInfo: SoftwareInfo = {
    name: pkg.software_name,
    description: pkg.software_description,
    category: pkg.software_category,
    developer: pkg.software_developer,
  };
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
      value.installer_type !== "nopkg" && installerFile
        ? await packageUpload.upload(installerFile)
        : null;
    const uninstallerArtifact =
      value.uninstall_method === "uninstall_package" && uninstallerFile
        ? await packageUpload.upload(uninstallerFile)
        : null;
    const body = packageMutationFromForm(value, {
      installerArtifactID: installerArtifact?.id ?? pkg.installer_artifact_id,
      uninstallerArtifactID: uninstallerArtifact?.id ?? pkg.uninstaller_artifact_id,
    });
    await update.mutateAsync({ id: packageID, body });
    void navigate({ to: "/munki/packages" });
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
        <PageHeader title="Edit Package" />
        <PackageEditorTabs
          form={form}
          softwareInfo={softwareInfo}
          packageOptions={(packages.data?.items ?? []).filter((item) => item.id !== packageID)}
          installerFile={installerFile}
          uninstallerFile={uninstallerFile}
          installerArtifactLocation={pkg.installer_artifact_location ?? ""}
          uninstallerArtifactLocation={pkg.uninstaller_artifact_location ?? ""}
          onInstallerFileChange={setInstallerFile}
          onUninstallerFileChange={setUninstallerFile}
        />
        <PackageFormActions
          pending={update.isPending || packageUpload.isUploading}
          error={preflightError ?? update.error?.message ?? packageUpload.error?.message}
        />
      </form>
    </PageShell>
  );
}
