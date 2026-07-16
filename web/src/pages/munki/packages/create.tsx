import { useNavigate, useSearch } from "@tanstack/react-router";
import { useRef, useState } from "react";
import { toast } from "sonner";

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
import { encodeSort } from "@/hooks/use-data-table-search";
import { useCreateMunkiPackage, useMunkiPackages } from "@/hooks/use-munki-packages";
import { useMunkiSoftware } from "@/hooks/use-munki-software";
import { deleteUnclaimedMunkiInstaller, useUploadMunkiInstaller } from "@/hooks/use-munki-uploads";
import type { MunkiSoftware } from "@/lib/api";
import { MAX_PAGE_SIZE } from "@/lib/pagination";

import { usePackageEditorForm } from "./editor-form";
import { PackageForm, type SoftwareInfo } from "./fields";
import { emptyPackageForm, packageMutationFromForm } from "./form-state";

export function MunkiPackageCreatePage() {
  const navigate = useNavigate();
  const search = useSearch({ strict: false });
  const initialSoftwareID =
    typeof search.software_id === "number" && search.software_id > 0 ? search.software_id : null;
  const [softwareID, setSoftwareID] = useState<number | null>(initialSoftwareID);
  const create = useCreateMunkiPackage();
  const installerUpload = useUploadMunkiInstaller();
  const cancelled = useRef(false);
  const packageMutationAbort = useRef<AbortController | null>(null);
  const packages = useMunkiPackages({ per_page: MAX_PAGE_SIZE, sort: encodeSort("name") });
  const software = useMunkiSoftware({ per_page: MAX_PAGE_SIZE, sort: encodeSort("name") });
  const [installerFile, setInstallerFile] = useState<File | null>(null);
  const form = usePackageEditorForm(emptyPackageForm(), async (value) => {
    cancelled.current = false;
    if (softwareID === null) {
      toast.error("Pick software.");
      return;
    }
    let installerObjectID: number | undefined;
    if (value.installer_type !== "nopkg") {
      if (!installerFile) {
        toast.error("Select an installer file.");
        return;
      }
      installerObjectID = (await installerUpload.upload({ file: installerFile })).id;
      if (cancelled.current) {
        await deleteUnclaimedMunkiInstaller(installerObjectID).catch(() => undefined);
        return;
      }
    }
    const abortController = new AbortController();
    packageMutationAbort.current = abortController;
    try {
      await create.mutateAsync({
        body: {
          software_id: softwareID,
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
    void navigate({ to: "/munki/packages" });
  });
  const softwareRows = software.data?.items ?? [];
  const selectedSoftware = softwareRows.find((item) => item.id === softwareID) ?? null;
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
    <SoftwareSelector
      key={selectedSoftware?.id ?? "none"}
      rows={softwareRows}
      selected={selectedSoftware}
      loading={software.isLoading}
      onChange={setSoftwareID}
    />
  );

  return (
    <PackageForm
      form={form}
      title="New Package"
      submitLabel="Create"
      softwareInfo={softwareInfo}
      softwareSelector={softwareSelector}
      packageOptions={packages.data?.items ?? []}
      installerFile={installerFile}
      installerMetadata={undefined}
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

function SoftwareSelector({
  rows,
  selected,
  loading,
  onChange,
}: {
  rows: MunkiSoftware[];
  selected: MunkiSoftware | null;
  loading: boolean;
  onChange: (id: number | null) => void;
}) {
  const [inputValue, setInputValue] = useState(selected?.name ?? "");

  return (
    <Field>
      <FieldLabel htmlFor="munki-package-software" required>
        Software
      </FieldLabel>
      <Combobox
        value={selected ? String(selected.id) : ""}
        inputValue={inputValue}
        onInputValueChange={setInputValue}
        onValueChange={(next) => {
          const item = rows.find((candidate) => String(candidate.id) === next);
          onChange(item?.id ?? null);
          setInputValue(item?.name ?? "");
        }}
      >
        <ComboboxAnchor className="w-full">
          <ComboboxInput
            id="munki-package-software"
            placeholder={loading ? "Loading Software..." : "Select Software"}
            required
          />
          <ComboboxTrigger aria-label="Open software" />
        </ComboboxAnchor>
        <ComboboxContent>
          <ComboboxEmpty>
            {rows.length === 0 ? "No Software Available." : "No Software Found."}
          </ComboboxEmpty>
          {rows.map((item) => (
            <ComboboxItem key={item.id} value={String(item.id)} label={item.name}>
              {item.name}
            </ComboboxItem>
          ))}
        </ComboboxContent>
      </Combobox>
    </Field>
  );
}
