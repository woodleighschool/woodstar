import type { ColumnDef } from "@tanstack/react-table";
import { Code2, ExternalLink, Trash2 } from "lucide-react";
import { useMemo } from "react";

import { CodeEditor } from "@/components/editor/code-editor";
import { LabelTargetRowsTable } from "@/components/targeting/label-target-rows-table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Field, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  useSantaRuleReferences,
  type SantaRulePolicy,
  type SantaRuleReference,
  type SantaRuleType,
} from "@/hooks/use-santa";
import {
  ruleIdentifierHint,
  type RuleFormState,
  type RuleIncludeErrors,
  type RuleIncludeForm,
} from "@/lib/santa-rule-form";
import { POLICY_OPTIONS, ruleTypeLabel } from "@/lib/santa-rules";

export function RuleReferencePicker({
  form,
  identifierError,
  identifierInvalid,
  onChange,
}: {
  form: RuleFormState;
  identifierError?: string;
  identifierInvalid: boolean;
  onChange: (form: RuleFormState) => void;
}) {
  const references = useSantaRuleReferences({ rule_type: form.rule_type, limit: 50 });
  const rows = references.data ?? [];
  const selected =
    rows.find((candidate) => candidate.rule_type === form.rule_type && candidate.identifier === form.identifier) ??
    currentRuleReferenceCandidate(form);
  const items =
    selected &&
    !rows.some(
      (candidate) => candidate.rule_type === selected.rule_type && candidate.identifier === selected.identifier,
    )
      ? [selected, ...rows]
      : rows;

  return (
    <Field data-invalid={identifierInvalid}>
      <FieldLabel htmlFor="santa-rule-reference" required>
        Identifier
      </FieldLabel>
      <Combobox
        items={items}
        value={selected}
        itemToStringLabel={referenceLabel}
        itemToStringValue={(candidate) => candidate.identifier}
        onValueChange={(candidate) => {
          if (!candidate) {
            onChange({ ...form, identifier: "" });
            return;
          }
          onChange({
            ...form,
            rule_type: candidate.rule_type,
            identifier: candidate.identifier,
            name: form.name.trim() === "" && candidate.display_name ? candidate.display_name : form.name,
          });
        }}
      >
        <ComboboxInput
          id="santa-rule-reference"
          required
          aria-invalid={identifierInvalid ? true : undefined}
          showClear
          disabled={references.isLoading}
          placeholder={references.isLoading ? "Loading References" : "Select Identifier"}
        />
        <ComboboxContent>
          <ComboboxEmpty>
            {references.error
              ? references.error.message
              : references.isLoading
                ? "Loading References..."
                : "No References Found."}
          </ComboboxEmpty>
          <ComboboxList>{ruleReferenceItem}</ComboboxList>
        </ComboboxContent>
      </Combobox>
      <FieldDescription>{ruleReferenceDescription(selected, form.rule_type)}</FieldDescription>
      {identifierInvalid && identifierError ? <FieldError>{identifierError}</FieldError> : null}
    </Field>
  );
}

function currentRuleReferenceCandidate(form: RuleFormState): SantaRuleReference | null {
  const identifier = form.identifier.trim();
  if (identifier === "") return null;
  return {
    rule_type: form.rule_type,
    identifier,
    rule_count: 0,
    complete: true,
  };
}

function referenceLabel(reference: SantaRuleReference) {
  return reference.identifier;
}

function ruleReferenceItem(reference: SantaRuleReference) {
  const disabled = reference.rule_type === "bundle" && !reference.complete;
  const secondary = ruleReferenceSecondary(reference);
  return (
    <ComboboxItem key={`${reference.rule_type}:${reference.identifier}`} value={reference} disabled={disabled}>
      <span className="min-w-0 flex-1">
        <span className="block break-all leading-snug">{referenceLabel(reference)}</span>
        {secondary ? <span className="text-muted-foreground block truncate text-xs">{secondary}</span> : null}
      </span>
      <span className="flex shrink-0 items-center gap-2">
        {reference.rule_type === "bundle" ? (
          <span className="text-muted-foreground text-xs tabular-nums">
            {reference.collected_binary_count ?? 0}/{reference.binary_count ?? 0}
          </span>
        ) : null}
        <Badge variant={disabled ? "outline" : "secondary"} className="font-normal">
          {ruleTypeLabel(reference.rule_type)}
        </Badge>
      </span>
    </ComboboxItem>
  );
}

function ruleReferenceSecondary(reference: SantaRuleReference) {
  if (reference.rule_type === "teamid") return undefined;
  if (!reference.display_name) return undefined;
  return reference.display_name;
}

function ruleReferenceDescription(reference: SantaRuleReference | null, ruleType: SantaRuleType) {
  if (!reference) return ruleIdentifierHint(ruleType);
  if (reference.rule_type === "bundle" && !reference.complete) {
    return "Bundle reference is incomplete.";
  }
  const description = ruleReferenceDescriptionParts(reference).join(" | ");
  return description || ruleIdentifierHint(reference.rule_type);
}

function ruleReferenceDescriptionParts(reference: SantaRuleReference) {
  const parts: string[] = [];
  if (reference.rule_type === "teamid") {
    if (reference.certificate_organization) parts.push(reference.certificate_organization);
    return parts;
  }
  if (reference.rule_type === "certificate" && reference.certificate_common_name) {
    parts.push(reference.certificate_common_name);
  }
  if (reference.rule_type === "certificate" && reference.certificate_organization) {
    parts.push(reference.certificate_organization);
  }
  if (reference.bundle_identifier) parts.push(reference.bundle_identifier);
  if (reference.version) parts.push(reference.version);
  if (reference.path) parts.push(reference.path);
  return parts;
}

export function IncludeTargetsTable({
  includeRows,
  showErrors,
  includeErrors,
  excludedLabelIDs,
  onChange,
  onUpdate,
  onEditCEL,
  onDelete,
}: {
  includeRows: RuleIncludeForm[];
  showErrors: boolean;
  includeErrors: Partial<Record<number, RuleIncludeErrors>>;
  excludedLabelIDs: number[];
  onChange: (includeRows: RuleIncludeForm[]) => void;
  onUpdate: (id: number, include: Partial<RuleIncludeForm>) => void;
  onEditCEL: (id: number) => void;
  onDelete: (id: number) => void;
}) {
  const policyColumns = useMemo<ColumnDef<RuleIncludeForm>[]>(
    () => [
      {
        id: "policy",
        header: "Policy",
        enableSorting: false,
        cell: ({ row }) => {
          const hasExpression = row.original.cel_expression.trim() !== "";
          const celLabel = hasExpression ? "Edit CEL Expression" : "Add CEL Expression";
          const error = showErrors ? includeErrors[row.original.id]?.cel_expression : undefined;
          const celInvalid = row.original.policy === "cel" && (!hasExpression || error !== undefined);

          return (
            <Field className="min-w-56 gap-1">
              <div className="flex items-center gap-2">
                <Select
                  value={row.original.policy}
                  onValueChange={(policy) => {
                    const nextPolicy = policy as SantaRulePolicy;
                    onUpdate(row.original.id, { policy: nextPolicy });
                    if (nextPolicy === "cel" && row.original.cel_expression.trim() === "") {
                      onEditCEL(row.original.id);
                    }
                  }}
                >
                  <SelectTrigger id={`include-policy-${row.original.id}`} className="w-full">
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
                {row.original.policy === "cel" ? (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        type="button"
                        variant="outline"
                        size="icon-sm"
                        data-invalid={celInvalid ? true : undefined}
                        className={celInvalid ? "text-destructive hover:text-destructive" : undefined}
                        onClick={() => onEditCEL(row.original.id)}
                      >
                        <Code2 />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>{celLabel}</TooltipContent>
                  </Tooltip>
                ) : null}
              </div>
              {error ? <FieldError>{error}</FieldError> : null}
            </Field>
          );
        },
      },
    ],
    [includeErrors, onEditCEL, onUpdate, showErrors],
  );

  return (
    <LabelTargetRowsTable
      rows={includeRows}
      excludeLabelIDs={excludedLabelIDs}
      labelErrors={includeLabelErrors(showErrors, includeErrors)}
      columnsAfterLabel={policyColumns}
      onChange={onChange}
      onLabelChange={(id, labelID) => onUpdate(id, { label_id: labelID })}
      renderActions={(row) => (
        <Button type="button" variant="ghost" size="icon" onClick={() => onDelete(row.id)}>
          <Trash2 />
        </Button>
      )}
      empty={<span className="text-muted-foreground text-sm">No Includes</span>}
      emptyClassName="min-h-24 items-center py-6"
    />
  );
}

function includeLabelErrors(showErrors: boolean, includeErrors: Partial<Record<number, RuleIncludeErrors>>) {
  if (!showErrors) return {};
  const errors: Partial<Record<number, string>> = {};
  for (const [id, error] of Object.entries(includeErrors)) {
    if (error?.label_id) errors[Number(id)] = error.label_id;
  }
  return errors;
}

export function CELDialog({
  include,
  error,
  showRequiredError,
  onOpenChange,
  onChange,
}: {
  include?: RuleIncludeForm;
  error?: string;
  showRequiredError: boolean;
  onOpenChange: (open: boolean) => void;
  onChange: (cel_expression: string) => void;
}) {
  const visibleError = include && (showRequiredError || include.cel_expression.trim() !== "") ? error : undefined;
  return (
    <Dialog open={include !== undefined} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>CEL</DialogTitle>
        </DialogHeader>

        {include ? (
          <Field data-invalid={visibleError ? true : undefined} className="gap-2">
            <FieldLabel required>Expression</FieldLabel>
            <CodeEditor
              value={include.cel_expression}
              onChange={onChange}
              placeholder="target.signing_time >= timestamp('2025-05-31T00:00:00Z')"
              lineNumbers={false}
              highlightActiveLine={false}
              invalid={visibleError ? true : undefined}
              className="[&_.cm-content]:min-h-28 [&_.cm-scroller]:max-h-48 [&_.cm-scroller]:overflow-auto"
            />
            {visibleError ? <FieldError>{visibleError}</FieldError> : null}
          </Field>
        ) : null}

        <DialogFooter className="gap-2 sm:justify-between">
          <Button asChild type="button" variant="outline" size="sm">
            <a href="https://northpole.dev/features/binary-authorization/#cel" target="_blank" rel="noreferrer">
              <ExternalLink data-icon="inline-start" />
              Northpole CEL
            </a>
          </Button>
          <Button type="button" size="sm" onClick={() => onOpenChange(false)}>
            Done
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
