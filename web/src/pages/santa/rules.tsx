import { useNavigate, useSearch } from "@tanstack/react-router";
import { ListChecks, Loader2, MoreHorizontal, Plus, Trash2 } from "lucide-react";
import type { ReactNode } from "react";
import { useState } from "react";

import { DataTableFacetedFilter } from "@/components/data-table/data-table-faceted-filter";
import { DataTableSearch } from "@/components/data-table/data-table-search";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { LabelPicker } from "@/components/santa/label-picker";
import { SortableList, type SortableItem } from "@/components/santa/sortable-list";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import {
  useCreateSantaRule,
  useDeleteSantaRule,
  useSantaRules,
  useUpdateSantaRule,
  type SantaRule,
  type SantaRuleMutation,
  type SantaRuleUpdate,
} from "@/hooks/use-santa";

const RULE_TYPES = [
  { value: "binary", label: "Binary" },
  { value: "certificate", label: "Certificate" },
  { value: "teamid", label: "Team ID" },
  { value: "signingid", label: "Signing ID" },
  { value: "cdhash", label: "CDHash" },
] as const;

const POLICIES = ["allowlist", "allowlist_compiler", "blocklist", "silent_blocklist", "cel"] as const;

interface RuleIncludeForm extends SortableItem {
  policy: string;
  cel_expression: string;
  label_ids: number[];
}

interface RuleFormState {
  rule_type: string;
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
  const [draft, setDraft] = useDebouncedSearchParam("q");
  const looseSearch = search as Record<string, unknown>;
  const ruleType = typeof looseSearch.rule_type === "string" ? looseSearch.rule_type : undefined;
  const query = useSantaRules({
    q: typeof search.q === "string" ? search.q : undefined,
    rule_type: ruleType,
    per_page: 500,
  });
  const create = useCreateSantaRule();
  const update = useUpdateSantaRule();
  const remove = useDeleteSantaRule();
  const [editing, setEditing] = useState<SantaRule | "new" | null>(null);
  const rows = query.data?.items ?? [];

  return (
    <PageShell>
      <PageHeader
        title="Santa rules"
        description="Manage execution rules and ordered include targets."
        actions={
          <Button size="sm" onClick={() => setEditing("new")}>
            <Plus data-icon="inline-start" />
            Add rule
          </Button>
        }
      />

      <div className="flex flex-wrap items-center gap-2">
        <DataTableSearch value={draft} onChange={setDraft} placeholder="Search rules" label="Search rules" />
        <DataTableFacetedFilter
          title="Rule type"
          selected={ruleType ? [ruleType] : []}
          options={[...RULE_TYPES]}
          singleSelect
          onChange={(next) =>
            void navigate({
              search: ((prev: Record<string, unknown>) => ({ ...prev, rule_type: next[0], page: undefined })) as never,
              replace: true,
            })
          }
        />
      </div>

      {query.error ? (
        <Alert variant="destructive">
          <AlertTitle>Failed to load rules</AlertTitle>
          <AlertDescription>{query.error.message}</AlertDescription>
        </Alert>
      ) : query.isLoading ? (
        <div className="text-muted-foreground flex items-center gap-2 text-sm">
          <Loader2 className="size-4 animate-spin" /> Loading...
        </div>
      ) : rows.length === 0 ? (
        <Empty>
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <ListChecks />
            </EmptyMedia>
            <EmptyTitle>No rules</EmptyTitle>
            <EmptyDescription>Add a Santa rule to make it effective for matching labels.</EmptyDescription>
          </EmptyHeader>
        </Empty>
      ) : (
        <div className="rounded-md border">
          <div className="divide-y">
            {rows.map((rule) => (
              <RuleRow
                key={rule.id}
                rule={rule}
                pending={remove.isPending}
                onEdit={() => setEditing(rule)}
                onDelete={() => remove.mutate(rule.id)}
              />
            ))}
          </div>
        </div>
      )}

      <RuleDialog
        key={editing === "new" ? "new" : (editing?.id ?? "closed")}
        open={editing !== null}
        rule={editing === "new" ? null : editing}
        pending={create.isPending || update.isPending}
        error={create.error?.message ?? update.error?.message}
        onOpenChange={(open) => {
          if (!open) {
            create.reset();
            update.reset();
            setEditing(null);
          }
        }}
        onSubmit={async (body) => {
          if (editing === "new") await create.mutateAsync(body as SantaRuleMutation);
          else if (editing) await update.mutateAsync({ id: editing.id, body });
          setEditing(null);
        }}
      />
    </PageShell>
  );
}

function RuleRow({
  rule,
  pending,
  onEdit,
  onDelete,
}: {
  rule: SantaRule;
  pending: boolean;
  onEdit: () => void;
  onDelete: () => void;
}) {
  return (
    <div className="grid gap-3 p-3 md:grid-cols-[1fr_auto] md:items-center">
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-2">
          <span className="font-medium">{rule.name || rule.identifier}</span>
          <Badge variant="secondary">{rule.rule_type}</Badge>
          {rule.includes?.length ? (
            <span className="text-muted-foreground text-xs tabular-nums">{rule.includes.length} includes</span>
          ) : (
            <Badge variant="outline">inactive</Badge>
          )}
        </div>
        <div className="text-muted-foreground truncate font-mono text-xs">{rule.identifier}</div>
      </div>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button type="button" variant="ghost" size="icon" disabled={pending}>
            <MoreHorizontal />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuGroup>
            <DropdownMenuItem onSelect={onEdit}>Edit</DropdownMenuItem>
            <DropdownMenuItem variant="destructive" onSelect={onDelete}>
              Delete
            </DropdownMenuItem>
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

function RuleDialog({
  open,
  rule,
  pending,
  error,
  onOpenChange,
  onSubmit,
}: {
  open: boolean;
  rule: SantaRule | null;
  pending: boolean;
  error?: string;
  onOpenChange: (open: boolean) => void;
  onSubmit: (body: SantaRuleMutation | SantaRuleUpdate) => Promise<void>;
}) {
  const [form, setForm] = useState(() => formFromRule(rule));

  async function submit() {
    await onSubmit(rule ? ruleUpdateBody(form) : ruleCreateBody(form));
  }

  function updateInclude(id: number, next: Partial<RuleIncludeForm>) {
    setForm({
      ...form,
      includes: form.includes.map((include) => (include.id === id ? { ...include, ...next } : include)),
    });
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>{rule ? "Edit Santa rule" : "New Santa rule"}</DialogTitle>
        </DialogHeader>
        {error ? (
          <Alert variant="destructive">
            <AlertTitle>Unable to save rule</AlertTitle>
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        ) : null}
        <div className="grid gap-4 md:grid-cols-2">
          <Field label="Rule type">
            <Select
              value={form.rule_type}
              disabled={rule !== null}
              onValueChange={(rule_type) => setForm({ ...form, rule_type })}
            >
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {RULE_TYPES.map((type) => (
                  <SelectItem key={type.value} value={type.value}>
                    {type.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>
          <Field label="Identifier">
            <Input
              disabled={rule !== null}
              value={form.identifier}
              onChange={(event) => setForm({ ...form, identifier: event.target.value })}
            />
          </Field>
          <Field label="Name">
            <Input value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} />
          </Field>
          <Field label="Custom URL">
            <Input value={form.custom_url} onChange={(event) => setForm({ ...form, custom_url: event.target.value })} />
          </Field>
          <div className="grid gap-2 md:col-span-2">
            <Label>Custom message</Label>
            <Textarea
              rows={3}
              value={form.custom_message}
              onChange={(event) => setForm({ ...form, custom_message: event.target.value })}
            />
          </div>
          <div className="grid gap-2 md:col-span-2">
            <Label>Exclude labels</Label>
            <LabelPicker
              value={form.exclude_label_ids}
              onChange={(exclude_label_ids) => setForm({ ...form, exclude_label_ids })}
            />
          </div>
          <div className="grid gap-2 md:col-span-2">
            <div className="flex items-center justify-between gap-2">
              <Label>Include targets</Label>
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
              <p className="text-muted-foreground text-sm">
                Rules without include targets are saved but not effective.
              </p>
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
          </div>
        </div>
        <DialogFooter>
          <Button type="button" variant="ghost" size="sm" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            type="button"
            size="sm"
            disabled={pending || form.rule_type.trim() === "" || form.identifier.trim() === ""}
            onClick={() => void submit()}
          >
            {pending ? <Loader2 data-icon="inline-start" className="animate-spin" /> : null}
            Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
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
    <div className="grid gap-3">
      <div className="flex flex-wrap items-center gap-2">
        <Select value={include.policy} onValueChange={(policy) => onChange({ policy })}>
          <SelectTrigger className="w-56">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {POLICIES.map((policy) => (
              <SelectItem key={policy} value={policy}>
                {policy}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button type="button" variant="ghost" size="icon" onClick={onDelete}>
          <Trash2 />
        </Button>
      </div>
      {include.policy === "cel" ? (
        <Textarea
          rows={3}
          placeholder="CEL expression"
          value={include.cel_expression}
          onChange={(event) => onChange({ cel_expression: event.target.value })}
        />
      ) : null}
      <LabelPicker value={include.label_ids} onChange={(label_ids) => onChange({ label_ids })} />
    </div>
  );
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="grid gap-2">
      <Label>{label}</Label>
      {children}
    </div>
  );
}

function formFromRule(rule: SantaRule | null): RuleFormState {
  if (!rule) return { ...emptyRuleForm, includes: [] };
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

function ruleCreateBody(form: RuleFormState): SantaRuleMutation {
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

function ruleUpdateBody(form: RuleFormState): SantaRuleUpdate {
  return {
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
