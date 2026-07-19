import { useNavigate, useParams } from "@tanstack/react-router";
import { useMemo, useRef } from "react";

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

import { PackageForm, packageFormFromPackage, type SoftwareInfo } from "./fields";

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
  const packages = useMunkiPackages({
    per_page: MAX_PAGE_SIZE,
    sort: encodeSort("software_name"),
  });
  const initial = useMemo(() => packageFormFromPackage(pkg), [pkg]);
  const softwareInfo: SoftwareInfo = {
    id: pkg.software.id,
    name: pkg.software.name,
    iconUrl: pkg.software.icon_url,
  };
  return (
    <PackageForm
      initial={initial}
      title={`${pkg.software.name} ${pkg.version}`}
      submitLabel="Save"
      softwareInfo={softwareInfo}
      packageOptions={(packages.data?.items ?? []).filter((item) => item.id !== packageID)}
      installerMetadata={pkg.installer_file}
      canCancelWhileSubmitting={installerUpload.isUploading}
      onSubmit={async ({ installerFile, mutation }) => {
        cancelled.current = false;
        let replacementObjectID: number | undefined;
        if (mutation.installer_type !== "nopkg" && installerFile) {
          replacementObjectID = (await installerUpload.upload({ file: installerFile })).id;
          if (cancelled.current) {
            await deleteUnclaimedMunkiInstaller(replacementObjectID).catch(() => undefined);
            return false;
          }
        }
        const installerObjectID =
          mutation.installer_type === "nopkg"
            ? undefined
            : (replacementObjectID ?? mutation.installer_object_id);
        const abortController = new AbortController();
        packageMutationAbort.current = abortController;
        try {
          await update.mutateAsync({
            id: packageID,
            body: { ...mutation, installer_object_id: installerObjectID },
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
        return true;
      }}
      onSuccess={() => void navigate({ to: "/munki/packages" })}
      onCancel={() => {
        cancelled.current = true;
        installerUpload.cancel();
        packageMutationAbort.current?.abort();
        void navigate({ to: "/munki/packages" });
      }}
    />
  );
}
