import { Link, useNavigate, useParams } from "@tanstack/react-router";
import type { ColumnDef, PaginationState, SortingState } from "@tanstack/react-table";
import type { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { Loader2, ServerCog, UsersRound } from "lucide-react";
import type { ReactNode } from "react";
import { useMemo, useRef, useState } from "react";
import { z } from "zod";

import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableSearch } from "@/components/data-table/data-table-search";
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
import {
  useDirectoryDepartments,
  useDirectoryGroups,
  useDirectoryUsers,
  type DirectoryDepartment,
  type DirectoryGroup,
  type DirectoryUser,
} from "@/hooks/use-directory";
import { useHosts, type Host } from "@/hooks/use-hosts";
import { useCreateLabel, useLabel, useUpdateLabel, type LabelCreate, type LabelMutation } from "@/hooks/use-labels";
import { useSchemaSidebar } from "@/hooks/use-schema-sidebar";
import { cn } from "@/lib/utils";

type MembershipType = "dynamic" | "manual" | "derived";
type DerivedAttribute = "directory_department" | "directory_group" | "directory_user";

const MEMBERSHIP_OPTIONS: { value: MembershipType; label: string; helpText: string }[] = [
  {
    value: "dynamic",
    label: "Dynamic",
    helpText: "osquery evaluates a SQL query on each host. Hosts where the query returns rows match the label.",
  },
  {
    value: "manual",
    label: "Manual",
    helpText: "Membership is managed by the server. Useful for ad-hoc grouping and host targeting.",
  },
  {
    value: "derived",
    label: "Derived",
    helpText: "Membership is computed from non-osquery host attributes such as directory department or group.",
  },
];

const DERIVED_ATTRIBUTE_OPTIONS: { value: DerivedAttribute; label: string }[] = [
  { value: "directory_department", label: "Directory Department" },
  { value: "directory_group", label: "Directory Group" },
  { value: "directory_user", label: "Directory User" },
];

interface FormState {
  name: string;
  description: string;
  query: string;
  host_ids: number[];
  derived_attribute: DerivedAttribute;
  derived_values: string[];
  label_membership_type: MembershipType;
}

const empty: FormState = {
  name: "",
  description: "",
  query: "select 1 from os_version where major >= 13;",
  host_ids: [],
  derived_attribute: "directory_department",
  derived_values: [],
  label_membership_type: "dynamic",
};

const labelFormSchema = z
  .object({
    name: z.string().trim().min(1, "Name is required."),
    description: z.string().trim(),
    query: z.string().trim(),
    host_ids: z.array(z.number().int().positive()),
    derived_attribute: z.enum(["directory_department", "directory_group", "directory_user"]),
    derived_values: z.array(z.string().trim().min(1)),
    label_membership_type: z.enum(["dynamic", "manual", "derived"]),
  })
  .refine((value) => value.label_membership_type !== "dynamic" || value.query !== "", {
    message: "Dynamic labels need a query.",
    path: ["query"],
  })
  .refine((value) => value.label_membership_type !== "derived" || value.derived_values.length > 0, {
    message: "Derived labels need at least one selected item.",
    path: ["derived_values"],
  });

type LabelFormParse = ReturnType<typeof labelFormSchema.safeParse>;

export function LabelEditPage({ mode }: { mode: "create" | "edit" }) {
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

export function LabelCreatePage() {
  return <LabelEditPage mode="create" />;
}

export function LabelUpdatePage() {
  return <LabelEditPage mode="edit" />;
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
  const editorRef = useRef<ReactCodeMirrorRef>(null);

  const pending = createLabel.isPending || updateLabel.isPending;
  const isDynamic = form.label_membership_type === "dynamic";
  const isManual = form.label_membership_type === "manual";
  const isDerived = form.label_membership_type === "derived";
  const memberOption = MEMBERSHIP_OPTIONS.find((o) => o.value === form.label_membership_type);
  const parsed = useMemo(() => labelFormSchema.safeParse(form), [form]);
  const errors = useMemo(() => fieldErrors(parsed), [parsed]);

  async function submit() {
    if (!parsed.success) {
      setShowErrors(true);
      return;
    }
    const cleaned = parsed.data;
    const body: LabelCreate | LabelMutation = {
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

  return (
    <PageShell
      asChild
      className={cn("h-full transition-[padding] duration-200 ease-out", isDynamic && schemaOpen && "pr-[21rem]")}
    >
      <form
        onSubmit={(event) => {
          event.preventDefault();
          void submit();
        }}
      >
        <PageHeader title={mode === "create" ? "New Label" : "Edit Label"} />

        <FieldGroup className="max-w-5xl">
          <Field>
            <FieldLabel htmlFor="label-name">Name</FieldLabel>
            <Input
              id="label-name"
              required
              value={form.name}
              onChange={(event) => setForm({ ...form, name: event.target.value })}
            />
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
                if (value) setForm({ ...form, label_membership_type: value as MembershipType });
              }}
              variant="outline"
              size="sm"
              className="flex-wrap"
            >
              {MEMBERSHIP_OPTIONS.map((option) => (
                <ToggleGroupItem key={option.value} value={option.value}>
                  {option.label}
                </ToggleGroupItem>
              ))}
            </ToggleGroup>
            {memberOption ? <FieldDescription>{memberOption.helpText}</FieldDescription> : null}
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
                <FieldLabel>{derivedSelectorLabel(form.derived_attribute)}</FieldLabel>
                <DerivedSelector
                  attribute={form.derived_attribute}
                  value={form.derived_values}
                  onChange={(derived_values) => setForm({ ...form, derived_values })}
                />
                <FieldDescription>Matches linked directory records.</FieldDescription>
                {showErrors && errors.derived_values ? <FieldError>{errors.derived_values}</FieldError> : null}
              </Field>
            </FieldGroup>
          ) : null}
        </FieldGroup>

        {isDynamic ? (
          <div className="max-w-3xl">
            <SQLEditor
              ref={editorRef}
              value={form.query}
              onChange={(query) => setForm({ ...form, query })}
              placeholder="SELECT ..."
            />
            {showErrors && errors.query ? <FieldError className="mt-2">{errors.query}</FieldError> : null}
          </div>
        ) : null}

        {isDynamic ? (
          <SchemaSidebar open={schemaOpen} onOpenChange={setSchemaOpen} onInsertColumn={insertAtCursor} />
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
  return host.display_name || host.hostname || host.computer_name || host.hardware_uuid;
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
        accessorKey: "display_name",
        header: ({ column }) => <DataTableColumnHeader column={column} title="Host" />,
        cell: ({ row }) => (
          <div className="min-w-0">
            <div className="truncate font-medium">{hostName(row.original)}</div>
            <div className="text-muted-foreground truncate text-xs">{row.original.hostname || "No Hostname"}</div>
          </div>
        ),
      },
      {
        accessorKey: "hardware_serial",
        header: ({ column }) => <DataTableColumnHeader column={column} title="Serial" />,
        cell: ({ row }) => (
          <span className="text-muted-foreground truncate">
            {row.original.hardware_serial || row.original.hardware_uuid}
          </span>
        ),
      },
      {
        accessorKey: "hardware_model",
        header: ({ column }) => <DataTableColumnHeader column={column} title="Model" />,
        cell: ({ row }) => (
          <span className="text-muted-foreground truncate">{row.original.hardware_model || "Unknown"}</span>
        ),
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
      return <DirectoryGroupSelector value={value} onChange={onChange} />;
    case "directory_user":
      return <DirectoryUserSelector value={value} onChange={onChange} />;
    default:
      return <DepartmentSelector value={value} onChange={onChange} />;
  }
}

function DepartmentSelector({ value, onChange }: { value: string[]; onChange: (value: string[]) => void }) {
  const controls = useSelectorControls([{ id: "value", desc: false }]);
  const showSelected = controls.scope === "selected";
  const departments = useDirectoryDepartments({
    q: controls.q,
    page_index: controls.pagination.pageIndex,
    page_size: controls.pagination.pageSize,
    sort: sortParam(controls.sorting),
    values: showSelected ? value : undefined,
  });
  const rows = showSelected && value.length === 0 ? [] : (departments.data?.items ?? []);
  const count = showSelected && value.length === 0 ? 0 : (departments.data?.count ?? 0);
  const columns = useMemo<ColumnDef<DirectoryDepartment>[]>(
    () => [
      {
        accessorKey: "value",
        header: ({ column }) => <DataTableColumnHeader column={column} title="Department" />,
        cell: ({ row }) => <span className="font-medium">{row.original.value}</span>,
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
        showSelected
          ? "Selected departments will appear here."
          : "Try another search term or sync directory users first."
      }
      emptyIcon={<UsersRound className="size-5" />}
    />
  );
}

function DirectoryGroupSelector({ value, onChange }: { value: string[]; onChange: (value: string[]) => void }) {
  const controls = useSelectorControls([{ id: "display_name", desc: false }]);
  const showSelected = controls.scope === "selected";
  const groups = useDirectoryGroups({
    q: controls.q,
    page_index: controls.pagination.pageIndex,
    page_size: controls.pagination.pageSize,
    sort: sortParam(controls.sorting),
    values: showSelected ? value : undefined,
  });
  const rows = showSelected && value.length === 0 ? [] : (groups.data?.items ?? []);
  const count = showSelected && value.length === 0 ? 0 : (groups.data?.count ?? 0);
  const columns = useMemo<ColumnDef<DirectoryGroup>[]>(
    () => [
      {
        accessorKey: "display_name",
        header: ({ column }) => <DataTableColumnHeader column={column} title="Group" />,
        cell: ({ row }) => (
          <div className="min-w-0">
            <div className="truncate font-medium">{row.original.display_name}</div>
            <div className="text-muted-foreground truncate text-xs">{row.original.external_id}</div>
          </div>
        ),
      },
      {
        accessorKey: "mail_nickname",
        header: ({ column }) => <DataTableColumnHeader column={column} title="Nickname" />,
        cell: ({ row }) => (
          <span className="text-muted-foreground truncate">{row.original.mail_nickname ?? "None"}</span>
        ),
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

function DirectoryUserSelector({ value, onChange }: { value: string[]; onChange: (value: string[]) => void }) {
  const controls = useSelectorControls([{ id: "display_name", desc: false }]);
  const showSelected = controls.scope === "selected";
  const users = useDirectoryUsers({
    q: controls.q,
    page_index: controls.pagination.pageIndex,
    page_size: controls.pagination.pageSize,
    sort: sortParam(controls.sorting),
    values: showSelected ? value : undefined,
  });
  const rows = showSelected && value.length === 0 ? [] : (users.data?.items ?? []);
  const count = showSelected && value.length === 0 ? 0 : (users.data?.count ?? 0);
  const columns = useMemo<ColumnDef<DirectoryUser>[]>(
    () => [
      {
        accessorKey: "display_name",
        header: ({ column }) => <DataTableColumnHeader column={column} title="User" />,
        cell: ({ row }) => (
          <div className="min-w-0">
            <div className="truncate font-medium">{row.original.display_name}</div>
            <div className="text-muted-foreground truncate text-xs">{row.original.user_principal_name}</div>
          </div>
        ),
      },
      {
        accessorKey: "department",
        header: ({ column }) => <DataTableColumnHeader column={column} title="Department" />,
        cell: ({ row }) => <span className="text-muted-foreground truncate">{row.original.department ?? "None"}</span>,
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
      getRowId={(user) => user.external_id}
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
    case "directory_user":
      return "Users";
    default:
      return "Departments";
  }
}

function fieldErrors(result: LabelFormParse): Record<string, string> {
  if (result.success) return {};
  const out: Record<string, string> = {};
  for (const issue of result.error.issues) {
    const key = issue.path[0];
    if (typeof key === "string" && !out[key]) out[key] = issue.message;
  }
  return out;
}

function membershipFromString(value: string | undefined): MembershipType {
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
    case "directory_user":
    case "directory_department":
      return value;
    default:
      return "directory_department";
  }
}
