import { useNavigate, useSearch } from "@tanstack/react-router";
import { useRef } from "react";

import { encodeSort } from "@/hooks/use-data-table-search";
import { useCreateMunkiPackage, useMunkiPackages } from "@/hooks/use-munki-packages";
import { useMunkiSoftware } from "@/hooks/use-munki-software";
import { deleteUnclaimedMunkiInstaller, useUploadMunkiInstaller } from "@/hooks/use-munki-uploads";
import { MAX_PAGE_SIZE } from "@/lib/pagination";

import { usePackageEditorForm } from "./editor-form";
import { PackageForm } from "./fields";
import { emptyPackageForm, packageMutationFromForm } from "./form-state";

export function MunkiPackageCreatePage() {
  const navigate = useNavigate();
  const search = useSearch({ strict: false });
  const initialSoftwareID =
    typeof search.software_id === "number" && search.software_id > 0 ? search.software_id : null;
  const create = useCreateMunkiPackage();
  const installerUpload = useUploadMunkiInstaller();
  const cancelled = useRef(false);
  const packageMutationAbort = useRef<AbortController | null>(null);
  const packages = useMunkiPackages({
    per_page: MAX_PAGE_SIZE,
    sort: encodeSort("software_name"),
  });
  const software = useMunkiSoftware({ per_page: MAX_PAGE_SIZE, sort: encodeSort("name") });
  const form = usePackageEditorForm(
    emptyPackageForm(initialSoftwareID),
    async (value) => {
      cancelled.current = false;
      if (value.software_id === null) throw new Error("Validated package is missing software.");
      let installerObjectID: number | undefined;
      if (value.installer_type !== "nopkg") {
        if (!value.installer_file) throw new Error("Validated package is missing its installer.");
        installerObjectID = (await installerUpload.upload({ file: value.installer_file })).id;
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
            software_id: value.software_id,
            ...packageMutationFromForm(value, installerObjectID),
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
    },
    () => void navigate({ to: "/munki/packages" }),
  );
  const softwareRows = software.data?.items ?? [];

  return (
    <PackageForm
      form={form}
      title="New Package"
      submitLabel="Create"
      softwareInfo={null}
      softwareOptions={softwareRows}
      softwareLoading={software.isLoading}
      packageOptions={packages.data?.items ?? []}
      installerMetadata={undefined}
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
