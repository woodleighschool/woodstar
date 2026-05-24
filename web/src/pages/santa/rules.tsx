import { Link, useNavigate, useParams, useSearch } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { ListChecks, Loader2, Plus, Trash2 } from "lucide-react";
import { useState } from "react";

import { BulkDeleteDialog } from "@/components/data-table/bulk-delete-dialog";
import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableFacetedFilter } from "@/components/data-table/data-table-faceted-filter";
import { DataTableSearch } from "@/components/data-table/data-table-search";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { LabelPicker } from "@/components/santa/label-picker";
import { SortableList, type SortableItem } from "@/components/santa/sortable-list";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Field, FieldDescription, FieldGroup, FieldLabel, FieldLegend, FieldSet } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import {
  useBulkDeleteSantaRules,
  useCreateSantaRule,
  useSantaRule,
  useSantaRules,
  useUpdateSantaRule,
  type SantaRule,
  type SantaRuleMutation,
} from "@/hooks/use-santa";
import { tableQueryParams, useTablePaginationParams } from "@/hooks/use-table-pagination-params";

const RULE_TYPES = [
  { value: "binary", label: "Binary" },
  { value: "certificate", label: "Certificate" },
  { value: "teamid", label: "Team ID" },
  { value: "signingid", label: "Signing ID" },
  { value: "cdhash", label: "CDHash" },
] as const satisfies readonly { value: SantaRuleMutation["rule_type"]; label: string }[];

const POLICIES = [
  { value: "allowlist", label: "Allowlist" },
  { value: "allowlist_compiler", label: "Compiler allowlist" },
  { value: "blocklist", label: "Blocklist" },
  { value: "silent_blocklist", label: "Silent blocklist" },
  { value: "cel", label: "CEL" },
] as const;

type RuleType = (typeof RULE_TYPES)[number]["value"];
type RulePolicy = (typeof POLICIES)[number]["value"];

const RULE_IDENTIFIER_RULES: Record<RuleType, { pattern: RegExp; hint: string }> = {
  binary: {
    pattern: /^[0-9a-fA-F]{64}$/,
    hint: "Use a 64 character SHA-256 hex hash.",
  },
  certificate: {
    pattern: /^[0-9a-fA-F]{64}$/,
    hint: "Use a 64 character certificate SHA-256 hex fingerprint.",
  },
  cdhash: {
    pattern: /^[0-9a-fA-F]{40}$/,
    hint: "Use a 40 character CDHash hex value.",
  },
  signingid: {
    pattern: /^(?:[A-Z0-9]{10}|platform):[a-zA-Z0-9.-]+$/,
    hint: "Use TEAMID:bundle.identifier or platform:bundle.identifier.",
  },
  teamid: {
    pattern: /^[A-Z0-9]{10}$/,
    hint: "Use a 10 character uppercase Team ID.",
  },
};

interface RuleIncludeForm extends SortableItem {
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

export function SantaRulesPage() {
  const search = useSearch({ strict: false });
  const navigate = useNavigate();
  const { state, setters } = useTablePaginationParams();
  const [draft, setDraft] = useDebouncedSearchParam("q");
  const looseSearch = search as Record<string, unknown>;
  const ruleType = typeof looseSearch.rule_type === "string" ? looseSearch.rule_type : undefined;
  const query = useSantaRules({
    q: typeof search.q === "string" ? search.q : undefined,
    rule_type: ruleType,
    ...tableQueryParams(state),
  });
  const bulkDelete = useBulkDeleteSantaRules();
  const [selectedRuleIds, setSelectedRuleIds] = useState<string[]>([]);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const rows = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const hasFilters = !!search.q || !!ruleType;
  const selectedIDs = selectedRuleIds.map(Number);

  function deleteSelectedRules() {
    bulkDelete.mutate(selectedIDs, {
      onSuccess: () => {
        setSelectedRuleIds([]);
        setDeleteOpen(false);
      },
    });
  }

  const columns: ColumnDef<SantaRule>[] = [
    {
      id: "name",
      accessorKey: "name",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
      cell: ({ row }) => (
        <div className="grid gap-1">
          <span className="font-medium">{row.original.name || row.original.identifier}</span>
          <span className="text-muted-foreground truncate font-mono text-xs">{row.original.identifier}</span>
        </div>
      ),
    },
    {
      id: "rule_type",
      accessorKey: "rule_type",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Rule type" />,
      cell: ({ row }) => <Badge variant="secondary">{ruleTypeLabel(row.original.rule_type)}</Badge>,
    },
    {
      id: "includes",
      header: "Targets",
      enableSorting: false,
      cell: ({ row }) =>
        row.original.includes?.length ? (
          <span className="text-muted-foreground text-sm tabular-nums">
            {row.original.includes.length} include{row.original.includes.length === 1 ? "" : "s"}
          </span>
        ) : (
          <Badge variant="outline">inactive</Badge>
        ),
    },
  ];

  return (
    <PageShell>
      <PageHeader
        title="Santa rules"
        description="Manage execution rules and ordered include targets."
        actions={
          <Button asChild size="sm">
            <Link to="/santa/rules/new">
              <Plus data-icon="inline-start" />
              Add rule
            </Link>
          </Button>
        }
      />

      {query.error ? (
        <Alert variant="destructive">
          <AlertTitle>Failed to load rules</AlertTitle>
          <AlertDescription>{query.error.message}</AlertDescription>
        </Alert>
      ) : (
        <DataTable
          columns={columns}
          data={rows}
          totalCount={totalCount}
          pagination={state.pagination}
          sorting={state.sorting}
          onPaginationChange={setters.setPagination}
          onSortingChange={setters.setSorting}
          isLoading={query.isLoading}
          enableRowSelection
          selectedRowIds={selectedRuleIds}
          onSelectedRowIdsChange={setSelectedRuleIds}
          bulkActions={
            <Button variant="destructive" size="sm" onClick={() => setDeleteOpen(true)} disabled={bulkDelete.isPending}>
              <Trash2 data-icon="inline-start" />
              Delete
            </Button>
          }
          rowHref={(row) => ({ to: "/santa/rules/$ruleId/edit", params: { ruleId: String(row.id) } })}
          toolbar={
            <div className="flex flex-wrap items-center gap-2">
              <DataTableSearch value={draft} onChange={setDraft} placeholder="Search rules" label="Search rules" />
              <DataTableFacetedFilter
                title="Rule type"
                selected={ruleType ? [ruleType] : []}
                options={[...RULE_TYPES]}
                singleSelect
                onChange={(next) =>
                  void navigate({
                    search: ((prev: Record<string, unknown>) => ({
                      ...prev,
                      rule_type: next[0],
                      page_index: undefined,
                    })) as never,
                    replace: true,
                  })
                }
              />
            </div>
          }
          empty={
            <Empty>
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <ListChecks />
                </EmptyMedia>
                <EmptyTitle>{hasFilters ? "No matches" : "No rules yet"}</EmptyTitle>
                <EmptyDescription>
                  {hasFilters
                    ? "No Santa rules matched the current filters."
                    : "Add a Santa rule to make it effective for matching labels."}
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
          }
        />
      )}

      <BulkDeleteDialog
        open={deleteOpen}
        onOpenChange={(open) => {
          if (!open) bulkDelete.reset();
          setDeleteOpen(open);
        }}
        count={selectedIDs.length}
        noun="rule"
        description="Deleted rules stop syncing to Santa clients."
        error={bulkDelete.error?.message}
        pending={bulkDelete.isPending}
        onConfirm={deleteSelectedRules}
      />
    </PageShell>
  );
}

export function SantaRuleEditPage({ mode }: { mode: "create" | "edit" }) {
  const params = useParams({ strict: false });
  const ruleId = params.ruleId ?? "";
  const detail = useSantaRule(ruleId);

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

  return <RuleForm key={ruleId || "new"} mode={mode} ruleId={ruleId} initial={initial} />;
}

function RuleForm({ mode, ruleId, initial }: { mode: "create" | "edit"; ruleId: string; initial: RuleFormState }) {
  const navigate = useNavigate();
  const create = useCreateSantaRule();
  const update = useUpdateSantaRule();
  const [form, setForm] = useState<RuleFormState>(initial);
  const pending = create.isPending || update.isPending;
  const error = create.error ?? update.error;
  const identifierError = ruleIdentifierError(form.rule_type, form.identifier);
  const identifierInvalid = form.identifier.trim() !== "" && identifierError !== null;

  async function submit() {
    if (!canSaveRule(form)) return;
    if (mode === "create") await create.mutateAsync(ruleBody(form));
    else await update.mutateAsync({ id: Number(ruleId), body: ruleBody(form) });
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

        {error ? (
          <Alert variant="destructive">
            <AlertTitle>Unable to save rule</AlertTitle>
            <AlertDescription>{error.message}</AlertDescription>
          </Alert>
        ) : null}

        <FieldGroup className="max-w-5xl">
          <FieldSet>
            <FieldLegend>Identity</FieldLegend>
            <div className="grid gap-4 md:grid-cols-2">
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
              <Field className="md:col-span-2">
                <FieldLabel htmlFor="santa-rule-custom-message">Custom message</FieldLabel>
                <Textarea
                  id="santa-rule-custom-message"
                  rows={3}
                  value={form.custom_message}
                  onChange={(event) => setForm({ ...form, custom_message: event.target.value })}
                />
              </Field>
            </div>
          </FieldSet>

          <FieldSet>
            <FieldLegend>Targeting</FieldLegend>
            <Field>
              <FieldLabel>Exclude labels</FieldLabel>
              <LabelPicker
                value={form.exclude_label_ids}
                onChange={(exclude_label_ids) => setForm({ ...form, exclude_label_ids })}
              />
              <FieldDescription>
                Excluded labels suppress this rule even when an include target matches.
              </FieldDescription>
            </Field>
            <Field>
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div>
                  <FieldLabel>Include targets</FieldLabel>
                  <FieldDescription>Rules without include targets are saved but not effective.</FieldDescription>
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
                  No include targets.
                </div>
              ) : (
                <SortableList
                  items={form.includes}
                  onChange={(includes) => setForm({ ...form, includes })}
                  renderItem={(include) => (
                    <IncludeEditor
                      include={include}
                      onChange={(next) => updateInclude(include.id, next)}
                      onDelete={() =>
                        setForm({ ...form, includes: form.includes.filter((item) => item.id !== include.id) })
                      }
                    />
                  )}
                />
              )}
            </Field>
          </FieldSet>
        </FieldGroup>

        <div className="flex max-w-5xl items-center gap-2 border-t pt-4">
          <Button type="submit" size="sm" disabled={pending || !canSaveRule(form)}>
            {pending ? <Loader2 data-icon="inline-start" className="animate-spin" /> : null}
            Save
          </Button>
          <Button asChild type="button" variant="ghost" size="sm">
            <Link to="/santa/rules">Cancel</Link>
          </Button>
        </div>
      </form>
    </PageShell>
  );
}

function IncludeEditor({
  include,
  onChange,
  onDelete,
}: {
  include: RuleIncludeForm;
  onChange: (include: Partial<RuleIncludeForm>) => void;
  onDelete: () => void;
}) {
  return (
    <div className="grid gap-4 rounded-md border p-4">
      <div className="flex flex-wrap items-start gap-3">
        <Field className="min-w-56 flex-1">
          <FieldLabel htmlFor={`include-policy-${include.id}`}>Policy</FieldLabel>
          <Select value={include.policy} onValueChange={(policy) => onChange({ policy: policy as RulePolicy })}>
            <SelectTrigger id={`include-policy-${include.id}`} className="w-full">
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
        </Field>
        <Button type="button" variant="ghost" size="icon" onClick={onDelete}>
          <Trash2 />
        </Button>
      </div>
      {include.policy === "cel" ? (
        <Field>
          <FieldLabel htmlFor={`include-cel-${include.id}`}>CEL expression</FieldLabel>
          <Textarea
            id={`include-cel-${include.id}`}
            rows={3}
            value={include.cel_expression}
            onChange={(event) => onChange({ cel_expression: event.target.value })}
          />
        </Field>
      ) : null}
      <Field>
        <FieldLabel>Labels</FieldLabel>
        <LabelPicker value={include.label_ids} onChange={(label_ids) => onChange({ label_ids })} />
      </Field>
    </div>
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
    cel_expression: optionalText(include.cel_expression),
    label_ids: include.label_ids,
  };
}

function optionalText(value: string) {
  const trimmed = value.trim();
  return trimmed === "" ? undefined : trimmed;
}

function canSaveRule(form: RuleFormState) {
  return (
    form.rule_type.trim() !== "" &&
    form.identifier.trim() !== "" &&
    ruleIdentifierError(form.rule_type, form.identifier) === null
  );
}

function ruleTypeLabel(ruleType: string) {
  return RULE_TYPES.find((type) => type.value === ruleType)?.label ?? ruleType;
}

function ruleIdentifierHint(ruleType: RuleType) {
  return RULE_IDENTIFIER_RULES[ruleType].hint;
}

function ruleIdentifierError(ruleType: RuleType, identifier: string) {
  const trimmed = identifier.trim();
  if (trimmed === "") return null;
  const rule = RULE_IDENTIFIER_RULES[ruleType];
  return rule.pattern.test(trimmed) ? null : rule.hint;
}
