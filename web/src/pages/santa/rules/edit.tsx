import { Link, useNavigate, useParams, useSearch } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Code2, ExternalLink, Loader2, Plus, Trash2 } from "lucide-react";
import { useMemo, useState } from "react";
import { z } from "zod";

import { DataTable, DataTableRowDragHandle } from "@/components/data-table";
import { CodeEditor } from "@/components/editor/code-editor";
import { LabelPicker } from "@/components/labels/label-picker";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
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
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Textarea } from "@/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  useCreateSantaRule,
  useSantaRule,
  useSantaRuleTargets,
  useUpdateSantaRule,
  type SantaRule,
  type SantaRuleMutation,
  type SantaRuleTarget,
} from "@/hooks/use-santa";
import { fieldErrors, optionalText, positiveIntegerArray, requiredString } from "@/lib/form-validation";
import { santaCELExpressionError } from "@/lib/santa-cel";

import {
  POLICY_OPTIONS,
  POLICY_VALUES,
  RULE_IDENTIFIER_RULES,
  RULE_TYPE_OPTIONS,
  RULE_TYPE_VALUES,
  ruleTypeLabel,
  type RulePolicy,
  type RuleType,
} from "./shared";

const includeSchema = z
  .object({
    id: z.number(),
    policy: z.enum(POLICY_VALUES),
    cel_expression: z.string().trim(),
    label_id: z.number().int().positive("Pick a label.").nullable(),
  })
  .refine((value) => value.label_id !== null, {
    message: "Pick a label.",
    path: ["label_id"],
  })
  .superRefine((value, ctx) => {
    if (value.policy !== "cel") return;
    if (value.cel_expression === "") {
      ctx.addIssue({ code: "custom", message: "CEL policy requires an expression.", path: ["cel_expression"] });
      return;
    }
    const error = santaCELExpressionError(value.cel_expression);
    if (error) {
      ctx.addIssue({ code: "custom", message: error, path: ["cel_expression"] });
    }
  });

const ruleFormSchema = z
  .object({
    rule_type: z.enum(RULE_TYPE_VALUES),
    identifier: requiredString("Target"),
    name: requiredString("Name"),
    description: z.string().trim(),
    custom_message: z.string().trim(),
    custom_url: z.string().trim(),
    exclude_label_ids: positiveIntegerArray("Label"),
    includes: z.array(includeSchema),
  })
  .superRefine((value, ctx) => {
    if (value.identifier === "") return;
    const rule = RULE_IDENTIFIER_RULES[value.rule_type];
    if (!rule.pattern.test(value.identifier)) {
      ctx.addIssue({ code: "custom", message: rule.hint, path: ["identifier"] });
    }
  });

type RuleFormParse = ReturnType<typeof ruleFormSchema.safeParse>;
type RuleIncludeErrors = { cel_expression?: string; label_id?: string };

interface RuleIncludeForm {
  id: number;
  policy: RulePolicy;
  cel_expression: string;
  label_id: number | null;
}

interface RuleFormState {
  rule_type: RuleType;
  identifier: string;
  name: string;
  description: string;
  custom_message: string;
  custom_url: string;
  exclude_label_ids: number[];
  includes: RuleIncludeForm[];
}

const emptyRuleForm: RuleFormState = {
  rule_type: "binary",
  identifier: "",
  name: "",
  description: "",
  custom_message: "",
  custom_url: "",
  exclude_label_ids: [],
  includes: [],
};

export function SantaRuleEditPage({ mode }: { mode: "create" | "edit" }) {
  const params = useParams({ strict: false });
  const search = useSearch({ strict: false });
  const ruleId = params.ruleId ?? "";
  const ruleID = mode === "edit" ? Number(ruleId) : null;
  const detail = useSantaRule(ruleID);

  if (mode === "edit") {
    if (detail.error) {
      return (
        <PageShell>
          <Alert variant="destructive">
            <AlertTitle>Failed to Load Rule</AlertTitle>
            <AlertDescription>{detail.error.message}</AlertDescription>
          </Alert>
        </PageShell>
      );
    }
    if (!detail.data) {
      return (
        <PageShell className="text-muted-foreground flex-row items-center gap-2 text-sm">
          <Loader2 className="animate-spin" /> Loading Rule...
        </PageShell>
      );
    }
  }

  const initial = mode === "edit" && detail.data ? formFromRule(detail.data) : formFromSearch(search);

  return <RuleForm key={ruleId || "new"} mode={mode} ruleId={ruleID} initial={initial} />;
}

function RuleForm({
  mode,
  ruleId,
  initial,
}: {
  mode: "create" | "edit";
  ruleId: number | null;
  initial: RuleFormState;
}) {
  const navigate = useNavigate();
  const create = useCreateSantaRule();
  const update = useUpdateSantaRule();
  const [form, setForm] = useState<RuleFormState>(initial);
  const [showErrors, setShowErrors] = useState(false);
  const [celDialogID, setCELDialogID] = useState<number | null>(null);
  const pending = create.isPending || update.isPending;
  const parsed = useMemo(() => ruleFormSchema.safeParse(form), [form]);
  const errors = useMemo(() => fieldErrors(parsed), [parsed]);
  const identifierError = identifierErrorFor(parsed);
  const identifierInvalid = identifierError !== undefined && (showErrors || form.identifier.trim() !== "");
  const includeErrors = useMemo(() => includeErrorMap(parsed, form.includes), [parsed, form.includes]);
  const includeLabelIDs = useMemo(() => selectedIncludeLabelIDs(form.includes), [form.includes]);
  const canSave = parsed.success;

  async function submit() {
    if (!canSave) {
      setShowErrors(true);
      return;
    }
    if (mode === "create") await create.mutateAsync(ruleBody(form));
    else await update.mutateAsync({ id: ruleId ?? 0, body: ruleBody(form) });
    void navigate({ to: "/santa/rules" });
  }

  function updateInclude(id: number, next: Partial<RuleIncludeForm>) {
    if (next.policy && next.policy !== "cel" && celDialogID === id) {
      setCELDialogID(null);
    }
    setForm({
      ...form,
      includes: form.includes.map((include) => (include.id === id ? { ...include, ...next } : include)),
    });
  }

  return (
    <PageShell asChild>
      <form
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          void submit();
        }}
      >
        <PageHeader title={mode === "create" ? "New Rule" : "Edit Rule"} />

        <FieldGroup>
          <Field data-invalid={showErrors && errors.name ? true : undefined}>
            <FieldLabel htmlFor="santa-rule-name" required>
              Name
            </FieldLabel>
            <Input
              id="santa-rule-name"
              required
              aria-invalid={showErrors && errors.name ? true : undefined}
              value={form.name}
              onChange={(event) => setForm({ ...form, name: event.target.value })}
            />
            {showErrors && errors.name ? <FieldError>{errors.name}</FieldError> : null}
          </Field>
          <Field>
            <FieldLabel htmlFor="santa-rule-description">Description</FieldLabel>
            <Textarea
              id="santa-rule-description"
              rows={3}
              value={form.description}
              onChange={(event) => setForm({ ...form, description: event.target.value })}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="santa-rule-type">Rule Type</FieldLabel>
            <Select
              value={form.rule_type}
              onValueChange={(rule_type) => setForm({ ...form, rule_type: rule_type as RuleType, identifier: "" })}
            >
              <SelectTrigger id="santa-rule-type" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {RULE_TYPE_OPTIONS.map((type) => (
                    <SelectItem key={type.value} value={type.value}>
                      {type.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </Field>
          <RuleTargetPicker
            form={form}
            identifierError={identifierError}
            identifierInvalid={identifierInvalid}
            onChange={setForm}
          />
          <Field>
            <FieldLabel htmlFor="santa-rule-custom-url">Custom URL</FieldLabel>
            <Input
              id="santa-rule-custom-url"
              value={form.custom_url}
              onChange={(event) => setForm({ ...form, custom_url: event.target.value })}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="santa-rule-custom-message">Custom Message</FieldLabel>
            <Textarea
              id="santa-rule-custom-message"
              rows={3}
              value={form.custom_message}
              onChange={(event) => setForm({ ...form, custom_message: event.target.value })}
            />
          </Field>
          <section className="mt-2 flex flex-col gap-4 border-t pt-6">
            <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold">Assignments</h2>
              <p className="text-muted-foreground text-sm">
                Includes are evaluated top to bottom, like configuration order. Higher rows have higher priority.
              </p>
            </div>
            <Field>
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div>
                  <FieldLabel>Include</FieldLabel>
                  <FieldDescription>
                    Pick the labels this rule can match. If a host matches more than one include, the highest matching
                    row sets the policy.
                  </FieldDescription>
                </div>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() =>
                    setForm({
                      ...form,
                      includes: [
                        ...form.includes,
                        { id: Date.now(), policy: "allowlist", cel_expression: "", label_id: null },
                      ],
                    })
                  }
                >
                  <Plus data-icon="inline-start" />
                  Add Include
                </Button>
              </div>
              {form.includes.length === 0 ? (
                <div className="text-muted-foreground rounded-md border border-dashed px-4 py-6 text-sm">
                  No Include Assignments.
                </div>
              ) : (
                <IncludeTargetsTable
                  includes={form.includes}
                  showErrors={showErrors}
                  includeErrors={includeErrors}
                  excludeLabelIDs={form.exclude_label_ids}
                  onChange={(includes) => setForm({ ...form, includes })}
                  onUpdate={updateInclude}
                  onEditCEL={setCELDialogID}
                  onDelete={(id) => {
                    if (celDialogID === id) setCELDialogID(null);
                    setForm({ ...form, includes: form.includes.filter((item) => item.id !== id) });
                  }}
                />
              )}
            </Field>
            <Separator />
            <Field>
              <FieldLabel>Exclude</FieldLabel>
              <LabelPicker
                value={form.exclude_label_ids}
                unavailableLabelIDs={includeLabelIDs}
                onChange={(exclude_label_ids) => setForm({ ...form, exclude_label_ids })}
              />
              <FieldDescription>Hosts with these labels never receive this rule.</FieldDescription>
            </Field>
          </section>
        </FieldGroup>

        <div className="flex items-center gap-2 border-t pt-4">
          <Button type="submit" size="sm" disabled={pending}>
            {pending ? <Loader2 data-icon="inline-start" className="animate-spin" /> : null}
            Save
          </Button>
          <Button asChild type="button" variant="ghost" size="sm">
            <Link to="/santa/rules">Cancel</Link>
          </Button>
        </div>

        <CELDialog
          include={form.includes.find((include) => include.id === celDialogID)}
          error={celDialogID !== null ? includeErrors[celDialogID]?.cel_expression : undefined}
          showRequiredError={showErrors}
          onOpenChange={(open) => {
            if (!open) setCELDialogID(null);
          }}
          onChange={(cel_expression) => {
            if (celDialogID !== null) updateInclude(celDialogID, { cel_expression });
          }}
        />
      </form>
    </PageShell>
  );
}

function RuleTargetPicker({
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

function ruleTargetDescription(target: SantaRuleTarget | null, ruleType: RuleType) {
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

function IncludeTargetsTable({
  includes,
  showErrors,
  includeErrors,
  excludeLabelIDs,
  onChange,
  onUpdate,
  onEditCEL,
  onDelete,
}: {
  includes: RuleIncludeForm[];
  showErrors: boolean;
  includeErrors: Partial<Record<number, RuleIncludeErrors>>;
  excludeLabelIDs: number[];
  onChange: (includes: RuleIncludeForm[]) => void;
  onUpdate: (id: number, include: Partial<RuleIncludeForm>) => void;
  onEditCEL: (id: number) => void;
  onDelete: (id: number) => void;
}) {
  const columns = useMemo<ColumnDef<RuleIncludeForm>[]>(
    () => [
      {
        id: "drag",
        header: () => null,
        enableSorting: false,
        enableHiding: false,
        cell: () => <DataTableRowDragHandle />,
        meta: { headClassName: "w-10", cellClassName: "w-10 align-top pt-3" },
      },
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
                    const nextPolicy = policy as RulePolicy;
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
      {
        id: "labels",
        header: () => (
          <span className="inline-flex items-center gap-1">
            Labels
            <span aria-hidden="true" className="text-destructive">
              *
            </span>
          </span>
        ),
        enableSorting: false,
        cell: ({ row }) => {
          const error = showErrors ? includeErrors[row.original.id]?.label_id : undefined;
          const unavailableLabelIDs = unavailableIncludeLabelIDs(includes, excludeLabelIDs, row.original.id);

          return (
            <Field data-invalid={error ? true : undefined} className="min-w-72 gap-1">
              <LabelPicker
                value={row.original.label_id === null ? [] : [row.original.label_id]}
                selectionMode="single"
                includeBuiltins
                required
                unavailableLabelIDs={unavailableLabelIDs}
                placeholder="Select Label"
                emptyMessage="No Unused Labels Available."
                emptyPlaceholder="No Unused Labels"
                invalid={error ? true : undefined}
                onChange={(label_ids) => onUpdate(row.original.id, { label_id: label_ids[0] ?? null })}
              />
              {error ? <FieldError>{error}</FieldError> : null}
            </Field>
          );
        },
      },
      {
        id: "actions",
        header: () => null,
        enableSorting: false,
        enableHiding: false,
        cell: ({ row }) => (
          <Button type="button" variant="ghost" size="icon" onClick={() => onDelete(row.original.id)}>
            <Trash2 />
          </Button>
        ),
        meta: { headClassName: "w-10", cellClassName: "w-10 align-top pt-3" },
      },
    ],
    [excludeLabelIDs, includeErrors, includes, onDelete, onEditCEL, onUpdate, showErrors],
  );

  return (
    <DataTable
      columns={columns}
      data={includes}
      totalCount={includes.length}
      pagination={{ pageIndex: 0, pageSize: Math.max(includes.length, 1) }}
      sorting={[]}
      onPaginationChange={() => undefined}
      onSortingChange={() => undefined}
      getRowId={(include) => String(include.id)}
      clientSort
      rowReorderDisabled={includes.length <= 1}
      onRowReorder={onChange}
      empty={<span className="text-muted-foreground text-sm">No Include Targets.</span>}
      emptyClassName="min-h-24 items-center py-6"
    />
  );
}

function CELDialog({
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

function formFromSearch(search: Record<string, unknown>): RuleFormState {
  const ruleType = isRuleType(search.rule_type) ? search.rule_type : emptyRuleForm.rule_type;
  return {
    ...emptyRuleForm,
    rule_type: ruleType,
    identifier: typeof search.identifier === "string" ? search.identifier : "",
    name: typeof search.name === "string" ? search.name : "",
    description: "",
  };
}

function isRuleType(value: unknown): value is RuleType {
  return typeof value === "string" && RULE_TYPE_VALUES.includes(value as RuleType);
}

function formFromRule(rule: SantaRule): RuleFormState {
  return {
    rule_type: rule.rule_type,
    identifier: rule.identifier,
    name: rule.name,
    description: rule.description,
    custom_message: rule.custom_message,
    custom_url: rule.custom_url,
    exclude_label_ids: rule.exclude_label_ids ?? [],
    includes: (rule.includes ?? []).map((include) => ({
      id: include.id,
      policy: include.policy,
      cel_expression: include.cel_expression ?? "",
      label_id: include.label_id,
    })),
  };
}

function ruleBody(form: RuleFormState): SantaRuleMutation {
  return {
    rule_type: form.rule_type,
    identifier: form.identifier.trim(),
    name: form.name.trim(),
    description: optionalText(form.description),
    custom_message: optionalText(form.custom_message),
    custom_url: optionalText(form.custom_url),
    exclude_label_ids: form.exclude_label_ids,
    includes: form.includes.map(includeBody),
  };
}

function includeBody(include: RuleIncludeForm) {
  if (include.label_id === null) {
    throw new Error("validated include is missing a label");
  }
  return {
    policy: include.policy,
    cel_expression: include.policy === "cel" ? optionalText(include.cel_expression) : undefined,
    label_id: include.label_id,
  };
}

function includeErrorMap(
  result: RuleFormParse,
  includes: RuleIncludeForm[],
): Partial<Record<number, RuleIncludeErrors>> {
  if (result.success) return {};
  const out: Partial<Record<number, RuleIncludeErrors>> = {};
  for (const issue of result.error.issues) {
    if (issue.path[0] !== "includes") continue;
    const index = issue.path[1];
    if (typeof index !== "number" || index >= includes.length) continue;
    const include = includes[index];
    const entry = out[include.id] ?? {};
    const field = issue.path[2];
    if (field === "cel_expression" && !entry.cel_expression) entry.cel_expression = issue.message;
    if (field === "label_id" && !entry.label_id) entry.label_id = issue.message;
    out[include.id] = entry;
  }
  return out;
}

function selectedIncludeLabelIDs(includes: RuleIncludeForm[]) {
  return includes.flatMap((include) => (include.label_id === null ? [] : [include.label_id]));
}

function unavailableIncludeLabelIDs(includes: RuleIncludeForm[], excludeLabelIDs: number[], currentIncludeID: number) {
  return [
    ...excludeLabelIDs,
    ...includes.flatMap((include) =>
      include.id === currentIncludeID || include.label_id === null ? [] : [include.label_id],
    ),
  ];
}

function ruleIdentifierHint(ruleType: RuleType) {
  return RULE_IDENTIFIER_RULES[ruleType].hint;
}

function identifierErrorFor(result: RuleFormParse): string | undefined {
  if (result.success) return undefined;
  return result.error.issues.find((issue) => issue.path[0] === "identifier")?.message;
}
