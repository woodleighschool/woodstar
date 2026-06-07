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
import { useSantaRuleTargets, type SantaRulePolicy, type SantaRuleTarget, type SantaRuleType } from "@/hooks/use-santa";
import {
  ruleIdentifierHint,
  type RuleFormState,
  type RuleIncludeErrors,
  type RuleIncludeForm,
} from "@/lib/santa-rule-form";
import { POLICY_OPTIONS, ruleTypeLabel } from "@/lib/santa-rules";

export function RuleTargetPicker({
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
  const targets = useSantaRuleTargets({ target_type: form.rule_type, limit: 50 });
  const rows = targets.data ?? [];
  const selected =
    rows.find((target) => target.target_type === form.rule_type && target.identifier === form.identifier) ??
    currentRuleTarget(form);
  const items =
    selected &&
    !rows.some((target) => target.target_type === selected.target_type && target.identifier === selected.identifier)
      ? [selected, ...rows]
      : rows;

  return (
    <Field data-invalid={identifierInvalid}>
      <FieldLabel htmlFor="santa-rule-target" required>
        Target
      </FieldLabel>
      <Combobox
        items={items}
        value={selected}
        itemToStringLabel={targetLabel}
        itemToStringValue={(target) => target.identifier}
        onValueChange={(target) => {
          if (!target) {
            onChange({ ...form, identifier: "" });
            return;
          }
          onChange({
            ...form,
            rule_type: target.target_type,
            identifier: target.identifier,
            name: form.name.trim() === "" && target.display_name ? target.display_name : form.name,
          });
        }}
      >
        <ComboboxInput
          id="santa-rule-target"
          required
          aria-invalid={identifierInvalid ? true : undefined}
          showClear
          disabled={targets.isLoading}
          placeholder={targets.isLoading ? "Loading Targets" : "Select Target"}
        />
        <ComboboxContent>
          <ComboboxEmpty>
            {targets.error ? targets.error.message : targets.isLoading ? "Loading Targets..." : "No Targets Found."}
          </ComboboxEmpty>
          <ComboboxList>{ruleTargetItem}</ComboboxList>
        </ComboboxContent>
      </Combobox>
      <FieldDescription>{ruleTargetDescription(selected, form.rule_type)}</FieldDescription>
      {identifierInvalid && identifierError ? <FieldError>{identifierError}</FieldError> : null}
    </Field>
  );
}

function currentRuleTarget(form: RuleFormState): SantaRuleTarget | null {
  const identifier = form.identifier.trim();
  if (identifier === "") return null;
  return {
    target_type: form.rule_type,
    identifier,
    rule_count: 0,
    complete: true,
  };
}

function targetLabel(target: SantaRuleTarget) {
  return target.identifier;
}

function ruleTargetItem(target: SantaRuleTarget) {
  const disabled = target.target_type === "bundle" && !target.complete;
  const secondary = ruleTargetSecondary(target);
  return (
    <ComboboxItem key={`${target.target_type}:${target.identifier}`} value={target} disabled={disabled}>
      <span className="min-w-0 flex-1">
        <span className="block break-all leading-snug">{targetLabel(target)}</span>
        {secondary ? <span className="text-muted-foreground block truncate text-xs">{secondary}</span> : null}
      </span>
      <span className="flex shrink-0 items-center gap-2">
        {target.target_type === "bundle" ? (
          <span className="text-muted-foreground text-xs tabular-nums">
            {target.collected_binary_count ?? 0}/{target.binary_count ?? 0}
          </span>
        ) : null}
        <Badge variant={disabled ? "outline" : "secondary"} className="font-normal">
          {ruleTypeLabel(target.target_type)}
        </Badge>
      </span>
    </ComboboxItem>
  );
}

function ruleTargetSecondary(target: SantaRuleTarget) {
  if (target.target_type === "teamid") return undefined;
  if (!target.display_name) return undefined;
  return target.display_name;
}

function ruleTargetDescription(target: SantaRuleTarget | null, ruleType: SantaRuleType) {
  if (!target) return ruleIdentifierHint(ruleType);
  if (target.target_type === "bundle" && !target.complete) {
    return "Bundle target is incomplete.";
  }
  const description = ruleTargetDescriptionParts(target).join(" | ");
  return description || ruleIdentifierHint(target.target_type);
}

function ruleTargetDescriptionParts(target: SantaRuleTarget) {
  const parts: string[] = [];
  if (target.target_type === "teamid") {
    if (target.certificate_organization) parts.push(target.certificate_organization);
    return parts;
  }
  if (target.target_type === "certificate" && target.certificate_common_name) {
    parts.push(target.certificate_common_name);
  }
  if (target.target_type === "certificate" && target.certificate_organization) {
    parts.push(target.certificate_organization);
  }
  if (target.bundle_identifier) parts.push(target.bundle_identifier);
  if (target.version) parts.push(target.version);
  if (target.path) parts.push(target.path);
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
      columnsBeforeLabel={policyColumns}
      onChange={onChange}
      onLabelChange={(id, labelID) => onUpdate(id, { label_id: labelID })}
      renderActions={(row) => (
        <Button type="button" variant="ghost" size="icon" onClick={() => onDelete(row.id)}>
          <Trash2 />
        </Button>
      )}
      empty={<span className="text-muted-foreground text-sm">No Include Targets.</span>}
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
