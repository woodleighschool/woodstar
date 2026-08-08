import { getRouteApi } from "@tanstack/react-router";
import { ListChecks, MoreHorizontal, Plus } from "lucide-react";
import { useMemo, useState } from "react";

import { BulkDeleteActionBar } from "@components/bulk-delete-action-bar";
import { DataTable } from "@components/data-table/data-table";
import { DataTableEmpty } from "@components/data-table/data-table-empty";
import { DataTableFacetedFilter } from "@components/data-table/data-table-faceted-filter";
import { DataTableSearchInput } from "@components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@components/data-table/data-table-skeleton";
import { selectColumn } from "@components/data-table/select-column";
import type { DataTableCellContext, DataTableColumnDef } from "@components/data-table/types";
import { useDataTable } from "@components/data-table/use-data-table";
import { useDataTableSearch } from "@components/data-table/use-data-table-search";
import { EnumBadge } from "@components/enum-badge";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { QueryError } from "@components/query-error";
import { QueryGate } from "@components/query-gate";
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
import { parseRouteID } from "@lib/route-params";
import { formatRelative } from "@lib/utils";

import { RuleDeleteDialog } from "./delete-dialog";
import { POLICIES, RULE_TYPES, RULE_TYPE_OPTIONS } from "./metadata";
import { useBulkDeleteSantaRules, useSantaRules } from "./queries";

const routeApi = getRouteApi("/_authenticated/santa/configurations/$id/rules/");
const RULE_FILTER_KEYS = [{ id: "rule_type", multiple: true }] as const;

interface RuleTableRow {
  id: number;
  rule: SantaRule;
  onDelete: (rule: SantaRule) => void;
}

function RuleNameCell({ row }: DataTableCellContext<RuleTableRow>) {
  return (
    <Link
      to="/santa/configurations/$id/rules/$ruleId"
      params={{
        id: String(row.original.rule.configuration_id),
        ruleId: String(row.original.rule.id),
      }}
      className="font-medium"
    >
      {row.original.rule.name}
    </Link>
  );
}

function RuleActionsCell({ row }: DataTableCellContext<RuleTableRow>) {
  return <RuleRowActions rule={row.original.rule} onDelete={row.original.onDelete} />;
}

function ruleColumns(isAdmin: boolean): DataTableColumnDef<RuleTableRow>[] {
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
      id: "policy",
      accessorFn: (row) => row.rule.policy,
      header: "Policy",
      cell: ({ row }) => <EnumBadge value={row.original.rule.policy} metadata={POLICIES} />,
      meta: { label: "Policy" },
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
          } satisfies DataTableColumnDef<RuleTableRow>,
        ]
      : []),
  ];
}

export function RuleListPage() {
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const { id } = routeApi.useParams();
  const configurationID = parseRouteID(id);
  const tableSearch = useDataTableSearch({
    search,
    onSearchChange: (updater) => void navigate({ search: updater, replace: true }),
    filterKeys: RULE_FILTER_KEYS,
  });
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const [deleting, setDeleting] = useState<SantaRule | null>(null);
  const ruleType = search.rule_type;
  const query = useSantaRules(
    {
      q: tableSearch.q,
      page: tableSearch.page,
      per_page: tableSearch.per_page,
      sort: tableSearch.sort,
      configuration_id: configurationID === null ? undefined : [configurationID],
      rule_type: ruleType,
    },
    configurationID !== null,
  );
  const tableRows = useMemo<RuleTableRow[]>(
    () =>
      query.data?.items.map((rule) => ({
        id: rule.id,
        rule,
        onDelete: setDeleting,
      })) ?? [],
    [query.data?.items],
  );
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
  if (configurationID === null) {
    return (
      <QueryGate
        title="Failed to load rules"
        error={{ message: "Configuration route is invalid." }}
      />
    );
  }
  return (
    <PageShell>
      <PageHeader
        title="Rules"
        actions={
          isAdmin ? (
            <Button
              size="sm"
              render={
                <Link
                  to="/santa/configurations/$id/rules/new"
                  params={{ id: String(configurationID) }}
                />
              }
              nativeButton={false}
            >
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
        <DataTableSkeleton columnCount={isAdmin ? 7 : 5} filterCount={1} />
      ) : (
        <DataTable
          table={table}
          pending={query.isPlaceholderData}
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
            loading={query.isPlaceholderData}
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
            render={
              <Link
                to="/santa/configurations/$id/rules/$ruleId/edit"
                params={{ id: String(rule.configuration_id), ruleId: String(rule.id) }}
              />
            }
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
