import { FileArchive, Trash2 } from "lucide-react";
import { type ReactNode, useRef } from "react";

import { FormField } from "@/components/form-field";
import {
  Attachment,
  AttachmentAction,
  AttachmentActions,
  AttachmentContent,
  AttachmentDescription,
  AttachmentMedia,
  AttachmentTitle,
  AttachmentTrigger,
} from "@/components/ui/attachment";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Editable, EditableArea, EditableInput, EditablePreview } from "@/components/ui/editable";
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@/components/ui/input-group";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { MunkiPackage } from "@/lib/api";
import { formatBytes } from "@/lib/utils";

import { MUNKI_INSTALL_ITEM_TYPE_OPTIONS } from "../software/munki-software";
import type { PackageEditorForm } from "./editor-form";
import {
  type Architecture,
  emptyStringRow,
  type InstallerEnvironmentRow,
  type InstallItemRow,
  type ItemToCopyRow,
  numberOrUndefined,
  type ReceiptRow,
  type StringRow,
  toggleArray,
} from "./form-state";
import { CheckboxControl } from "./package-form-controls";

export function InstallerFileField({
  form,
  metadata,
}: {
  form: PackageEditorForm;
  metadata?: MunkiPackage["installer_file"];
}) {
  const inputRef = useRef<HTMLInputElement>(null);

  return (
    <form.Field name="installer_file">
      {(field) => (
        <FormField
          field={field}
          label={metadata ? "Replacement installer" : "Installer file"}
          htmlFor="munki-package-installer-file"
          required={!metadata}
        >
          {(control) => {
            const file = field.state.value;
            const filename = file?.name ?? metadata?.filename ?? "Choose an installer";
            const description = file
              ? `${formatBytes(file.size)} selected`
              : metadata
                ? `${formatBytes(metadata.size_bytes)} · select to replace`
                : "Select an installer file.";
            return (
              <div className="relative w-full">
                <Input
                  ref={inputRef}
                  key={file ? `${file.name}:${file.size}:${file.lastModified}` : "empty"}
                  id="munki-package-installer-file-input"
                  type="file"
                  hidden
                  onBlur={field.handleBlur}
                  onChange={(event) => field.handleChange(event.target.files?.[0] ?? null)}
                />
                <Attachment
                  state={control["aria-invalid"] ? "error" : file || metadata ? "done" : "idle"}
                  className="w-full"
                >
                  <AttachmentMedia>
                    <FileArchive />
                  </AttachmentMedia>
                  <AttachmentContent>
                    <AttachmentTitle>{filename}</AttachmentTitle>
                    <AttachmentDescription>{description}</AttachmentDescription>
                  </AttachmentContent>
                  {file ? (
                    <AttachmentActions>
                      <AttachmentAction
                        type="button"
                        aria-label="Clear selected installer"
                        onClick={() => field.handleChange(null)}
                      >
                        <Trash2 />
                      </AttachmentAction>
                    </AttachmentActions>
                  ) : null}
                  <AttachmentTrigger
                    id="munki-package-installer-file"
                    aria-label={metadata ? "Select replacement installer" : "Select installer"}
                    aria-invalid={control["aria-invalid"]}
                    onClick={() => inputRef.current?.click()}
                  />
                </Attachment>
                {metadata && !file ? (
                  <p className="mt-2 font-mono text-xs break-all text-muted-foreground">
                    SHA-256 {metadata.sha256}
                  </p>
                ) : null}
              </div>
            );
          }}
        </FormField>
      )}
    </form.Field>
  );
}

export function ArchitectureEditor({
  values,
  onChange,
}: {
  values: Architecture[];
  onChange: (values: Architecture[]) => void;
}) {
  return (
    <FieldSet>
      <FieldLegend>Supported Architectures</FieldLegend>
      <FieldGroup data-slot="checkbox-group">
        <CheckboxControl
          id="munki-package-arch-arm64"
          label="Apple silicon"
          checked={values.includes("arm64")}
          onChange={(checked) => onChange(toggleArray(values, "arm64", checked))}
        />
        <CheckboxControl
          id="munki-package-arch-x86"
          label="Intel"
          checked={values.includes("x86_64")}
          onChange={(checked) => onChange(toggleArray(values, "x86_64", checked))}
        />
      </FieldGroup>
    </FieldSet>
  );
}

export function BlockingApplicationsEditor({ form }: { form: PackageEditorForm }) {
  return (
    <form.Subscribe selector={(state) => state.values.blocking_applications_none}>
      {(blockingApplicationsNone) => (
        <form.Field
          name="blocking_applications"
          mode="array"
          children={(field) => (
            <FormField field={field}>
              {(control) => (
                <div {...control} tabIndex={-1}>
                  <FieldSet className="gap-4">
                    <FieldLegend variant="label">Blocking Applications</FieldLegend>
                    <FieldGroup className="gap-2">
                      <form.Field
                        name="blocking_applications_none"
                        children={(noneField) => (
                          <Field orientation="horizontal">
                            <Checkbox
                              id="munki-package-blocking-applications-none"
                              checked={noneField.state.value}
                              onCheckedChange={(checked) => {
                                noneField.handleChange(checked);
                                if (checked && field.state.value.length > 0) {
                                  field.handleChange([]);
                                }
                              }}
                            />
                            <FieldContent>
                              <FieldLabel htmlFor="munki-package-blocking-applications-none">
                                No blocking applications
                              </FieldLabel>
                              <FieldDescription>
                                Install without checking for open applications.
                              </FieldDescription>
                            </FieldContent>
                          </Field>
                        )}
                      />
                      {blockingApplicationsNone ? null : (
                        <>
                          <StringArrayRows
                            removeLabel="Remove application"
                            rows={field.state.value}
                            onReplace={(index, row) => field.replaceValue(index, row)}
                            onRemove={(index) => field.removeValue(index)}
                          />
                          <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            className="w-fit"
                            onClick={() => field.pushValue(emptyStringRow())}
                          >
                            Add application
                          </Button>
                        </>
                      )}
                    </FieldGroup>
                  </FieldSet>
                </div>
              )}
            </FormField>
          )}
        />
      )}
    </form.Subscribe>
  );
}

function StringArrayRows({
  removeLabel,
  rows,
  onReplace,
  onRemove,
}: {
  removeLabel: string;
  rows: StringRow[];
  onReplace: (index: number, row: StringRow) => void;
  onRemove: (index: number) => void;
}) {
  return (
    <>
      {rows.map((row, index) => (
        <InputGroup key={row.rowID}>
          <InputGroupInput
            aria-label="Application"
            value={row.value}
            onChange={(event) => onReplace(index, { ...row, value: event.target.value })}
          />
          <InputGroupAddon align="inline-end">
            <InputGroupButton
              type="button"
              variant="ghost"
              size="icon-xs"
              aria-label={removeLabel}
              onClick={() => onRemove(index)}
            >
              <Trash2 />
            </InputGroupButton>
          </InputGroupAddon>
        </InputGroup>
      ))}
    </>
  );
}

export function InstallerEnvironmentEditor({
  rows,
  onAdd,
  onReplace,
  onRemove,
}: {
  rows: InstallerEnvironmentRow[];
  onAdd: () => void;
  onReplace: (index: number, row: InstallerEnvironmentRow) => void;
  onRemove: (index: number) => void;
}) {
  return (
    <FieldSet className="gap-4">
      <FieldLegend variant="label">Installer Environment</FieldLegend>
      <FieldGroup className="gap-2">
        {rows.map((row, index) => (
          <div key={row.rowID} className="grid gap-2 md:grid-cols-[minmax(0,12rem)_minmax(0,1fr)]">
            <Input
              aria-label="Name"
              value={row.name}
              onChange={(event) => onReplace(index, { ...row, name: event.target.value })}
            />
            <InputGroup>
              <InputGroupInput
                aria-label="Value"
                value={row.value}
                onChange={(event) => onReplace(index, { ...row, value: event.target.value })}
              />
              <InputGroupAddon align="inline-end">
                <InputGroupButton
                  type="button"
                  variant="ghost"
                  size="icon-xs"
                  aria-label="Remove variable"
                  onClick={() => onRemove(index)}
                >
                  <Trash2 />
                </InputGroupButton>
              </InputGroupAddon>
            </InputGroup>
          </div>
        ))}
        <Button type="button" variant="outline" size="sm" className="w-fit" onClick={onAdd}>
          Add variable
        </Button>
      </FieldGroup>
    </FieldSet>
  );
}

export function InstallsTable({
  rows,
  onAdd,
  onReplace,
  onRemove,
}: {
  rows: InstallItemRow[];
  onAdd: () => void;
  onReplace: (index: number, row: InstallItemRow) => void;
  onRemove: (index: number) => void;
}) {
  return (
    <FieldSet className="min-w-0 gap-4">
      <FieldLegend variant="label">Installs</FieldLegend>
      {rows.length > 0 ? (
        <div className="overflow-x-auto rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="min-w-[14rem]">Path</TableHead>
                <TableHead className="w-[9rem]">Type</TableHead>
                <TableHead className="min-w-[10rem]">CFBundleName</TableHead>
                <TableHead className="min-w-[12rem]">CFBundleIdentifier</TableHead>
                <TableHead className="min-w-[9rem]">CFBundleShortVersionString</TableHead>
                <TableHead className="min-w-[9rem]">CFBundleVersion</TableHead>
                <TableHead className="min-w-[9rem]">Minimum Update</TableHead>
                <TableHead className="w-9" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row, index) => (
                <TableRow key={row.rowID} className="hover:bg-transparent">
                  <TableCell className="p-0">
                    <CellInput
                      aria-label="Path"
                      value={row.path}
                      onValueChange={(value) => onReplace(index, { ...row, path: value })}
                    />
                  </TableCell>
                  <TableCell className="p-0">
                    <Select
                      value={row.type}
                      onValueChange={(next) =>
                        onReplace(index, {
                          ...row,
                          type: next as InstallItemRow["type"],
                        })
                      }
                    >
                      <SelectTrigger
                        aria-label="Type"
                        className="h-8 rounded-none border-0 bg-transparent px-2 shadow-none focus-visible:ring-1 focus-visible:ring-inset dark:bg-transparent"
                      >
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectGroup>
                          {MUNKI_INSTALL_ITEM_TYPE_OPTIONS.map((option) => (
                            <SelectItem key={option.value} value={option.value}>
                              {option.label}
                            </SelectItem>
                          ))}
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                  </TableCell>
                  <TableCell className="p-0">
                    <CellInput
                      aria-label="CFBundleName"
                      value={row.bundle_name ?? ""}
                      onValueChange={(value) => onReplace(index, { ...row, bundle_name: value })}
                    />
                  </TableCell>
                  <TableCell className="p-0">
                    <CellInput
                      aria-label="CFBundleIdentifier"
                      value={row.bundle_identifier ?? ""}
                      onValueChange={(value) =>
                        onReplace(index, { ...row, bundle_identifier: value })
                      }
                    />
                  </TableCell>
                  <TableCell className="p-0">
                    <CellInput
                      aria-label="CFBundleShortVersionString"
                      value={row.bundle_short_version ?? ""}
                      onValueChange={(value) =>
                        onReplace(index, {
                          ...row,
                          bundle_short_version: value,
                        })
                      }
                    />
                  </TableCell>
                  <TableCell className="p-0">
                    <CellInput
                      aria-label="CFBundleVersion"
                      value={row.bundle_version ?? ""}
                      onValueChange={(value) => onReplace(index, { ...row, bundle_version: value })}
                    />
                  </TableCell>
                  <TableCell className="p-0">
                    <CellInput
                      aria-label="Minimum Update Version"
                      value={row.minimum_update_version ?? ""}
                      onValueChange={(value) =>
                        onReplace(index, {
                          ...row,
                          minimum_update_version: value,
                        })
                      }
                    />
                  </TableCell>
                  <TableCell className="w-9 p-0 pr-1 text-right">
                    <IconButton label="Remove" onClick={() => onRemove(index)}>
                      <Trash2 />
                    </IconButton>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : null}
      <Button type="button" variant="outline" size="sm" className="w-fit" onClick={onAdd}>
        Add install item
      </Button>
    </FieldSet>
  );
}

export function ReceiptsTable({
  rows,
  onAdd,
  onReplace,
  onRemove,
}: {
  rows: ReceiptRow[];
  onAdd: () => void;
  onReplace: (index: number, row: ReceiptRow) => void;
  onRemove: (index: number) => void;
}) {
  return (
    <FieldSet className="min-w-0 gap-4">
      <FieldLegend variant="label">Receipts</FieldLegend>
      {rows.length > 0 ? (
        <div className="overflow-x-auto rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="min-w-[16rem]">Package ID</TableHead>
                <TableHead className="min-w-[9rem]">Version</TableHead>
                <TableHead className="min-w-[10rem]">Name</TableHead>
                <TableHead className="min-w-[8rem]">Installed Size</TableHead>
                <TableHead className="w-24 text-center">Optional</TableHead>
                <TableHead className="w-9" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row, index) => (
                <TableRow key={row.rowID} className="hover:bg-transparent">
                  <TableCell className="p-0">
                    <CellInput
                      aria-label="Package ID"
                      value={row.package_id}
                      onValueChange={(value) => onReplace(index, { ...row, package_id: value })}
                    />
                  </TableCell>
                  <TableCell className="p-0">
                    <CellInput
                      aria-label="Version"
                      value={row.version ?? ""}
                      onValueChange={(value) => onReplace(index, { ...row, version: value })}
                    />
                  </TableCell>
                  <TableCell className="p-0">
                    <CellInput
                      aria-label="Name"
                      value={row.name ?? ""}
                      onValueChange={(value) => onReplace(index, { ...row, name: value })}
                    />
                  </TableCell>
                  <TableCell className="p-0">
                    <CellInput
                      aria-label="Installed Size"
                      type="number"
                      min={0}
                      value={row.installed_size ?? ""}
                      onValueChange={(value) =>
                        onReplace(index, {
                          ...row,
                          installed_size: numberOrUndefined(value),
                        })
                      }
                    />
                  </TableCell>
                  <TableCell className="text-center">
                    <Checkbox
                      aria-label="Optional"
                      checked={row.optional === true}
                      onCheckedChange={(value) => onReplace(index, { ...row, optional: value })}
                    />
                  </TableCell>
                  <TableCell className="w-9 p-0 pr-1 text-right">
                    <IconButton label="Remove" onClick={() => onRemove(index)}>
                      <Trash2 />
                    </IconButton>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : null}
      <Button type="button" variant="outline" size="sm" className="w-fit" onClick={onAdd}>
        Add receipt
      </Button>
    </FieldSet>
  );
}

function CellInput({
  value,
  onValueChange,
  type = "text",
  min,
  "aria-label": ariaLabel,
}: {
  value: string | number;
  onValueChange: (value: string) => void;
  type?: "number" | "text";
  min?: number;
  "aria-label": string;
}) {
  const text = String(value);
  return (
    <Editable value={text} onValueChange={onValueChange} placeholder="—" className="gap-0">
      <EditableArea className="block">
        <EditablePreview
          aria-label={ariaLabel}
          className="h-8 rounded-none px-2 focus-visible:ring-inset"
        />
        <EditableInput
          aria-label={ariaLabel}
          type={type}
          min={min}
          className="h-8 rounded-none border-0 px-2 shadow-none focus-visible:ring-inset"
        />
      </EditableArea>
    </Editable>
  );
}

export function ItemsToCopyEditor({
  rows,
  onAdd,
  onReplace,
  onRemove,
}: {
  rows: ItemToCopyRow[];
  onAdd: () => void;
  onReplace: (index: number, row: ItemToCopyRow) => void;
  onRemove: (index: number) => void;
}) {
  return (
    <FieldSet className="gap-4">
      <FieldLegend variant="label">Items to Copy</FieldLegend>
      <FieldGroup className="gap-4">
        {rows.map((row, index) => (
          <div key={row.rowID} className="flex flex-col gap-3 rounded-md border p-3">
            <div className="grid gap-3 md:grid-cols-2">
              <Field>
                <FieldLabel htmlFor={`munki-copy-source-${row.rowID}`}>Source Item</FieldLabel>
                <Input
                  id={`munki-copy-source-${row.rowID}`}
                  value={row.source_item}
                  onChange={(event) =>
                    onReplace(index, {
                      ...row,
                      source_item: event.target.value,
                    })
                  }
                />
              </Field>
              <Field>
                <FieldLabel htmlFor={`munki-copy-destination-${row.rowID}`}>
                  Destination Path
                </FieldLabel>
                <InputGroup>
                  <InputGroupInput
                    id={`munki-copy-destination-${row.rowID}`}
                    value={row.destination_path}
                    onChange={(event) =>
                      onReplace(index, {
                        ...row,
                        destination_path: event.target.value,
                      })
                    }
                  />
                  <InputGroupAddon align="inline-end">
                    <InputGroupButton
                      type="button"
                      variant="ghost"
                      size="icon-xs"
                      aria-label="Remove copy item"
                      onClick={() => onRemove(index)}
                    >
                      <Trash2 />
                    </InputGroupButton>
                  </InputGroupAddon>
                </InputGroup>
              </Field>
            </div>
            <FieldGroup className="grid gap-3 md:grid-cols-4">
              <Field>
                <FieldLabel htmlFor={`munki-copy-destination-item-${row.rowID}`}>
                  Destination Item
                </FieldLabel>
                <Input
                  id={`munki-copy-destination-item-${row.rowID}`}
                  value={row.destination_item ?? ""}
                  onChange={(event) =>
                    onReplace(index, {
                      ...row,
                      destination_item: event.target.value,
                    })
                  }
                />
              </Field>
              <Field>
                <FieldLabel htmlFor={`munki-copy-user-${row.rowID}`}>User</FieldLabel>
                <Input
                  id={`munki-copy-user-${row.rowID}`}
                  value={row.user ?? ""}
                  onChange={(event) => onReplace(index, { ...row, user: event.target.value })}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor={`munki-copy-group-${row.rowID}`}>Group</FieldLabel>
                <Input
                  id={`munki-copy-group-${row.rowID}`}
                  value={row.group ?? ""}
                  onChange={(event) => onReplace(index, { ...row, group: event.target.value })}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor={`munki-copy-mode-${row.rowID}`}>Mode</FieldLabel>
                <Input
                  id={`munki-copy-mode-${row.rowID}`}
                  value={row.mode ?? ""}
                  onChange={(event) => onReplace(index, { ...row, mode: event.target.value })}
                />
              </Field>
            </FieldGroup>
          </div>
        ))}
        <Button type="button" variant="outline" size="sm" className="w-fit" onClick={onAdd}>
          Add copy item
        </Button>
      </FieldGroup>
    </FieldSet>
  );
}

function IconButton({
  label,
  children,
  onClick,
}: {
  label: string;
  children: ReactNode;
  onClick: () => void;
}) {
  return (
    <Button type="button" variant="ghost" size="icon-sm" title={label} onClick={onClick}>
      {children}
    </Button>
  );
}
