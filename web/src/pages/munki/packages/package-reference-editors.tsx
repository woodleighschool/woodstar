import { Link } from "@tanstack/react-router";
import { Trash2 } from "lucide-react";
import { useState } from "react";

import { FormField } from "@/components/form-field";
import { SoftwareArtwork } from "@/components/software/software-icon";
import {
  Attachment,
  AttachmentContent,
  AttachmentDescription,
  AttachmentMedia,
  AttachmentTitle,
} from "@/components/ui/attachment";
import { Button } from "@/components/ui/button";
import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxGroup,
  ComboboxInput,
  ComboboxItem,
  ComboboxLabel,
  ComboboxList,
} from "@/components/ui/combobox";
import { Field, FieldGroup, FieldLabel, FieldLegend, FieldSet } from "@/components/ui/field";
import type { MunkiPackage, MunkiSoftware } from "@/lib/api";

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
          to="/munki/software/$softwareId"
          params={{ softwareId: String(software.id) }}
          aria-label={`Open ${software.name}`}
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
          <FormField field={field} label="Software" htmlFor="munki-package-software" required>
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
          </FormField>
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
      items={rows.map((item) => String(item.id))}
      value={selected ? String(selected.id) : null}
      inputValue={inputValue}
      onInputValueChange={setInputValue}
      onValueChange={(next) => {
        const item = rows.find((candidate) => String(candidate.id) === next);
        onChange(item?.id ?? null);
        setInputValue(item?.name ?? "");
      }}
    >
      <ComboboxInput
        {...control}
        id="munki-package-software"
        className="w-full"
        placeholder={loading ? "Loading software..." : "Select software"}
        onBlur={onBlur}
      />
      <ComboboxContent>
        <ComboboxEmpty>
          {rows.length === 0 ? "No software available." : "No software found."}
        </ComboboxEmpty>
        <ComboboxList>
          {rows.map((item) => (
            <ComboboxItem key={item.id} value={String(item.id)}>
              {item.name}
            </ComboboxItem>
          ))}
        </ComboboxList>
      </ComboboxContent>
    </Combobox>
  );
}

export function PackageReferenceEditor({
  legend,
  addLabel,
  rows,
  packageOptions,
  onAdd,
  onReplace,
  onRemove,
}: {
  legend: string;
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

  return (
    <Combobox
      items={packageGroups.flatMap((group) => [
        packageReferenceSoftwareValue(group.softwareID),
        ...group.packages.map((pkg) => packageReferencePackageValue(pkg.id)),
      ])}
      value={selectedValue || null}
      inputValue={inputValue}
      onInputValueChange={setInputValue}
      onValueChange={(value) => {
        const selection = packageReferenceSelection(value, packageGroups);
        if (!selection) return;
        onChange({ rowID: row.rowID, ...selection.reference });
        setInputValue(selection.label);
      }}
    >
      <ComboboxInput className="w-full" placeholder="Select Package">
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          aria-label="Remove package reference"
          onClick={(event) => {
            event.stopPropagation();
            onRemove();
          }}
        >
          <Trash2 />
        </Button>
      </ComboboxInput>
      <ComboboxContent>
        <ComboboxEmpty>
          {packageGroups.length === 0 ? "No Packages Available." : "No Packages Found."}
        </ComboboxEmpty>
        <ComboboxList>
          {packageGroups.map((group) => (
            <ComboboxGroup key={group.softwareID}>
              <ComboboxLabel>{group.softwareTitle}</ComboboxLabel>
              <ComboboxItem value={packageReferenceSoftwareValue(group.softwareID)}>
                All versions
              </ComboboxItem>
              {group.packages.map((option) => (
                <ComboboxItem key={option.id} value={packageReferencePackageValue(option.id)}>
                  {packageLabel(option)}
                </ComboboxItem>
              ))}
            </ComboboxGroup>
          ))}
        </ComboboxList>
      </ComboboxContent>
    </Combobox>
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
    { softwareID: number; softwareTitle: string; packages: MunkiPackage[] }
  >();
  for (const pkg of packages) {
    const group = groups.get(pkg.software.id) ?? {
      softwareID: pkg.software.id,
      softwareTitle: pkg.software.name,
      packages: [],
    };
    group.packages.push(pkg);
    groups.set(pkg.software.id, group);
  }
  return [...groups.values()];
}

function packageLabel(pkg: MunkiPackage) {
  return `${pkg.software.name} ${pkg.version}`;
}
