import { useNavigate, useParams, useSearch } from "@tanstack/react-router";
import { Loader2 } from "lucide-react";
import { useMemo, useState } from "react";

import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { PackageEditorTabs, PackageFormActions } from "@/components/munki/packages/package-editor-fields";
import { MutationError } from "@/components/mutation-error";
import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox";
import { Field, FieldError, FieldLabel } from "@/components/ui/field";
import { useUploadMunkiArtifact } from "@/hooks/munki/artifacts";
import { usePackageEditorForm } from "@/hooks/munki/package-editor-form";
import {
  useCreateMunkiPackage,
  useMunkiPackage,
  useMunkiPackages,
  useUpdateMunkiPackage,
  type MunkiPackage,
} from "@/hooks/munki/packages";
import { useMunkiSoftware } from "@/hooks/munki/software";
import {
  emptyPackageForm,
  packageFormFromPackage,
  packageMutationFromForm,
  packageSubmitPreflightError,
} from "@/lib/munki-package-form";
import { MAX_PAGE_SIZE } from "@/lib/pagination";

export function MunkiPackageNewPage() {
  const navigate = useNavigate();
  const search = useSearch({ strict: false });
  const initialSoftwareID =
    typeof search.software_id === "number" && search.software_id > 0 ? search.software_id : null;
  const [softwareID, setSoftwareID] = useState<number | null>(initialSoftwareID);
  const create = useCreateMunkiPackage();
  const packageUpload = useUploadMunkiArtifact("package");
  const packages = useMunkiPackages({ page_size: MAX_PAGE_SIZE, sort: "name.asc" });
  const software = useMunkiSoftware({ page_size: MAX_PAGE_SIZE, sort: "name.asc" });
  const [installerFile, setInstallerFile] = useState<File | null>(null);
  const [uninstallerFile, setUninstallerFile] = useState<File | null>(null);
  const [preflightError, setPreflightError] = useState<string | undefined>();
  const form = usePackageEditorForm(emptyPackageForm(), async (value) => {
    if (softwareID === null) {
      setPreflightError("Pick software.");
      return;
    }
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
    await create.mutateAsync({
      software_id: softwareID,
      ...packageMutationFromForm(value, {
        installerArtifactID: installerArtifact?.id,
        uninstallerArtifactID: uninstallerArtifact?.id,
      }),
    });
    void navigate({ to: "/munki/packages" });
  });
  const softwareRows = software.data?.items ?? [];
  const selectedSoftware = softwareRows.find((item) => item.id === softwareID) ?? null;
  const softwareError = preflightError === "Pick software." ? preflightError : undefined;

  return (
    <PageShell asChild>
      <form
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit();
        }}
      >
        <PageHeader title="New Package" />
        <MutationError
          title="Failed to Create Package"
          message={
            softwareError ? undefined : (preflightError ?? create.error?.message ?? packageUpload.error?.message)
          }
        />
        <Field data-invalid={softwareError ? true : undefined} className="max-w-xl">
          <FieldLabel htmlFor="munki-package-software" required>
            Software
          </FieldLabel>
          <Combobox
            items={softwareRows}
            value={selectedSoftware}
            itemToStringLabel={(item) => item.name}
            itemToStringValue={(item) => String(item.id)}
            onValueChange={(next) => setSoftwareID(next?.id ?? null)}
          >
            <ComboboxInput
              id="munki-package-software"
              className="w-full"
              placeholder={software.isLoading ? "Loading Software..." : "Select Software"}
              required
              aria-invalid={softwareError ? true : undefined}
            />
            <ComboboxContent>
              <ComboboxEmpty>
                {softwareRows.length === 0 ? "No Software Available." : "No Software Found."}
              </ComboboxEmpty>
              <ComboboxList>
                {(item) => (
                  <ComboboxItem key={item.id} value={item}>
                    {item.name}
                  </ComboboxItem>
                )}
              </ComboboxList>
            </ComboboxContent>
          </Combobox>
          {softwareError ? <FieldError>{softwareError}</FieldError> : null}
        </Field>
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
        <PackageFormActions pending={create.isPending || packageUpload.isUploading || softwareID === null} />
      </form>
    </PageShell>
  );
}

export function MunkiPackageEditPage() {
  const params = useParams({ strict: false });
  const packageID = Number(params.packageId);
  const validPackageID = Number.isFinite(packageID) && packageID > 0 ? packageID : null;
  const pkg = useMunkiPackage(validPackageID);

  if (validPackageID === null) {
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
    <MunkiPackageEditForm key={`${pkg.data.id}:${pkg.data.updated_at}`} packageID={validPackageID} pkg={pkg.data} />
  );
}

function MunkiPackageEditForm({ packageID, pkg }: { packageID: number; pkg: MunkiPackage }) {
  const navigate = useNavigate();
  const update = useUpdateMunkiPackage();
  const packageUpload = useUploadMunkiArtifact("package");
  const packages = useMunkiPackages({ page_size: MAX_PAGE_SIZE, sort: "name.asc" });
  const [installerFile, setInstallerFile] = useState<File | null>(null);
  const [uninstallerFile, setUninstallerFile] = useState<File | null>(null);
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
        <MutationError
          title="Failed to Update Package"
          message={preflightError ?? update.error?.message ?? packageUpload.error?.message}
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
        <PackageFormActions pending={update.isPending || packageUpload.isUploading} />
      </form>
    </PageShell>
  );
}
