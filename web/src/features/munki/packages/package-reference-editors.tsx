import { AppWindow, Package as PackageIcon, Trash2 } from "lucide-react";
import { useState } from "react";

import { InputGroupLoadingAddon } from "@components/input-group-loading-addon";
import { Link } from "@components/link";
import {
  Attachment,
  AttachmentContent,
  AttachmentDescription,
  AttachmentMedia,
  AttachmentTitle,
} from "@components/ui/attachment";
import { Button } from "@components/ui/button";
import {
  Combobox,
  ComboboxCollection,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxGroup,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@components/ui/combobox";
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from "@components/ui/field";
import { InputGroupAddon, InputGroupButton } from "@components/ui/input-group";
import { ValidatedFormField } from "@components/validated-form-field";
import { SoftwareArtwork } from "@features/software/software-icon";
import type { MunkiPackage, MunkiSoftware } from "@lib/api";

import type { PackageEditorForm } from "./fields";
import type { PackageReferenceRow } from "./form-schema";

export type SoftwareInfo = {
  id: number;
  name: string;
  iconUrl?: string;
};

export function ParentSoftwareField({ software }: { software: SoftwareInfo }) {
  return (
    <Field>
      <FieldLabel>Software</FieldLabel>
      <Attachment className="w-full">
        <AttachmentMedia className="overflow-visible rounded-none bg-transparent">
          <SoftwareArtwork src={software.iconUrl} size="md" />
        </AttachmentMedia>
        <AttachmentContent>
          <AttachmentTitle>{software.name}</AttachmentTitle>
          <AttachmentDescription>Parent software</AttachmentDescription>
        </AttachmentContent>
        <Link
          to="/munki/software/$id"
          params={{ id: String(software.id) }}
          className="absolute inset-0 z-10 outline-none"
        />
      </Attachment>
    </Field>
  );
}

export function SoftwareSelector({
  form,
  rows,
  loading,
}: {
  form: PackageEditorForm;
  rows: MunkiSoftware[];
  loading: boolean;
}) {
  return (
    <form.Field name="software_id">
      {(field) => {
        const selected = rows.find((item) => item.id === field.state.value) ?? null;
        return (
          <ValidatedFormField
            field={field}
            label="Software"
            htmlFor="munki-package-software"
            required
          >
            {(control) => (
              <SoftwareCombobox
                key={selected?.id ?? "none"}
                control={control}
                rows={rows}
                selected={selected}
                loading={loading}
                onBlur={field.handleBlur}
                onChange={field.handleChange}
              />
            )}
          </ValidatedFormField>
        );
      }}
    </form.Field>
  );
}

function SoftwareCombobox({
  control,
  rows,
  selected,
  loading,
  onBlur,
  onChange,
}: {
  control: { id: string | undefined; "aria-invalid": true | undefined };
  rows: MunkiSoftware[];
  selected: MunkiSoftware | null;
  loading: boolean;
  onBlur: () => void;
  onChange: (value: number | null) => void;
}) {
  const [inputValue, setInputValue] = useState(selected?.name ?? "");

  return (
    <Combobox
      items={rows}
      itemToStringLabel={(item) => item.name}
      itemToStringValue={(item) => String(item.id)}
      isItemEqualToValue={(item, selectedItem) => item.id === selectedItem.id}
      value={selected}
      inputValue={inputValue}
      onInputValueChange={(next, eventDetails) => {
        if (eventDetails.reason !== "item-press") {
          setInputValue(next);
        }
      }}
      onValueChange={(next) => {
        onChange(next?.id ?? null);
        setInputValue(next?.name ?? "");
      }}
    >
      <ComboboxInput
        {...control}
        id="munki-package-software"
        className="w-full"
        placeholder="Select Software"
        aria-busy={loading}
        showTrigger={!loading}
        onBlur={onBlur}
      >
        {loading ? <InputGroupLoadingAddon /> : null}
      </ComboboxInput>
      {loading ? null : (
        <ComboboxContent>
          <ComboboxEmpty>
            {rows.length === 0 ? "No software available." : "No software found."}
          </ComboboxEmpty>
          <ComboboxList>
            {(item) => (
              <ComboboxItem key={item.id} value={item}>
                {item.name}
              </ComboboxItem>
            )}
          </ComboboxList>
        </ComboboxContent>
      )}
    </Combobox>
  );
}

export function PackageReferenceEditor({
  legend,
  description,
  addLabel,
  rows,
  packageOptions,
  onAdd,
  onReplace,
  onRemove,
}: {
  legend: string;
  description?: string;
  addLabel: string;
  rows: PackageReferenceRow[];
  packageOptions: MunkiPackage[];
  onAdd: () => void;
  onReplace: (index: number, row: PackageReferenceRow) => void;
  onRemove: (index: number) => void;
}) {
  const packageGroups = packageReferenceGroups(packageOptions);

  return (
    <FieldSet className="gap-4">
      <FieldLegend variant="label">{legend}</FieldLegend>
      {description ? <FieldDescription>{description}</FieldDescription> : null}
      <FieldGroup className="gap-2">
        {rows.map((row, index) => (
          <PackageReferenceCombobox
            key={row.rowID}
            row={row}
            packageGroups={packageGroups}
            onChange={(next) => onReplace(index, next)}
            onRemove={() => onRemove(index)}
          />
        ))}
        <Button type="button" variant="outline" size="sm" className="w-fit" onClick={onAdd}>
          {addLabel}
        </Button>
      </FieldGroup>
    </FieldSet>
  );
}

function PackageReferenceCombobox({
  row,
  packageGroups,
  onChange,
  onRemove,
}: {
  row: PackageReferenceRow;
  packageGroups: ReturnType<typeof packageReferenceGroups>;
  onChange: (row: PackageReferenceRow) => void;
  onRemove: () => void;
}) {
  const [inputValue, setInputValue] = useState(packageReferenceInputValue(row, packageGroups));
  const selectedValue = row.package_id
    ? packageReferencePackageValue(row.package_id)
    : row.software_id
      ? packageReferenceSoftwareValue(row.software_id)
      : "";
  const optionGroups = packageReferenceOptionGroups(packageGroups);
  const selectedOption = optionGroups
    .flatMap((group) => group.items)
    .find((option) => option.value === selectedValue);

  return (
    <Combobox
      items={optionGroups}
      itemToStringLabel={(option) => option.label}
      itemToStringValue={(option) => option.value}
      isItemEqualToValue={(option, selected) => option.value === selected.value}
      value={selectedOption ?? null}
      inputValue={inputValue}
      onInputValueChange={(next, eventDetails) => {
        if (eventDetails.reason !== "item-press") {
          setInputValue(next);
        }
      }}
      onValueChange={(option) => {
        const selection = packageReferenceSelection(option?.value ?? null, packageGroups);
        if (!selection) return;
        onChange({ rowID: row.rowID, ...selection.reference });
        setInputValue(selection.label);
      }}
    >
      <ComboboxInput className="w-full" placeholder="Select software or a version">
        <InputGroupAddon align="inline-end">
          <InputGroupButton
            type="button"
            variant="ghost"
            size="icon-xs"
            onClick={(event) => {
              event.stopPropagation();
              onRemove();
            }}
          >
            <Trash2 />
          </InputGroupButton>
        </InputGroupAddon>
      </ComboboxInput>
      <ComboboxContent>
        <ComboboxEmpty>
          {packageGroups.length === 0 ? "No Packages Available." : "No Packages Found."}
        </ComboboxEmpty>
        <ComboboxList>
          {(group) => (
            <ComboboxGroup key={group.value} items={group.items}>
              <ComboboxCollection>
                {(option) => <PackageReferenceComboboxItem key={option.value} option={option} />}
              </ComboboxCollection>
            </ComboboxGroup>
          )}
        </ComboboxList>
      </ComboboxContent>
    </Combobox>
  );
}

interface PackageReferenceOption {
  value: string;
  label: string;
  kind: "software" | "package";
  softwareIconURL?: string;
  version?: string;
}

interface PackageReferenceOptionGroup {
  value: string;
  items: PackageReferenceOption[];
}

function packageReferenceOptionGroups(
  packageGroups: ReturnType<typeof packageReferenceGroups>,
): PackageReferenceOptionGroup[] {
  return packageGroups.map((group) => {
    const items: PackageReferenceOption[] = [
      {
        value: packageReferenceSoftwareValue(group.softwareID),
        label: group.softwareTitle,
        kind: "software",
        softwareIconURL: group.softwareIconURL,
      },
      ...group.packages.map((pkg) => ({
        value: packageReferencePackageValue(pkg.id),
        label: packageLabel(pkg),
        kind: "package" as const,
        version: pkg.version,
      })),
    ];

    return { value: String(group.softwareID), items };
  });
}

function PackageReferenceComboboxItem({ option }: { option: PackageReferenceOption }) {
  if (option.kind === "software") {
    return (
      <ComboboxItem className="py-2" value={option}>
        <SoftwareArtwork src={option.softwareIconURL} fallbackIcon={AppWindow} />
        <span className="flex min-w-0 flex-1 flex-col">
          <span className="truncate font-medium">{option.label}</span>
          <span className="text-xs text-muted-foreground">All versions</span>
        </span>
      </ComboboxItem>
    );
  }

  return (
    <ComboboxItem className="py-2 pl-8" value={option}>
      <PackageIcon />
      <span className="min-w-0 flex-1 truncate">Version {option.version}</span>
    </ComboboxItem>
  );
}

function packageReferencePackageValue(packageID: number) {
  return `package:${packageID}`;
}

function packageReferenceSoftwareValue(softwareID: number) {
  return `software:${softwareID}`;
}

function packageReferenceInputValue(
  row: Pick<PackageReferenceRow, "software_name" | "package_version" | "package_id">,
  packageGroups: ReturnType<typeof packageReferenceGroups>,
) {
  const selectedPackage = packageGroups
    .flatMap((group) => group.packages)
    .find((pkg) => pkg.id === row.package_id);
  if (selectedPackage) return packageLabel(selectedPackage);
  if (!row.software_name) return "";
  if (!row.package_version) return row.software_name;
  return `${row.software_name} ${row.package_version}`;
}

function packageReferenceSelection(
  value: string | null,
  packageGroups: ReturnType<typeof packageReferenceGroups>,
) {
  if (!value) return null;
  if (value.startsWith("software:")) {
    const softwareID = Number(value.slice("software:".length));
    const group = packageGroups.find((option) => option.softwareID === softwareID);
    if (!group) return null;
    return {
      label: group.softwareTitle,
      reference: {
        software_id: group.softwareID,
        software_name: group.softwareTitle,
        package_id: undefined,
        package_version: undefined,
      },
    };
  }
  if (!value.startsWith("package:")) return null;
  const packageID = Number(value.slice("package:".length));
  const pkg = packageGroups
    .flatMap((group) => group.packages)
    .find((option) => option.id === packageID);
  if (!pkg) return null;
  return {
    label: packageLabel(pkg),
    reference: {
      software_id: pkg.software.id,
      software_name: pkg.software.name,
      package_id: pkg.id,
      package_version: pkg.version,
    },
  };
}

function packageReferenceGroups(packages: MunkiPackage[]) {
  const groups = new Map<
    number,
    {
      softwareID: number;
      softwareTitle: string;
      softwareIconURL?: string;
      packages: MunkiPackage[];
    }
  >();
  for (const pkg of packages) {
    const group = groups.get(pkg.software.id) ?? {
      softwareID: pkg.software.id,
      softwareTitle: pkg.software.name,
      softwareIconURL: pkg.software.icon_url,
      packages: [],
    };
    group.packages.push(pkg);
    groups.set(pkg.software.id, group);
  }
  return [...groups.values()];
}

function packageLabel(pkg: MunkiPackage) {
  return `${pkg.software.name} — ${pkg.version}`;
}
