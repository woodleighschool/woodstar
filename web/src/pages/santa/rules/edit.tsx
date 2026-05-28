import { Link, useNavigate, useParams } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Code2, Loader2, Plus, Trash2 } from "lucide-react";
import { useMemo, useState } from "react";
import { z } from "zod";

import { DataTable, DataTableRowDragHandle } from "@/components/data-table/data-table";
import { CodeEditor } from "@/components/editor/code-editor";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { LabelPicker } from "@/components/santa/label-picker";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Textarea } from "@/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  useCreateSantaRule,
  useSantaRule,
  useUpdateSantaRule,
  type SantaRule,
  type SantaRuleMutation,
} from "@/hooks/use-santa";

import { RULE_IDENTIFIER_RULES, RULE_TYPES, type RuleType } from "./shared";

const POLICIES = [
  { value: "allowlist", label: "Allowlist" },
  { value: "allowlist_compiler", label: "Compiler allowlist" },
  { value: "blocklist", label: "Blocklist" },
  { value: "silent_blocklist", label: "Silent blocklist" },
  { value: "cel", label: "CEL" },
] as const;

type RulePolicy = (typeof POLICIES)[number]["value"];

const ruleFormSchema = z
  .object({
    rule_type: z.enum(["binary", "certificate", "teamid", "signingid", "cdhash"]),
    identifier: z.string().trim(),
    name: z.string().trim(),
    custom_message: z.string().trim(),
    custom_url: z.string().trim(),
    exclude_label_ids: z.array(z.number().int().positive()),
    includes: z.array(
      z
        .object({
          id: z.number(),
          policy: z.enum(["allowlist", "allowlist_compiler", "blocklist", "silent_blocklist", "cel"]),
          cel_expression: z.string().trim(),
          label_ids: z.array(z.number().int().positive()).min(1, "Pick at least one label."),
        })
        .refine((value) => value.policy !== "cel" || value.cel_expression !== "", {
          message: "CEL policy requires an expression.",
          path: ["cel_expression"],
        }),
    ),
  })
  .superRefine((value, ctx) => {
    // Empty identifier is the HTML `required` attribute's job; only check pattern when filled.
    if (value.identifier === "") return;
    const rule = RULE_IDENTIFIER_RULES[value.rule_type];
    if (!rule.pattern.test(value.identifier)) {
      ctx.addIssue({ code: "custom", message: rule.hint, path: ["identifier"] });
    }
  });

type RuleFormParse = ReturnType<typeof ruleFormSchema.safeParse>;
type RuleIncludeErrors = { cel_expression?: string; label_ids?: string };

interface RuleIncludeForm {
  id: number;
  policy: RulePolicy;
  cel_expression: string;
  label_ids: number[];
}

interface RuleFormState {
  rule_type: RuleType;
  identifier: string;
  name: string;
  custom_message: string;
  custom_url: string;
  exclude_label_ids: number[];
  includes: RuleIncludeForm[];
}

const emptyRuleForm: RuleFormState = {
  rule_type: "binary",
  identifier: "",
  name: "",
  custom_message: "",
  custom_url: "",
  exclude_label_ids: [],
  includes: [],
};

export function SantaRuleEditPage({ mode }: { mode: "create" | "edit" }) {
  const params = useParams({ strict: false });
  const ruleId = params.ruleId ?? "";
  const ruleID = mode === "edit" ? Number(ruleId) : null;
  const detail = useSantaRule(ruleID);

  if (mode === "edit") {
    if (detail.error) {
      return (
        <PageShell>
          <Alert variant="destructive">
            <AlertTitle>Failed to load rule</AlertTitle>
            <AlertDescription>{detail.error.message}</AlertDescription>
          </Alert>
        </PageShell>
      );
    }
    if (!detail.data) {
      return (
        <PageShell className="text-muted-foreground flex-row items-center gap-2 text-sm">
          <Loader2 className="animate-spin" /> Loading rule...
        </PageShell>
      );
    }
  }

  const initial = mode === "edit" && detail.data ? formFromRule(detail.data) : emptyRuleForm;

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
  const identifierError = identifierErrorFor(parsed);
  const identifierInvalid = form.identifier.trim() !== "" && identifierError !== undefined;
  const includeErrors = useMemo(() => includeErrorMap(parsed, form.includes), [parsed, form.includes]);
  const canSave = parsed.success && form.identifier.trim() !== "";

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
    setForm({
      ...form,
      includes: form.includes.map((include) => (include.id === id ? { ...include, ...next } : include)),
    });
  }

  return (
    <PageShell asChild>
      <form
        onSubmit={(event) => {
          event.preventDefault();
          void submit();
        }}
      >
        <PageHeader
          title={mode === "create" ? "New Santa rule" : "Edit Santa rule"}
          description="Define the Santa rule identity, policy targets, and user-facing block text."
        />

        <FieldGroup className="max-w-5xl">
          <Field>
            <FieldLabel htmlFor="santa-rule-type">Rule type</FieldLabel>
            <Select
              value={form.rule_type}
              onValueChange={(rule_type) => setForm({ ...form, rule_type: rule_type as RuleType })}
            >
              <SelectTrigger id="santa-rule-type" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {RULE_TYPES.map((type) => (
                    <SelectItem key={type.value} value={type.value}>
                      {type.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </Field>
          <Field data-invalid={identifierInvalid}>
            <FieldLabel htmlFor="santa-rule-identifier">Identifier</FieldLabel>
            <Input
              id="santa-rule-identifier"
              required
              aria-invalid={identifierInvalid}
              value={form.identifier}
              onChange={(event) => setForm({ ...form, identifier: event.target.value })}
            />
            <FieldDescription>{identifierError ?? ruleIdentifierHint(form.rule_type)}</FieldDescription>
          </Field>
          <Field>
            <FieldLabel htmlFor="santa-rule-name">Name</FieldLabel>
            <Input
              id="santa-rule-name"
              value={form.name}
              onChange={(event) => setForm({ ...form, name: event.target.value })}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="santa-rule-custom-url">Custom URL</FieldLabel>
            <Input
              id="santa-rule-custom-url"
              value={form.custom_url}
              onChange={(event) => setForm({ ...form, custom_url: event.target.value })}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="santa-rule-custom-message">Custom message</FieldLabel>
            <Textarea
              id="santa-rule-custom-message"
              rows={3}
              value={form.custom_message}
              onChange={(event) => setForm({ ...form, custom_message: event.target.value })}
            />
          </Field>
          <div className="flex flex-col gap-4">
            <div className="flex flex-col gap-1">
              <h2 className="text-sm font-medium">Assignments</h2>
              <p className="text-muted-foreground text-sm">
                Include labels make this rule effective; exclude labels suppress matching assignments.
              </p>
            </div>
            <Field>
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div>
                  <FieldLabel>Include</FieldLabel>
                  <FieldDescription>Rules without include assignments are saved but not effective.</FieldDescription>
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
                        { id: Date.now(), policy: "allowlist", cel_expression: "", label_ids: [] },
                      ],
                    })
                  }
                >
                  <Plus data-icon="inline-start" />
                  Add include
                </Button>
              </div>
              {form.includes.length === 0 ? (
                <div className="text-muted-foreground rounded-md border border-dashed px-4 py-6 text-sm">
                  No include assignments.
                </div>
              ) : (
                <IncludeTargetsTable
                  includes={form.includes}
                  showErrors={showErrors}
                  includeErrors={includeErrors}
                  onChange={(includes) => setForm({ ...form, includes })}
                  onUpdate={updateInclude}
                  onEditCEL={setCELDialogID}
                  onDelete={(id) => setForm({ ...form, includes: form.includes.filter((item) => item.id !== id) })}
                />
              )}
            </Field>
            <Separator />
            <Field>
              <FieldLabel>Exclude</FieldLabel>
              <LabelPicker
                value={form.exclude_label_ids}
                onChange={(exclude_label_ids) => setForm({ ...form, exclude_label_ids })}
              />
              <FieldDescription>
                Hosts in these labels are excluded even when an include assignment matches.
              </FieldDescription>
            </Field>
          </div>
        </FieldGroup>

        <div className="flex max-w-5xl items-center gap-2 border-t pt-4">
          <Button type="submit" size="sm" disabled={pending || !canSave}>
            {pending ? <Loader2 data-icon="inline-start" className="animate-spin" /> : null}
            Save
          </Button>
          <Button asChild type="button" variant="ghost" size="sm">
            <Link to="/santa/rules">Cancel</Link>
          </Button>
        </div>

        <CELDialog
          include={form.includes.find((include) => include.id === celDialogID)}
          error={celDialogID !== null && showErrors ? includeErrors[celDialogID]?.cel_expression : undefined}
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

function IncludeTargetsTable({
  includes,
  showErrors,
  includeErrors,
  onChange,
  onUpdate,
  onEditCEL,
  onDelete,
}: {
  includes: RuleIncludeForm[];
  showErrors: boolean;
  includeErrors: Partial<Record<number, RuleIncludeErrors>>;
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
        cell: () => <DataTableRowDragHandle label="Reorder include target" />,
        meta: { headClassName: "w-10", cellClassName: "w-10 align-top pt-3" },
      },
      {
        id: "policy",
        header: "Policy",
        enableSorting: false,
        cell: ({ row }) => {
          const hasExpression = row.original.cel_expression.trim() !== "";
          const celLabel = hasExpression ? "Edit CEL expression" : "Add CEL expression";
          const celInvalid = row.original.policy === "cel" && !hasExpression;

          return (
            <Field className="min-w-56 gap-1">
              <FieldLabel className="sr-only" htmlFor={`include-policy-${row.original.id}`}>
                Policy
              </FieldLabel>
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
                      {POLICIES.map((policy) => (
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
                        variant={celInvalid ? "destructive" : "outline"}
                        size="icon-sm"
                        aria-label={celLabel}
                        aria-invalid={celInvalid}
                        onClick={() => onEditCEL(row.original.id)}
                      >
                        <Code2 />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>{celLabel}</TooltipContent>
                  </Tooltip>
                ) : null}
              </div>
            </Field>
          );
        },
      },
      {
        id: "labels",
        header: "Labels",
        enableSorting: false,
        cell: ({ row }) => {
          const error = showErrors ? includeErrors[row.original.id]?.label_ids : undefined;

          return (
            <Field data-invalid={error ? true : undefined} className="min-w-72 gap-1">
              <FieldLabel className="sr-only">Labels</FieldLabel>
              <LabelPicker
                value={row.original.label_ids}
                onChange={(label_ids) => onUpdate(row.original.id, { label_ids })}
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
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label="Delete include assignment"
            onClick={() => onDelete(row.original.id)}
          >
            <Trash2 />
          </Button>
        ),
        meta: { headClassName: "w-10", cellClassName: "w-10 align-top pt-3" },
      },
    ],
    [includeErrors, onDelete, onEditCEL, onUpdate, showErrors],
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
      empty={<span className="text-muted-foreground text-sm">No include targets.</span>}
      emptyClassName="min-h-24 items-center py-6"
    />
  );
}

function CELDialog({
  include,
  error,
  onOpenChange,
  onChange,
}: {
  include?: RuleIncludeForm;
  error?: string;
  onOpenChange: (open: boolean) => void;
  onChange: (cel_expression: string) => void;
}) {
  return (
    <Dialog open={include !== undefined} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>CEL expression</DialogTitle>
          <DialogDescription>Edit the CEL predicate for this include assignment.</DialogDescription>
        </DialogHeader>

        {include ? (
          <Field data-invalid={error ? true : undefined}>
            <FieldLabel>Expression</FieldLabel>
            <CodeEditor
              value={include.cel_expression}
              onChange={onChange}
              placeholder="target.signing_time >= timestamp('2025-05-31T00:00:00Z')"
              className="[&_.cm-content]:min-h-40"
            />
            {error ? <FieldError>{error}</FieldError> : null}
          </Field>
        ) : null}

        <DialogFooter>
          <Button type="button" size="sm" onClick={() => onOpenChange(false)}>
            Done
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function formFromRule(rule: SantaRule): RuleFormState {
  return {
    rule_type: rule.rule_type,
    identifier: rule.identifier,
    name: rule.name,
    custom_message: rule.custom_message,
    custom_url: rule.custom_url,
    exclude_label_ids: rule.exclude_label_ids ?? [],
    includes: (rule.includes ?? []).map((include) => ({
      id: include.id,
      policy: include.policy,
      cel_expression: include.cel_expression ?? "",
      label_ids: include.label_ids ?? [],
    })),
  };
}

function ruleBody(form: RuleFormState): SantaRuleMutation {
  return {
    rule_type: form.rule_type,
    identifier: form.identifier.trim(),
    name: optionalText(form.name),
    custom_message: optionalText(form.custom_message),
    custom_url: optionalText(form.custom_url),
    exclude_label_ids: form.exclude_label_ids,
    includes: form.includes.map(includeBody),
  };
}

function includeBody(include: RuleIncludeForm) {
  return {
    policy: include.policy,
    cel_expression: include.policy === "cel" ? optionalText(include.cel_expression) : undefined,
    label_ids: include.label_ids,
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
    if (field === "label_ids" && !entry.label_ids) entry.label_ids = issue.message;
    out[include.id] = entry;
  }
  return out;
}

function optionalText(value: string) {
  const trimmed = value.trim();
  return trimmed === "" ? undefined : trimmed;
}

function ruleIdentifierHint(ruleType: RuleType) {
  return RULE_IDENTIFIER_RULES[ruleType].hint;
}

function identifierErrorFor(result: RuleFormParse): string | undefined {
  if (result.success) return undefined;
  return result.error.issues.find((issue) => issue.path[0] === "identifier")?.message;
}
