import { useNavigate, useParams } from "@tanstack/react-router";
import { useMemo, useRef, useState } from "react";
import { toast } from "sonner";

import { munkiSoftwareIconURL } from "@/components/munki/munki-icon";
import { QueryGate } from "@/components/query-gate";
import { encodeSort } from "@/hooks/use-data-table-search";
import {
  useMunkiPackage,
  useMunkiPackages,
  useUpdateMunkiPackage,
} from "@/hooks/use-munki-packages";
import { deleteUnclaimedMunkiInstaller, useUploadMunkiInstaller } from "@/hooks/use-munki-uploads";
import type { MunkiPackage } from "@/lib/api";
import { MAX_PAGE_SIZE } from "@/lib/pagination";
import { parseRouteID } from "@/lib/route-params";

import { usePackageEditorForm } from "./editor-form";
import { PackageForm, type SoftwareInfo } from "./fields";
import { packageFormFromPackage, packageMutationFromForm } from "./form-state";

export function MunkiPackageEditPage() {
  const params = useParams({ strict: false });
  const validPackageID = parseRouteID(params.packageId);
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
  const cancelled = useRef(false);
  const packageMutationAbort = useRef<AbortController | null>(null);
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
    cancelled.current = false;
    if (value.installer_type !== "nopkg" && !installerFile && !pkg.installer_object_id) {
      toast.error("Select an installer file.");
      return;
    }
    let replacementObjectID: number | undefined;
    if (value.installer_type !== "nopkg" && installerFile) {
      replacementObjectID = (await installerUpload.upload({ file: installerFile })).id;
      if (cancelled.current) {
        await deleteUnclaimedMunkiInstaller(replacementObjectID).catch(() => undefined);
        return;
      }
    }
    const installerObjectID =
      value.installer_type === "nopkg"
        ? undefined
        : (replacementObjectID ?? pkg.installer_object_id);
    const abortController = new AbortController();
    packageMutationAbort.current = abortController;
    try {
      await update.mutateAsync({
        id: packageID,
        body: packageMutationFromForm(value, installerObjectID),
        signal: abortController.signal,
      });
    } catch (error) {
      if (replacementObjectID !== undefined) {
        await deleteUnclaimedMunkiInstaller(replacementObjectID).catch(() => undefined);
      }
      throw error;
    } finally {
      if (packageMutationAbort.current === abortController) {
        packageMutationAbort.current = null;
      }
    }
    void navigate({ to: "/munki/packages" });
  });

  return (
    <PackageForm
      form={form}
      title={`${pkg.software_name} ${pkg.version}`}
      submitLabel="Save"
      softwareInfo={softwareInfo}
      packageOptions={(packages.data?.items ?? []).filter((item) => item.id !== packageID)}
      installerFile={installerFile}
      installerMetadata={pkg.installer_file}
      onInstallerFileChange={setInstallerFile}
      canCancelWhileSubmitting={installerUpload.isUploading}
      onCancel={() => {
        cancelled.current = true;
        installerUpload.cancel();
        packageMutationAbort.current?.abort();
        void navigate({ to: "/munki/packages" });
      }}
    />
  );
}
