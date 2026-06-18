import { encodeSort, MAX_PAGE_SIZE } from "@/hooks/use-data-table-search";
import { useNavigate, useParams } from "@tanstack/react-router";
import { useMemo, useState } from "react";
import { toast } from "sonner";

import { FormActions } from "@/components/form-actions";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { munkiSoftwareIconURL } from "@/components/munki/munki-icon";
import { QueryGate } from "@/components/query-gate";
import { useDeleteMunkiInstaller, useUploadMunkiInstaller } from "@/hooks/use-munki-uploads";
import {
  type MunkiPackage,
  useMunkiPackage,
  useMunkiPackages,
  useUpdateMunkiPackage,
} from "@/hooks/use-munki-packages";

import { usePackageEditorForm } from "./editor-form";
import { PackageEditorTabs, type SoftwareInfo } from "./fields";
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
      <QueryGate title="Failed to load package" error={{ message: "Package route is invalid." }} />
    );
  }

  if (pkg.error || !pkg.data) {
    return (
      <QueryGate
        title="Failed to load package"
        error={pkg.error}
        onRetry={() => void pkg.refetch()}
      />
    );
  }

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
  const installerUpload = useUploadMunkiInstaller();
  const installerDelete = useDeleteMunkiInstaller();
  const packages = useMunkiPackages({ per_page: MAX_PAGE_SIZE, sort: encodeSort("name") });
  const [installerFile, setInstallerFile] = useState<File | null>(null);
  const initial = useMemo(() => packageFormFromPackage(pkg), [pkg]);
  const softwareInfo: SoftwareInfo = {
    id: pkg.software_id,
    name: pkg.software_name,
    description: pkg.software_description,
    category: pkg.software_category,
    developer: pkg.software_developer,
    iconUrl: munkiSoftwareIconURL(pkg.software_id),
  };
  const form = usePackageEditorForm(initial, async (value) => {
    const validationError = packageSubmitPreflightError(value, {
      hasInstallerFile: !!installerFile || !!pkg.installer_object_id,
    });
    if (validationError) {
      toast.error(validationError);
      return;
    }
    await update.mutateAsync({ id: packageID, body: packageMutationFromForm(value) });
    if (value.installer_type !== "nopkg" && installerFile) {
      await installerUpload.upload({ packageId: packageID, file: installerFile });
    }
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
          installerMetadata={pkg.installer_file}
          hasInstallerObject={pkg.installer_object_id != null}
          onInstallerFileChange={setInstallerFile}
          onDeleteInstaller={async () => {
            await installerDelete.mutateAsync(packageID);
            setInstallerFile(null);
            toast.success("Installer deleted");
          }}
          deletingInstaller={installerDelete.isPending}
        />
        <FormActions
          form={form}
          submitLabel="Save"
          onCancel={() => void navigate({ to: "/munki/packages" })}
        />
      </form>
    </PageShell>
  );
}
