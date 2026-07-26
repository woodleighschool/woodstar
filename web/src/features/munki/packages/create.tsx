import { getRouteApi, useNavigate } from "@tanstack/react-router";
import { useRef } from "react";

import { encodeSort } from "@components/data-table/use-data-table-search";
import { MAX_PAGE_SIZE } from "@lib/pagination";

import { useMunkiSoftware } from "../software/queries";
import { deleteUnclaimedMunkiInstaller } from "../upload";
import { PackageForm } from "./fields";
import { emptyPackageForm } from "./form-adapter";
import { useCreateMunkiPackage, useMunkiPackages } from "./queries";
import { useUploadMunkiInstaller } from "./queries";

const routeApi = getRouteApi("/_authenticated/munki/packages/new");

export function MunkiPackageCreatePage() {
  const navigate = useNavigate();
  const search = routeApi.useSearch();
  const initialSoftwareID = search.software_id ?? null;
  const create = useCreateMunkiPackage();
  const installerUpload = useUploadMunkiInstaller();
  const cancelled = useRef(false);
  const packageMutationAbort = useRef<AbortController | null>(null);
  const packages = useMunkiPackages({
    per_page: MAX_PAGE_SIZE,
    sort: encodeSort("software_name"),
  });
  const software = useMunkiSoftware({ per_page: MAX_PAGE_SIZE, sort: encodeSort("name") });
  const initial = emptyPackageForm(initialSoftwareID);
  const softwareRows = software.data?.items ?? [];

  return (
    <PackageForm
      initial={initial}
      title="Create Package"
      submitLabel="Create"
      softwareInfo={null}
      softwareOptions={softwareRows}
      softwareLoading={software.isLoading}
      packageOptions={packages.data?.items ?? []}
      installerMetadata={undefined}
      canCancelWhileSubmitting={installerUpload.isUploading}
      onSubmit={async ({ softwareID, installerFile, mutation }) => {
        cancelled.current = false;
        let installerObjectID: number | undefined;
        if (mutation.installer_type !== "nopkg") {
          if (!installerFile) throw new Error("Validated package is missing its installer.");
          installerObjectID = (await installerUpload.upload({ file: installerFile })).id;
          if (cancelled.current) {
            await deleteUnclaimedMunkiInstaller(installerObjectID).catch(() => undefined);
            return false;
          }
        }
        const abortController = new AbortController();
        packageMutationAbort.current = abortController;
        try {
          await create.mutateAsync({
            body: {
              software_id: softwareID,
              ...mutation,
              installer_object_id: installerObjectID,
            },
            signal: abortController.signal,
          });
        } catch (error) {
          if (installerObjectID !== undefined) {
            await deleteUnclaimedMunkiInstaller(installerObjectID).catch(() => undefined);
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
