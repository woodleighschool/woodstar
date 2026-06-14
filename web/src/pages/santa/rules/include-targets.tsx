import { encodeSort, MAX_PAGE_SIZE } from "@/hooks/use-data-table-search";
import type { ColumnDef } from "@tanstack/react-table";
import { ExternalLink, Pencil, Plus, Trash2 } from "lucide-react";
import { useMemo, useState } from "react";

import { DataTableStatic } from "@/components/data-table/data-table-static";
import { CodeEditor } from "@/components/editor/code-editor";
import { EmptyPanel } from "@/components/empty-panel";
import {
  AssignmentLabelField,
  AssignmentLabelStatic,
} from "@/components/targeting/assignment-label-field";
import { TargetSection } from "@/components/targeting/target-section";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Field, FieldError, FieldLabel } from "@/components/ui/field";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useLabels } from "@/hooks/use-labels";
import { type SantaRulePolicy } from "@/hooks/use-santa-rules";
import { POLICY_OPTIONS } from "@/lib/santa-rules";

import { santaCELExpressionError } from "./cel";
import { type RuleIncludeForm } from "./form-state";

const POLICY_LABELS = new Map(POLICY_OPTIONS.map((option) => [option.value, option.label]));

interface IncludeDraft {
  label_id: number | null;
  policy: SantaRulePolicy;
  cel_expression: string;
}

type DialogState = { mode: "add" } | { mode: "edit"; id: number } | null;

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
  const [draft, setDraft] = useState<IncludeDraft>(emptyDraft);
  const labels = useLabels({ per_page: MAX_PAGE_SIZE, sort: encodeSort("name") });
  const labelsByID = useMemo(
    () =>
      new Map<number, string>((labels.data?.items ?? []).map((label) => [label.id, label.name])),
    [labels.data],
  );
  const usedLabelIDs = include.flatMap((row) => (row.label_id === null ? [] : [row.label_id]));
  const unavailableLabelIDs = [...usedLabelIDs, ...excludeLabelIDs];
  const celError = celDraftError(draft);
  const canSave = (dialog?.mode !== "add" || draft.label_id !== null) && celError === undefined;

  function openAdd() {
    setDraft(emptyDraft());
    setDialog({ mode: "add" });
  }

  function openEdit(row: RuleIncludeForm) {
    setDraft({ label_id: row.label_id, policy: row.policy, cel_expression: row.cel_expression });
    setDialog({ mode: "edit", id: row.id });
  }

  function save() {
    if (!canSave || dialog === null) return;
    if (dialog.mode === "add") {
      onChange([
        ...include,
        {
          id: nextIncludeID(include),
          label_id: draft.label_id,
          policy: draft.policy,
          cel_expression: draft.cel_expression,
        },
      ]);
    } else {
      onChange(
        include.map((row) =>
          row.id === dialog.id
            ? { ...row, policy: draft.policy, cel_expression: draft.cel_expression }
            : row,
        ),
      );
    }
    setDialog(null);
  }

  const columns = useMemo<ColumnDef<RuleIncludeForm>[]>(
    () => [
      {
        id: "label",
        header: "Label",
        enableSorting: false,
        cell: ({ row }) =>
          row.original.label_id === null
            ? "-"
            : (labelsByID.get(row.original.label_id) ?? `Label ${row.original.label_id}`),
      },
      {
        id: "policy",
        header: "Policy",
        enableSorting: false,
        cell: ({ row }) => (
          <div className="flex min-w-0 items-center gap-2">
            <Badge variant="secondary" className="font-normal">
              {POLICY_LABELS.get(row.original.policy) ?? row.original.policy}
            </Badge>
            {row.original.policy === "cel" && row.original.cel_expression ? (
              <code className="min-w-0 truncate text-xs text-muted-foreground">
                {row.original.cel_expression}
              </code>
            ) : null}
          </div>
        ),
      },
      {
        id: "actions",
        header: () => null,
        enableSorting: false,
        cell: ({ row }) => (
          <div className="flex justify-end gap-1">
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              aria-label="Edit include"
              onClick={() => openEdit(row.original)}
            >
              <Pencil />
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              aria-label="Remove include"
              onClick={() => onChange(include.filter((item) => item.id !== row.original.id))}
            >
              <Trash2 />
            </Button>
          </div>
        ),
      },
    ],
    // openEdit/onChange/include captured fresh each render; columns rebuild on data change.
    [include, labelsByID, onChange],
  );

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
        <DataTableStatic columns={columns} data={include} />
      ) : (
        <EmptyPanel>No includes yet</EmptyPanel>
      )}

      <Dialog open={dialog !== null} onOpenChange={(open) => (open ? null : setDialog(null))}>
        <DialogContent className="sm:max-w-xl">
          <DialogHeader>
            <DialogTitle>{dialog?.mode === "edit" ? "Edit Include" : "Add Include"}</DialogTitle>
          </DialogHeader>

          {dialog?.mode === "edit" ? (
            <AssignmentLabelStatic
              name={labelsByID.get(draft.label_id ?? -1) ?? `Label ${draft.label_id ?? ""}`}
            />
          ) : (
            <AssignmentLabelField
              value={draft.label_id}
              onChange={(label_id) => setDraft((current) => ({ ...current, label_id }))}
              unavailableLabelIDs={unavailableLabelIDs}
            />
          )}

          <Field>
            <FieldLabel htmlFor="santa-include-policy">Policy</FieldLabel>
            <Select
              value={draft.policy}
              onValueChange={(policy) =>
                setDraft((current) => ({ ...current, policy: policy as SantaRulePolicy }))
              }
            >
              <SelectTrigger id="santa-include-policy" className="w-full">
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
          </Field>

          {draft.policy === "cel" ? (
            <Field data-invalid={celError ? true : undefined} className="gap-2">
              <FieldLabel required>Expression</FieldLabel>
              <CodeEditor
                value={draft.cel_expression}
                onChange={(cel_expression) =>
                  setDraft((current) => ({ ...current, cel_expression }))
                }
                placeholder="target.signing_time >= timestamp('2025-05-31T00:00:00Z')"
                lineNumbers={false}
                highlightActiveLine={false}
                invalid={celError ? true : undefined}
                className="[&_.cm-content]:min-h-28 [&_.cm-scroller]:max-h-48 [&_.cm-scroller]:overflow-auto"
              />
              {celError ? <FieldError>{celError}</FieldError> : null}
              <a
                href="https://northpole.dev/features/binary-authorization/#cel"
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center gap-1 text-xs text-muted-foreground underline-offset-4 hover:text-primary hover:underline"
              >
                <ExternalLink className="size-3" />
                Northpole CEL
              </a>
            </Field>
          ) : null}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setDialog(null)}>
              Cancel
            </Button>
            <Button type="button" onClick={save} disabled={!canSave}>
              {dialog?.mode === "edit" ? "Save" : "Add"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </TargetSection>
  );
}

function emptyDraft(): IncludeDraft {
  return { label_id: null, policy: "allowlist", cel_expression: "" };
}

function celDraftError(draft: IncludeDraft): string | undefined {
  if (draft.policy !== "cel") return undefined;
  if (draft.cel_expression.trim() === "") return "CEL policy requires an expression.";
  return santaCELExpressionError(draft.cel_expression);
}

function nextIncludeID(include: RuleIncludeForm[]) {
  return include.reduce((max, row) => Math.max(max, row.id), 0) + 1;
}
