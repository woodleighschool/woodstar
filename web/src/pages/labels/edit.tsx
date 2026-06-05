import { Link, useNavigate, useParams } from "@tanstack/react-router";
import type { ColumnDef, PaginationState, SortingState } from "@tanstack/react-table";
import type { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { Loader2, ServerCog, UsersRound } from "lucide-react";
import type { ReactNode } from "react";
import { useCallback, useMemo, useRef, useState } from "react";
import { z } from "zod";

import { DataTable, DataTableColumnHeader, DataTableSearch } from "@/components/data-table";
import { SchemaSidebar } from "@/components/editor/schema-sidebar";
import { SQLEditor } from "@/components/editor/sql-editor";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { useGroups, type Group } from "@/hooks/use-groups";
import { useHosts, type Host } from "@/hooks/use-hosts";
import { useCreateLabel, useLabel, useUpdateLabel, type LabelMutation } from "@/hooks/use-labels";
import { useSchemaSidebar } from "@/hooks/use-schema-sidebar";
import { useUserDepartments, useUsers, type Department, type User } from "@/hooks/use-users";
import { fieldErrors, requiredString, selectedIDArray } from "@/lib/form-validation";
import { sqlSyntaxError } from "@/lib/sql-validation";
import { cn } from "@/lib/utils";
import {
  LABEL_MEMBERSHIP_OPTIONS,
  LABEL_MEMBERSHIP_TYPES,
  LABEL_MEMBERSHIP_VALUES,
  type LabelMembershipType,
} from "@/pages/labels/shared";

type DerivedAttribute = "user_department" | "directory_group" | "user";

const DERIVED_ATTRIBUTE_OPTIONS: { value: DerivedAttribute; label: string }[] = [
  { value: "user_department", label: "User Department" },
  { value: "directory_group", label: "Group" },
  { value: "user", label: "User" },
];

interface FormState {
  name: string;
  description: string;
  query: string;
  host_ids: number[];
  derived_attribute: DerivedAttribute;
  derived_values: string[];
  label_membership_type: LabelMembershipType;
}

const empty: FormState = {
  name: "",
  description: "",
  query: "select 1 from os_version where major >= 13;",
  host_ids: [],
  derived_attribute: "user_department",
  derived_values: [],
  label_membership_type: "dynamic",
};

const queryRequiredSchema = requiredString("Query");

const labelFormSchema = z
  .object({
    name: requiredString("Name"),
    description: z.string().trim(),
    query: z.string().trim(),
    host_ids: selectedIDArray("Host"),
    derived_attribute: z.enum(["user_department", "directory_group", "user"]),
    derived_values: z.array(requiredString("Derived value")),
    label_membership_type: z.enum(LABEL_MEMBERSHIP_VALUES),
  })
  .superRefine((value, ctx) => {
    if (value.label_membership_type === "dynamic") {
      const query = queryRequiredSchema.safeParse(value.query);
      if (!query.success) {
        ctx.addIssue({ code: "custom", message: query.error.issues[0]?.message ?? "Invalid query.", path: ["query"] });
      } else {
        const syntaxError = sqlSyntaxError(value.query);
        if (syntaxError) {
          ctx.addIssue({ code: "custom", message: syntaxError, path: ["query"] });
        }
      }
    }
    if (value.label_membership_type === "derived" && value.derived_values.length === 0) {
      ctx.addIssue({
        code: "custom",
        message: "Derived labels need at least one selected item.",
        path: ["derived_values"],
      });
    }
  });

export function LabelMutationPage({ mode }: { mode: "create" | "edit" }) {
  const params = useParams({ strict: false });
  const labelId = params.labelId ?? "";
  const labelID = mode === "edit" ? Number(labelId) : null;
  const detail = useLabel(labelID);

  if (mode === "edit") {
    if (detail.error) {
      return (
        <PageShell>
          <Alert variant="destructive">
            <AlertTitle>Failed to Load Label</AlertTitle>
            <AlertDescription>{detail.error.message}</AlertDescription>
          </Alert>
        </PageShell>
      );
    }
    if (!detail.data) {
      return (
        <PageShell className="text-muted-foreground flex-row items-center gap-2 text-sm">
          <Loader2 className="size-4 animate-spin" /> Loading Label...
        </PageShell>
      );
    }
    if (detail.data.label_type === "builtin") {
      return (
        <PageShell>
          <Alert>
            <AlertTitle>Built-In Label</AlertTitle>
            <AlertDescription>Built-in labels are managed by Woodstar and cannot be edited.</AlertDescription>
          </Alert>
        </PageShell>
      );
    }
  }

  const initial: FormState =
    mode === "edit" && detail.data
      ? {
          name: detail.data.name,
          description: detail.data.description,
          query: detail.data.query ?? empty.query,
          host_ids: detail.data.host_ids ?? [],
          derived_attribute: derivedAttributeFromString(detail.data.criteria?.attribute),
          derived_values: detail.data.criteria?.values ?? [],
          label_membership_type: membershipFromString(detail.data.label_membership_type),
        }
      : empty;

  return <LabelEditForm key={labelId || "new"} mode={mode} labelId={labelID} initial={initial} />;
}

export function LabelNewPage() {
  return <LabelMutationPage mode="create" />;
}

export function LabelEditPage() {
  return <LabelMutationPage mode="edit" />;
}

function LabelEditForm({
  mode,
  labelId,
  initial,
}: {
  mode: "create" | "edit";
  labelId: number | null;
  initial: FormState;
}) {
  const navigate = useNavigate();
  const createLabel = useCreateLabel();
  const updateLabel = useUpdateLabel(labelId);
  const [form, setForm] = useState<FormState>(initial);
  const [showErrors, setShowErrors] = useState(false);
  const [schemaOpen, setSchemaOpen] = useSchemaSidebar();
  const [selectedSchemaTable, setSelectedSchemaTable] = useState<string | null>(null);
  const editorRef = useRef<ReactCodeMirrorRef>(null);

  const pending = createLabel.isPending || updateLabel.isPending;
  const isDynamic = form.label_membership_type === "dynamic";
  const isManual = form.label_membership_type === "manual";
  const isDerived = form.label_membership_type === "derived";
  const memberOption = LABEL_MEMBERSHIP_TYPES[form.label_membership_type];
  const parsed = useMemo(() => labelFormSchema.safeParse(form), [form]);
  const errors = useMemo(() => fieldErrors(parsed), [parsed]);

  async function submit() {
    if (!parsed.success) {
      setShowErrors(true);
      return;
    }
    const cleaned = parsed.data;
    const body: LabelMutation = {
      name: cleaned.name,
      description: cleaned.description,
      label_membership_type: cleaned.label_membership_type,
      query: cleaned.label_membership_type === "dynamic" ? cleaned.query : undefined,
      host_ids: cleaned.label_membership_type === "manual" ? cleaned.host_ids : undefined,
      criteria:
        cleaned.label_membership_type === "derived"
          ? { attribute: cleaned.derived_attribute, values: cleaned.derived_values }
          : undefined,
    };
    if (mode === "create") {
      await createLabel.mutateAsync(body);
    } else {
      await updateLabel.mutateAsync(body);
    }
    void navigate({ to: "/labels" });
  }

  function insertAtCursor(snippet: string) {
    const view = editorRef.current?.view;
    if (!view) {
      setForm((prev) => ({ ...prev, query: prev.query + " " + snippet }));
      return;
    }
    view.dispatch({ changes: { from: view.state.selection.main.from, insert: snippet } });
  }

  const selectSchemaTable = useCallback(
    (tableName: string) => {
      setSelectedSchemaTable(tableName);
      setSchemaOpen(true);
    },
    [setSchemaOpen],
  );

  return (
    <PageShell
      asChild
      className={cn("h-full transition-[padding] duration-200 ease-out", isDynamic && schemaOpen && "pr-[21rem]")}
    >
      <form
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          void submit();
        }}
      >
        <PageHeader title={mode === "create" ? "New Label" : "Edit Label"} />

        <FieldGroup className="max-w-5xl">
          <Field data-invalid={showErrors && errors.name ? true : undefined}>
            <FieldLabel htmlFor="label-name" required>
              Name
            </FieldLabel>
            <Input
              id="label-name"
              required
              aria-invalid={showErrors && errors.name ? true : undefined}
              value={form.name}
              onChange={(event) => setForm({ ...form, name: event.target.value })}
            />
            {showErrors && errors.name ? <FieldError>{errors.name}</FieldError> : null}
          </Field>

          <Field>
            <FieldLabel htmlFor="label-description">Description</FieldLabel>
            <Textarea
              id="label-description"
              rows={3}
              placeholder="Why this label exists"
              value={form.description}
              onChange={(event) => setForm({ ...form, description: event.target.value })}
            />
          </Field>

          <Field>
            <FieldLabel>Type</FieldLabel>
            <ToggleGroup
              type="single"
              value={form.label_membership_type}
              onValueChange={(value) => {
                if (value) setForm({ ...form, label_membership_type: value as LabelMembershipType });
              }}
              variant="outline"
              size="sm"
              className="flex-wrap"
            >
              {LABEL_MEMBERSHIP_OPTIONS.map((option) => (
                <ToggleGroupItem key={option.value} value={option.value}>
                  {option.label}
                </ToggleGroupItem>
              ))}
            </ToggleGroup>
            {memberOption.description ? <FieldDescription>{memberOption.description}</FieldDescription> : null}
          </Field>

          {isManual ? (
            <Field data-invalid={showErrors && errors.host_ids ? true : undefined}>
              <FieldLabel>Hosts</FieldLabel>
              <HostSelector value={form.host_ids} onChange={(host_ids) => setForm({ ...form, host_ids })} />
              {showErrors && errors.host_ids ? <FieldError>{errors.host_ids}</FieldError> : null}
            </Field>
          ) : null}

          {isDerived ? (
            <FieldGroup>
              <Field>
                <FieldLabel htmlFor="label-derived-attribute">Attribute</FieldLabel>
                <Select
                  value={form.derived_attribute}
                  onValueChange={(value) =>
                    setForm({ ...form, derived_attribute: value as DerivedAttribute, derived_values: [] })
                  }
                >
                  <SelectTrigger id="label-derived-attribute" className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      {DERIVED_ATTRIBUTE_OPTIONS.map((option) => (
                        <SelectItem key={option.value} value={option.value}>
                          {option.label}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </Field>
              <Field data-invalid={showErrors && errors.derived_values ? true : undefined}>
                <FieldLabel required>{derivedSelectorLabel(form.derived_attribute)}</FieldLabel>
                <DerivedSelector
                  attribute={form.derived_attribute}
                  value={form.derived_values}
                  onChange={(derived_values) => setForm({ ...form, derived_values })}
                />
                <FieldDescription>Matches linked users and groups.</FieldDescription>
                {showErrors && errors.derived_values ? <FieldError>{errors.derived_values}</FieldError> : null}
              </Field>
            </FieldGroup>
          ) : null}
        </FieldGroup>

        {isDynamic ? (
          <Field data-invalid={showErrors && errors.query ? true : undefined} className="max-w-3xl">
            <FieldLabel required>Query</FieldLabel>
            <SQLEditor
              ref={editorRef}
              value={form.query}
              onChange={(query) => setForm({ ...form, query })}
              onTableMetaClick={selectSchemaTable}
              placeholder="SELECT ..."
              invalid={showErrors && errors.query ? true : undefined}
            />
            {showErrors && errors.query ? <FieldError>{errors.query}</FieldError> : null}
          </Field>
        ) : null}

        {isDynamic ? (
          <SchemaSidebar
            open={schemaOpen}
            onOpenChange={setSchemaOpen}
            onInsertColumn={insertAtCursor}
            selectedTable={selectedSchemaTable}
            onSelectedTableChange={setSelectedSchemaTable}
          />
        ) : null}

        <div className="flex max-w-5xl items-center gap-2 border-t pt-4">
          <Button type="submit" size="sm" disabled={pending}>
            {pending ? "Saving..." : "Save"}
          </Button>
          {mode === "edit" ? (
            <Button asChild type="button" variant="ghost" size="sm">
              <Link to="/labels">Cancel</Link>
            </Button>
          ) : null}
        </div>
      </form>
    </PageShell>
  );
}

function hostName(host: Host) {
  return host.display_name;
}

function HostSelector({ value, onChange }: { value: number[]; onChange: (value: number[]) => void }) {
  const controls = useSelectorControls([{ id: "display_name", desc: false }]);
  const showSelected = controls.scope === "selected";
  const hosts = useHosts({
    q: controls.q,
    page_index: controls.pagination.pageIndex,
    page_size: controls.pagination.pageSize,
    sort: sortParam(controls.sorting),
    ids: showSelected ? value : undefined,
  });
  const rows = showSelected && value.length === 0 ? [] : (hosts.data?.items ?? []);
  const count = showSelected && value.length === 0 ? 0 : (hosts.data?.count ?? 0);
  const columns = useMemo<ColumnDef<Host>[]>(
    () => [
      {
        id: "display_name",
        accessorFn: (row) => row.display_name,
        header: ({ column }) => <DataTableColumnHeader column={column} title="Host" />,
        cell: ({ row }) => hostName(row.original),
      },
      {
        id: "hardware.serial",
        accessorFn: (row) => row.hardware.serial,
        header: ({ column }) => <DataTableColumnHeader column={column} title="Serial" />,
        cell: ({ row }) => row.original.hardware.serial,
      },
      {
        id: "hardware.model_identifier",
        accessorFn: (row) => row.hardware.model_identifier,
        header: ({ column }) => <DataTableColumnHeader column={column} title="Model" />,
        cell: ({ row }) => row.original.hardware.model_identifier || "Unknown",
      },
    ],
    [],
  );

  return (
    <SelectorTable
      columns={columns}
      data={rows}
      totalCount={count}
      pagination={controls.pagination}
      sorting={controls.sorting}
      onPaginationChange={controls.setPagination}
      onSortingChange={controls.setSorting}
      searchValue={controls.q}
      searchPlaceholder="Search Hosts"
      scope={controls.scope}
      selectedCount={value.length}
      isLoading={hosts.isLoading}
      error={hosts.error?.message}
      selectedRowIds={value.map(String)}
      onSearchChange={controls.setSearch}
      onScopeChange={controls.setScopeFilter}
      onSelectedRowIdsChange={(ids) => onChange(ids.map(Number).filter((id) => Number.isInteger(id) && id > 0))}
      getRowId={(host) => String(host.id)}
      emptyTitle={showSelected ? "No Selected Hosts" : "No Hosts Found"}
      emptyDescription={
        showSelected ? "Selected hosts will appear here." : "Try another search term or enroll hosts first."
      }
      emptyIcon={<ServerCog className="size-5" />}
    />
  );
}

function DerivedSelector({
  attribute,
  value,
  onChange,
}: {
  attribute: DerivedAttribute;
  value: string[];
  onChange: (value: string[]) => void;
}) {
  switch (attribute) {
    case "directory_group":
      return <GroupSelector value={value} onChange={onChange} />;
    case "user":
      return <UserSelector value={value} onChange={onChange} />;
    default:
      return <DepartmentSelector value={value} onChange={onChange} />;
  }
}

function DepartmentSelector({ value, onChange }: { value: string[]; onChange: (value: string[]) => void }) {
  const controls = useSelectorControls([{ id: "value", desc: false }]);
  const showSelected = controls.scope === "selected";
  const departments = useUserDepartments({
    q: controls.q,
    page_index: controls.pagination.pageIndex,
    page_size: controls.pagination.pageSize,
    sort: sortParam(controls.sorting),
    values: showSelected ? value : undefined,
  });
  const rows = showSelected && value.length === 0 ? [] : (departments.data?.items ?? []);
  const count = showSelected && value.length === 0 ? 0 : (departments.data?.count ?? 0);
  const columns = useMemo<ColumnDef<Department>[]>(
    () => [
      {
        accessorKey: "value",
        header: ({ column }) => <DataTableColumnHeader column={column} title="Department" />,
        cell: ({ row }) => row.original.value,
      },
    ],
    [],
  );

  return (
    <SelectorTable
      columns={columns}
      data={rows}
      totalCount={count}
      pagination={controls.pagination}
      sorting={controls.sorting}
      onPaginationChange={controls.setPagination}
      onSortingChange={controls.setSorting}
      searchValue={controls.q}
      searchPlaceholder="Search Departments"
      scope={controls.scope}
      selectedCount={value.length}
      isLoading={departments.isLoading}
      error={departments.error?.message}
      selectedRowIds={value}
      onSearchChange={controls.setSearch}
      onScopeChange={controls.setScopeFilter}
      onSelectedRowIdsChange={onChange}
      getRowId={(department) => department.value}
      emptyTitle={showSelected ? "No Selected Departments" : "No Departments Found"}
      emptyDescription={
        showSelected ? "Selected departments will appear here." : "Try another search term or sync users first."
      }
      emptyIcon={<UsersRound className="size-5" />}
    />
  );
}

function GroupSelector({ value, onChange }: { value: string[]; onChange: (value: string[]) => void }) {
  const controls = useSelectorControls([{ id: "display_name", desc: false }]);
  const showSelected = controls.scope === "selected";
  const groups = useGroups({
    q: controls.q,
    page_index: controls.pagination.pageIndex,
    page_size: controls.pagination.pageSize,
    sort: sortParam(controls.sorting),
    values: showSelected ? value : undefined,
  });
  const rows = showSelected && value.length === 0 ? [] : (groups.data?.items ?? []);
  const count = showSelected && value.length === 0 ? 0 : (groups.data?.count ?? 0);
  const columns = useMemo<ColumnDef<Group>[]>(
    () => [
      {
        accessorKey: "display_name",
        header: ({ column }) => <DataTableColumnHeader column={column} title="Group" />,
        cell: ({ row }) => row.original.display_name,
      },
      {
        accessorKey: "mail_nickname",
        header: ({ column }) => <DataTableColumnHeader column={column} title="Nickname" />,
        cell: ({ row }) => row.original.mail_nickname ?? "None",
      },
    ],
    [],
  );

  return (
    <SelectorTable
      columns={columns}
      data={rows}
      totalCount={count}
      pagination={controls.pagination}
      sorting={controls.sorting}
      onPaginationChange={controls.setPagination}
      onSortingChange={controls.setSorting}
      searchValue={controls.q}
      searchPlaceholder="Search Groups"
      scope={controls.scope}
      selectedCount={value.length}
      isLoading={groups.isLoading}
      error={groups.error?.message}
      selectedRowIds={value}
      onSearchChange={controls.setSearch}
      onScopeChange={controls.setScopeFilter}
      onSelectedRowIdsChange={onChange}
      getRowId={(group) => group.external_id}
      emptyTitle={showSelected ? "No Selected Groups" : "No Groups Found"}
      emptyDescription={
        showSelected ? "Selected groups will appear here." : "Try another search term or sync groups first."
      }
      emptyIcon={<UsersRound className="size-5" />}
    />
  );
}

function UserSelector({ value, onChange }: { value: string[]; onChange: (value: string[]) => void }) {
  const controls = useSelectorControls([{ id: "name", desc: false }]);
  const showSelected = controls.scope === "selected";
  const users = useUsers({
    q: controls.q,
    page_index: controls.pagination.pageIndex,
    page_size: controls.pagination.pageSize,
    sort: sortParam(controls.sorting),
    values: showSelected ? value : undefined,
  });
  const rows = showSelected && value.length === 0 ? [] : (users.data?.items ?? []);
  const count = showSelected && value.length === 0 ? 0 : (users.data?.count ?? 0);
  const columns = useMemo<ColumnDef<User>[]>(
    () => [
      {
        accessorKey: "name",
        header: ({ column }) => <DataTableColumnHeader column={column} title="User" />,
        cell: ({ row }) => row.original.name,
      },
      {
        accessorKey: "department",
        header: ({ column }) => <DataTableColumnHeader column={column} title="Department" />,
        cell: ({ row }) => row.original.department ?? "None",
      },
    ],
    [],
  );

  return (
    <SelectorTable
      columns={columns}
      data={rows}
      totalCount={count}
      pagination={controls.pagination}
      sorting={controls.sorting}
      onPaginationChange={controls.setPagination}
      onSortingChange={controls.setSorting}
      searchValue={controls.q}
      searchPlaceholder="Search Users"
      scope={controls.scope}
      selectedCount={value.length}
      isLoading={users.isLoading}
      error={users.error?.message}
      selectedRowIds={value}
      onSearchChange={controls.setSearch}
      onScopeChange={controls.setScopeFilter}
      onSelectedRowIdsChange={onChange}
      getRowId={(user) => String(user.id)}
      emptyTitle={showSelected ? "No Selected Users" : "No Users Found"}
      emptyDescription={
        showSelected ? "Selected users will appear here." : "Try another search term or sync users first."
      }
      emptyIcon={<UsersRound className="size-5" />}
    />
  );
}

type SelectionScope = "all" | "selected";

function useSelectorControls(defaultSorting: SortingState) {
  const [q, setQ] = useState("");
  const [scope, setScope] = useState<SelectionScope>("all");
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 10 });
  const [sorting, setSorting] = useState<SortingState>(defaultSorting);

  function resetPage() {
    setPagination((prev) => ({ ...prev, pageIndex: 0 }));
  }

  const setSearch = (next: string) => {
    setQ(next);
    resetPage();
  };
  const setScopeFilter = (next: SelectionScope) => {
    setScope(next);
    resetPage();
  };

  return {
    q,
    scope,
    pagination,
    sorting,
    setPagination,
    setSorting,
    setSearch,
    setScopeFilter,
  };
}

interface SelectorTableProps<TData> {
  columns: ColumnDef<TData>[];
  data: TData[];
  totalCount: number;
  pagination: PaginationState;
  sorting: SortingState;
  onPaginationChange: (pagination: PaginationState | ((pagination: PaginationState) => PaginationState)) => void;
  onSortingChange: (sorting: SortingState | ((sorting: SortingState) => SortingState)) => void;
  searchValue: string;
  searchPlaceholder: string;
  scope: SelectionScope;
  selectedCount: number;
  isLoading: boolean;
  error?: string;
  selectedRowIds: string[];
  onSearchChange: (value: string) => void;
  onScopeChange: (value: SelectionScope) => void;
  onSelectedRowIdsChange: (ids: string[]) => void;
  getRowId: (row: TData) => string;
  emptyTitle: string;
  emptyDescription: string;
  emptyIcon: ReactNode;
}

function SelectorTable<TData>({
  columns,
  data,
  totalCount,
  pagination,
  sorting,
  onPaginationChange,
  onSortingChange,
  searchValue,
  searchPlaceholder,
  scope,
  selectedCount,
  isLoading,
  error,
  selectedRowIds,
  onSearchChange,
  onScopeChange,
  onSelectedRowIdsChange,
  getRowId,
  emptyTitle,
  emptyDescription,
  emptyIcon,
}: SelectorTableProps<TData>) {
  if (error) {
    return <p className="text-destructive text-sm">{error}</p>;
  }

  return (
    <DataTable
      columns={columns}
      data={data}
      totalCount={totalCount}
      pagination={pagination}
      sorting={sorting}
      onPaginationChange={onPaginationChange}
      onSortingChange={onSortingChange}
      isLoading={isLoading}
      enableRowSelection
      selectedRowIds={selectedRowIds}
      onSelectedRowIdsChange={onSelectedRowIdsChange}
      getRowId={getRowId}
      perPageOptions={[10, 25, 50]}
      toolbar={
        <div className="flex flex-wrap items-center gap-2">
          <DataTableSearch
            value={searchValue}
            onChange={onSearchChange}
            placeholder={searchPlaceholder}
            className="min-w-64"
          />
          <ToggleGroup
            type="single"
            value={scope}
            onValueChange={(next) => {
              if (next === "all" || next === "selected") onScopeChange(next);
            }}
            variant="outline"
            size="sm"
          >
            <ToggleGroupItem value="all">Show All</ToggleGroupItem>
            <ToggleGroupItem value="selected">Selected {selectedCount}</ToggleGroupItem>
          </ToggleGroup>
        </div>
      }
      empty={
        <Empty>
          <EmptyHeader>
            <EmptyMedia variant="icon">{emptyIcon}</EmptyMedia>
            <EmptyTitle>{emptyTitle}</EmptyTitle>
            <EmptyDescription>{emptyDescription}</EmptyDescription>
          </EmptyHeader>
        </Empty>
      }
    />
  );
}

function sortParam(sorting: SortingState) {
  if (sorting.length === 0) return undefined;
  const first = sorting[0];
  return `${first.id}.${first.desc ? "desc" : "asc"}`;
}

function derivedSelectorLabel(attribute: DerivedAttribute) {
  switch (attribute) {
    case "directory_group":
      return "Groups";
    case "user":
      return "Users";
    default:
      return "Departments";
  }
}

function membershipFromString(value: string | undefined): LabelMembershipType {
  switch (value) {
    case "manual":
    case "derived":
      return value;
    default:
      return "dynamic";
  }
}

function derivedAttributeFromString(value: string | undefined): DerivedAttribute {
  switch (value) {
    case "directory_group":
    case "user":
    case "user_department":
      return value;
    default:
      return "user_department";
  }
}
