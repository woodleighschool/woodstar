import { encodeSort } from "@/hooks/use-data-table-search";
import { useNavigate, useSearch } from "@tanstack/react-router";
import { useState } from "react";

import { PageHeader, PageShell } from "@/components/layout/page-layout";
import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox";
import { Field, FieldError, FieldLabel } from "@/components/ui/field";
import { MAX_PAGE_SIZE } from "@/hooks/use-data-table-search";
import { useUploadMunkiArtifact } from "@/hooks/use-munki-artifacts";
import { useCreateMunkiPackage, useMunkiPackages } from "@/hooks/use-munki-packages";
import { useMunkiSoftware, type MunkiSoftware } from "@/hooks/use-munki-software";

import { usePackageEditorForm } from "./editor-form";
import { PackageEditorTabs, PackageFormActions, type SoftwareInfo } from "./fields";
import { emptyPackageForm, packageMutationFromForm, packageSubmitPreflightError } from "./form-state";

export function MunkiPackageCreatePage() {
  const navigate = useNavigate();
  const search = useSearch({ strict: false });
  const initialSoftwareID =
    typeof search.software_id === "number" && search.software_id > 0 ? search.software_id : null;
  const [softwareID, setSoftwareID] = useState<number | null>(initialSoftwareID);
  const create = useCreateMunkiPackage();
  const packageUpload = useUploadMunkiArtifact("package");
  const packages = useMunkiPackages({ per_page: MAX_PAGE_SIZE, sort: encodeSort("name") });
  const software = useMunkiSoftware({ per_page: MAX_PAGE_SIZE, sort: encodeSort("name") });
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
  const softwareInfo: SoftwareInfo | null = selectedSoftware
    ? {
        name: selectedSoftware.name,
        description: selectedSoftware.description,
        category: selectedSoftware.category,
        developer: selectedSoftware.developer,
      }
    : null;

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
                {(item: MunkiSoftware) => (
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
          softwareInfo={softwareInfo}
          packageOptions={packages.data?.items ?? []}
          installerFile={installerFile}
          uninstallerFile={uninstallerFile}
          installerArtifactLocation=""
          uninstallerArtifactLocation=""
          onInstallerFileChange={setInstallerFile}
          onUninstallerFileChange={setUninstallerFile}
        />
        <PackageFormActions
          pending={create.isPending || packageUpload.isUploading || softwareID === null}
          error={softwareError ? undefined : (preflightError ?? create.error?.message ?? packageUpload.error?.message)}
        />
      </form>
    </PageShell>
  );
}
