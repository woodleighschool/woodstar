import { revalidateLogic, useForm } from "@tanstack/react-form";
import { ExternalLink, MoreHorizontal, Plus } from "lucide-react";
import { useMemo, useState } from "react";

import { CodeEditor } from "@/components/editor/code-editor";
import { EmptyPanel } from "@/components/empty-panel";
import { FormActions } from "@/components/form-actions";
import { FormField } from "@/components/form-field";
import { focusFirstInvalidField } from "@/components/form-tabs";
import { LabelPicker } from "@/components/labels/label-picker";
import { TargetSection } from "@/components/targeting/target-section";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
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
import { encodeSort } from "@/hooks/use-data-table-search";
import { useFormExitGuard } from "@/hooks/use-form-exit-guard";
import { useLabels } from "@/hooks/use-labels";
import { MAX_PAGE_SIZE } from "@/lib/pagination";
import { POLICY_OPTIONS, type SantaRulePolicy } from "@/lib/santa-rules";

import { type RuleIncludeForm, santaRuleIncludeSchema } from "./form-state";
const POLICY_LABELS = new Map(POLICY_OPTIONS.map((option) => [option.value, option.label]));
type DialogState =
  | {
      mode: "add";
    }
  | {
      mode: "edit";
      id: number;
    }
  | null;
export function SantaIncludeTargets({
  include,
  excludeLabelIDs,
  onChange,
}: {
  include: RuleIncludeForm[];
  excludeLabelIDs: number[];
  onChange: (include: RuleIncludeForm[]) => void;
}) {
  const [dialog, setDialog] = useState<DialogState>(null);
  const labels = useLabels({
    per_page: MAX_PAGE_SIZE,
    sort: encodeSort("name"),
  });
  const labelsByID = useMemo(
    () =>
      new Map<number, string>((labels.data?.items ?? []).map((label) => [label.id, label.name])),
    [labels.data],
  );
  const usedLabelIDs = include.flatMap((row) => (row.label_id === null ? [] : [row.label_id]));
  const unavailableLabelIDs = [...usedLabelIDs, ...excludeLabelIDs];
  function openAdd() {
    setDialog({ mode: "add" });
  }
  function openEdit(row: RuleIncludeForm) {
    setDialog({ mode: "edit", id: row.id });
  }
  return (
    <TargetSection
      title="Include"
      action={
        <Button type="button" variant="outline" size="sm" onClick={openAdd}>
          <Plus data-icon="inline-start" />
          Add Include
        </Button>
      }
    >
      {include.length > 0 ? (
        <div className="overflow-x-auto rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Label</TableHead>
                <TableHead>Policy</TableHead>
                <TableHead className="w-12">
                  <span className="sr-only">Actions</span>
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {include.map((row) => (
                <TableRow key={row.id}>
                  <TableCell>{includeLabel(row, labelsByID)}</TableCell>
                  <TableCell className="max-w-[28rem]">
                    <div className="flex min-w-0 items-center gap-2">
                      <Badge variant="secondary" className="font-normal">
                        {POLICY_LABELS.get(row.policy) ?? row.policy}
                      </Badge>
                      {row.policy === "cel" && row.cel_expression ? (
                        <code className="min-w-0 truncate text-xs text-muted-foreground">
                          {row.cel_expression}
                        </code>
                      ) : null}
                    </div>
                  </TableCell>
                  <TableCell className="w-12">
                    <SantaIncludeRowActions
                      onEdit={() => openEdit(row)}
                      onRemove={() => onChange(include.filter((item) => item.id !== row.id))}
                    />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : (
        <EmptyPanel>No includes yet</EmptyPanel>
      )}

      {dialog ? (
        <SantaIncludeDialog
          mode={dialog.mode}
          initial={
            dialog.mode === "edit"
              ? requiredInclude(include, dialog.id)
              : {
                  id: nextIncludeID(include),
                  label_id: null,
                  policy: "allowlist",
                  cel_expression: "",
                }
          }
          unavailableLabelIDs={unavailableLabelIDs}
          onClose={() => setDialog(null)}
          onSave={(target) => {
            onChange(
              dialog.mode === "add"
                ? [...include, target]
                : include.map((row) => (row.id === target.id ? target : row)),
            );
            setDialog(null);
          }}
        />
      ) : null}
    </TargetSection>
  );
}
function SantaIncludeDialog({
  mode,
  initial,
  unavailableLabelIDs,
  onClose,
  onSave,
}: {
  mode: "add" | "edit";
  initial: RuleIncludeForm;
  unavailableLabelIDs: readonly number[];
  onClose: () => void;
  onSave: (target: RuleIncludeForm) => void;
}) {
  const form = useForm({
    defaultValues: initial,
    validationLogic: revalidateLogic({
      mode: "submit",
      modeAfterSubmission: "change",
    }),
    validators: { onDynamic: santaRuleIncludeSchema },
    onSubmit: ({ value }) => onSave(santaRuleIncludeSchema.parse(value)),
  });
  const exitGuard = useFormExitGuard({
    form,
    onDiscard: onClose,
    blockNavigation: false,
  });
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) exitGuard.requestDiscard();
      }}
    >
      <DialogContent className="sm:max-w-xl">
        <form
          noValidate
          className="contents"
          onSubmit={(event) => {
            event.preventDefault();
            event.stopPropagation();
            void form.handleSubmit().then(() => {
              if (!form.state.isValid) focusFirstInvalidField();
              return undefined;
            });
          }}
        >
          <DialogHeader>
            <DialogTitle>{mode === "edit" ? "Edit Include" : "Add Include"}</DialogTitle>
          </DialogHeader>

          {mode === "add" ? (
            <form.Field name="label_id">
              {(field) => (
                <FormField field={field} label="Label" required>
                  {(control) => (
                    <LabelPicker
                      value={field.state.value === null ? [] : [field.state.value]}
                      onChange={(ids) => field.handleChange(ids[0] ?? null)}
                      selectionMode="single"
                      includeBuiltins
                      unavailableLabelIDs={unavailableLabelIDs}
                      required
                      invalid={control["aria-invalid"]}
                      placeholder="Select Label"
                    />
                  )}
                </FormField>
              )}
            </form.Field>
          ) : null}

          <form.Field name="policy">
            {(field) => (
              <FormField field={field} label="Policy" htmlFor="santa-include-policy" required>
                {(control) => (
                  <Select
                    value={field.state.value}
                    onValueChange={(policy) => field.handleChange(policy as SantaRulePolicy)}
                  >
                    <SelectTrigger {...control} className="w-full">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectGroup>
                        {POLICY_OPTIONS.map((policy) => (
                          <SelectItem key={policy.value} value={policy.value}>
                            {policy.label}
                          </SelectItem>
                        ))}
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                )}
              </FormField>
            )}
          </form.Field>

          <form.Subscribe selector={(state) => state.values.policy}>
            {(policy) =>
              policy === "cel" ? (
                <form.Field name="cel_expression">
                  {(field) => (
                    <FormField field={field} label="Expression" required className="gap-2">
                      {(control) => (
                        <>
                          <CodeEditor
                            value={field.state.value}
                            onChange={field.handleChange}
                            placeholder="target.signing_time >= timestamp('2025-05-31T00:00:00Z')"
                            lineNumbers={false}
                            highlightActiveLine={false}
                            invalid={control["aria-invalid"]}
                            className="[&_.cm-content]:min-h-28 [&_.cm-scroller]:max-h-48 [&_.cm-scroller]:overflow-auto"
                          />
                          <a
                            href="https://northpole.dev/features/binary-authorization/#cel"
                            target="_blank"
                            rel="noreferrer"
                            className="inline-flex items-center gap-1 text-xs text-muted-foreground underline-offset-4 hover:text-primary hover:underline"
                          >
                            <ExternalLink className="size-3" />
                            CEL Handbook
                          </a>
                        </>
                      )}
                    </FormField>
                  )}
                </form.Field>
              ) : null
            }
          </form.Subscribe>

          <FormActions
            form={form}
            submitLabel={mode === "edit" ? "Save" : "Add"}
            onCancel={exitGuard.requestDiscard}
            className="justify-end"
          />
        </form>
        {exitGuard.dialog}
      </DialogContent>
    </Dialog>
  );
}
function SantaIncludeRowActions({
  onEdit,
  onRemove,
}: {
  onEdit: () => void;
  onRemove: () => void;
}) {
  return (
    <div className="flex justify-end">
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              aria-label="Open include actions"
            />
          }
        >
          <MoreHorizontal />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-36">
          <DropdownMenuGroup>
            <DropdownMenuItem onSelect={onEdit}>Edit</DropdownMenuItem>
            <DropdownMenuItem variant="destructive" onSelect={onRemove}>
              Delete
            </DropdownMenuItem>
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
function includeLabel(row: RuleIncludeForm, labelsByID: Map<number, string>) {
  if (row.label_id === null) return "-";
  return labelsByID.get(row.label_id) ?? `Label ${row.label_id}`;
}
function nextIncludeID(include: RuleIncludeForm[]) {
  return include.reduce((max, row) => Math.max(max, row.id), 0) + 1;
}
function requiredInclude(include: RuleIncludeForm[], id: number) {
  const target = include.find((row) => row.id === id);
  if (!target) throw new Error("The selected Santa include no longer exists");
  return target;
}
