import { encodeSort, MAX_PAGE_SIZE } from "@/hooks/use-data-table-search";
import { useNavigate, useSearch } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { toast } from "sonner";

import { FormActions } from "@/components/form-actions";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import {
  Combobox,
  ComboboxAnchor,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxTrigger,
} from "@/components/ui/combobox";
import { Field, FieldLabel } from "@/components/ui/field";
import { useUploadMunkiInstaller, useUploadMunkiUninstaller } from "@/hooks/use-munki-uploads";
import { useCreateMunkiPackage, useMunkiPackages } from "@/hooks/use-munki-packages";
import { type MunkiSoftware, useMunkiSoftware } from "@/hooks/use-munki-software";

import { usePackageEditorForm } from "./editor-form";
import { PackageEditorTabs, type SoftwareInfo } from "./fields";
import {
  emptyPackageForm,
  packageMutationFromForm,
  packageSubmitPreflightError,
} from "./form-state";

export function MunkiPackageCreatePage() {
  const navigate = useNavigate();
  const search = useSearch({ strict: false });
  const initialSoftwareID =
    typeof search.software_id === "number" && search.software_id > 0 ? search.software_id : null;
  const [softwareID, setSoftwareID] = useState<number | null>(initialSoftwareID);
  const create = useCreateMunkiPackage();
  const installerUpload = useUploadMunkiInstaller();
  const uninstallerUpload = useUploadMunkiUninstaller();
  const packages = useMunkiPackages({ per_page: MAX_PAGE_SIZE, sort: encodeSort("name") });
  const software = useMunkiSoftware({ per_page: MAX_PAGE_SIZE, sort: encodeSort("name") });
  const [installerFile, setInstallerFile] = useState<File | null>(null);
  const [uninstallerFile, setUninstallerFile] = useState<File | null>(null);
  const form = usePackageEditorForm(emptyPackageForm(), async (value) => {
    if (softwareID === null) {
      toast.error("Pick software.");
      return;
    }
    const validationError = packageSubmitPreflightError(value, {
      hasInstallerFile: !!installerFile,
      hasUninstallerFile: !!uninstallerFile,
    });
    if (validationError) {
      toast.error(validationError);
      return;
    }
    const pkg = await create.mutateAsync({
      software_id: softwareID,
      ...packageMutationFromForm(value),
    });
    if (value.installer_type !== "nopkg" && installerFile) {
      await installerUpload.upload({ packageId: pkg.id, file: installerFile });
    }
    if (value.uninstall_method === "uninstall_package" && uninstallerFile) {
      await uninstallerUpload.upload({ packageId: pkg.id, file: uninstallerFile });
    }
    void navigate({ to: "/munki/packages" });
  });
  const softwareRows = software.data?.items ?? [];
  const selectedSoftware = softwareRows.find((item) => item.id === softwareID) ?? null;
  const [softwareInputValue, setSoftwareInputValue] = useState(selectedSoftware?.name ?? "");

  useEffect(() => {
    setSoftwareInputValue(selectedSoftware?.name ?? "");
  }, [selectedSoftware?.id, selectedSoftware?.name]);

  const softwareInfo: SoftwareInfo | null = selectedSoftware
    ? {
        id: selectedSoftware.id,
        name: selectedSoftware.name,
        description: selectedSoftware.description,
        category: selectedSoftware.category,
        developer: selectedSoftware.developer,
        iconUrl: selectedSoftware.icon_url,
      }
    : null;
  const softwareSelector = (
    <Field>
      <FieldLabel htmlFor="munki-package-software" required>
        Software
      </FieldLabel>
      <Combobox
        value={selectedSoftware ? String(selectedSoftware.id) : ""}
        inputValue={softwareInputValue}
        onInputValueChange={setSoftwareInputValue}
        onValueChange={(next) => {
          const item = softwareRows.find((candidate) => String(candidate.id) === next);
          setSoftwareID(item?.id ?? null);
          setSoftwareInputValue(item?.name ?? "");
        }}
      >
        <ComboboxAnchor className="w-full">
          <ComboboxInput
            id="munki-package-software"
            placeholder={software.isLoading ? "Loading Software..." : "Select Software"}
            required
          />
          <ComboboxTrigger aria-label="Open software" />
        </ComboboxAnchor>
        <ComboboxContent>
          <ComboboxEmpty>
            {softwareRows.length === 0 ? "No Software Available." : "No Software Found."}
          </ComboboxEmpty>
          {softwareRows.map((item: MunkiSoftware) => (
            <ComboboxItem key={item.id} value={String(item.id)} label={item.name}>
              {item.name}
            </ComboboxItem>
          ))}
        </ComboboxContent>
      </Combobox>
    </Field>
  );

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
        <PackageEditorTabs
          form={form}
          softwareInfo={softwareInfo}
          softwareSelector={softwareSelector}
          packageOptions={packages.data?.items ?? []}
          installerFile={installerFile}
          uninstallerFile={uninstallerFile}
          hasInstallerObject={false}
          hasUninstallerObject={false}
          onInstallerFileChange={setInstallerFile}
          onUninstallerFileChange={setUninstallerFile}
        />
        <FormActions
          form={form}
          submitLabel="Create"
          onCancel={() => void navigate({ to: "/munki/packages" })}
        />
      </form>
    </PageShell>
  );
}
