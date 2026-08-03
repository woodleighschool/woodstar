import { getRouteApi } from "@tanstack/react-router";
import type { CellContext, ColumnDef } from "@tanstack/react-table";
import { ListChecks, MoreHorizontal, Plus } from "lucide-react";
import { useMemo, useState } from "react";

import { BulkDeleteActionBar } from "@components/bulk-delete-action-bar";
import { DataTable } from "@components/data-table/data-table";
import { DataTableEmpty } from "@components/data-table/data-table-empty";
import { DataTableFacetedFilter } from "@components/data-table/data-table-faceted-filter";
import { DataTableSearchInput } from "@components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@components/data-table/data-table-skeleton";
import { selectColumn } from "@components/data-table/select-column";
import { useDataTable } from "@components/data-table/use-data-table";
import { useDataTableSearch } from "@components/data-table/use-data-table-search";
import { EnumBadge } from "@components/enum-badge";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { QueryError } from "@components/query-error";
import { Button } from "@components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@components/ui/dropdown-menu";
import { useAuth } from "@features/auth/queries";
import type { SantaRule } from "@lib/api";
import { DEFAULT_PAGE_SIZE } from "@lib/pagination";
import { formatRelative } from "@lib/utils";

import { RuleDeleteDialog } from "./delete-dialog";
import { RULE_TYPES, RULE_TYPE_OPTIONS } from "./metadata";
import { useBulkDeleteSantaRules, useSantaRules } from "./queries";

const routeApi = getRouteApi("/_authenticated/santa/rules/");
const RULE_TYPE_FILTER_KEYS = [{ id: "rule_type", multiple: true }] as const;

interface RuleTableRow {
  id: number;
  rule: SantaRule;
  onDelete: (rule: SantaRule) => void;
}

function RuleNameCell({ row }: CellContext<RuleTableRow, unknown>) {
  return (
    <Link
      to="/santa/rules/$id"
      params={{ id: String(row.original.rule.id) }}
      className="font-medium"
    >
      {row.original.rule.name}
    </Link>
  );
}

function RuleActionsCell({ row }: CellContext<RuleTableRow, unknown>) {
  return <RuleRowActions rule={row.original.rule} onDelete={row.original.onDelete} />;
}

function ruleColumns(isAdmin: boolean): ColumnDef<RuleTableRow>[] {
  return [
    ...(isAdmin ? [selectColumn<RuleTableRow>()] : []),
    {
      id: "name",
      accessorFn: (row) => row.rule.name,
      header: "Name",
      cell: RuleNameCell,
      enableHiding: false,
      meta: { label: "Name" },
    },
    {
      id: "rule_type",
      accessorFn: (row) => row.rule.rule_type,
      header: "Rule Type",
      cell: ({ row }) => <EnumBadge value={row.original.rule.rule_type} metadata={RULE_TYPES} />,
      meta: { label: "Rule Type", options: RULE_TYPE_OPTIONS },
      enableColumnFilter: true,
    },
    {
      id: "identifier",
      accessorFn: (row) => row.rule.identifier,
      header: "Identifier",
      cell: ({ row }) => row.original.rule.identifier || "-",
      meta: { label: "Identifier" },
    },
    {
      id: "updated_at",
      accessorFn: (row) => row.rule.updated_at,
      header: "Updated",
      cell: ({ row }) => formatRelative(row.original.rule.updated_at),
      meta: { label: "Updated" },
    },
    ...(isAdmin
      ? [
          {
            id: "actions",
            header: () => null,
            enableSorting: false,
            enableHiding: false,
            size: 44,
            minSize: 44,
            maxSize: 44,
            enableResizing: false,
            cell: RuleActionsCell,
          } satisfies ColumnDef<RuleTableRow>,
        ]
      : []),
  ];
}

export function RuleListPage() {
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const tableSearch = useDataTableSearch({
    search,
    onSearchChange: (updater) => void navigate({ search: updater, replace: true }),
    filterKeys: RULE_TYPE_FILTER_KEYS,
  });
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const [deleting, setDeleting] = useState<SantaRule | null>(null);
  const ruleType = search.rule_type;
  const query = useSantaRules({
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
    rule_type: ruleType,
  });
  const rules = query.data?.items ?? [];
  const tableRows: RuleTableRow[] = rules.map((rule) => ({
    id: rule.id,
    rule,
    onDelete: setDeleting,
  }));
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const columns = useMemo(() => ruleColumns(isAdmin), [isAdmin]);
  const table = useDataTable({
    tableState: tableSearch,
    data: tableRows,
    columns,
    pageCount,
    rowCount: totalCount,
    initialState: { pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE } },
    getRowId: (row) => String(row.id),
    enableRowSelection: isAdmin,
  });
  return (
    <PageShell>
      <PageHeader
        title="Rules"
        actions={
          isAdmin ? (
            <Button size="sm" render={<Link to="/santa/rules/new" />} nativeButton={false}>
              <Plus data-icon="inline-start" />
              Create
            </Button>
          ) : null
        }
      />

      {query.error ? (
        <QueryError
          title="Failed to load rules"
          error={query.error}
          onRetry={() => void query.refetch()}
        />
      ) : query.isLoading ? (
        <DataTableSkeleton columnCount={isAdmin ? 6 : 4} filterCount={1} />
      ) : (
        <DataTable
          table={table}
          actionBar={
            isAdmin ? (
              <BulkDeleteActionBar
                table={table}
                useBulkDelete={useBulkDeleteSantaRules}
                noun="rule"
                description="Deleted rules stop syncing."
              />
            ) : undefined
          }
          empty={
            <DataTableEmpty
              icon={<ListChecks />}
              filtered={tableSearch.isFiltered}
              title="No execution rules"
              description="Create a rule, then attach label targets."
              filteredDescription="No rules matched these filters."
            />
          }
        >
          <DataTableSearchInput
            value={tableSearch.q ?? ""}
            onValueChange={tableSearch.onQueryChange}
          />
          <DataTableFacetedFilter
            column={table.getColumn("rule_type")}
            title="Rule Type"
            options={RULE_TYPE_OPTIONS}
          />
        </DataTable>
      )}

      {isAdmin ? (
        <RuleDeleteDialog
          rule={deleting}
          open={deleting !== null}
          onOpenChange={(open) => {
            if (!open) setDeleting(null);
          }}
        />
      ) : null}
    </PageShell>
  );
}

function RuleRowActions({
  rule,
  onDelete,
}: {
  rule: SantaRule;
  onDelete: (rule: SantaRule) => void;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button type="button" size="icon" variant="ghost" />}>
        <MoreHorizontal />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuGroup>
          <DropdownMenuItem
            render={<Link to="/santa/rules/$id/edit" params={{ id: String(rule.id) }} />}
          >
            Edit
          </DropdownMenuItem>
          <DropdownMenuItem variant="destructive" onClick={() => onDelete(rule)}>
            Delete
          </DropdownMenuItem>
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
