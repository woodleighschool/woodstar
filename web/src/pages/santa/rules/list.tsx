import { getRouteApi, Link } from "@tanstack/react-router";
import type { CellContext, ColumnDef } from "@tanstack/react-table";
import { ListChecks, Plus } from "lucide-react";
import * as React from "react";

import { BulkDeleteActionBar } from "@/components/bulk-delete-action-bar";
import { DataTable } from "@/components/data-table/data-table";
import { DataTableEmpty } from "@/components/data-table/data-table-empty";
import { DataTableFacetedFilter } from "@/components/data-table/data-table-faceted-filter";
import { DataTableSearchInput } from "@/components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@/components/data-table/data-table-skeleton";
import { selectColumn } from "@/components/data-table/select-column";
import type { LabelChip } from "@/components/labels/label-chip-utils";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { QueryError } from "@/components/query-error";
import { TargetLabelsCell } from "@/components/targeting/target-labels-cell";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useAuth } from "@/hooks/use-auth";
import { useDataTable } from "@/hooks/use-data-table";
import { encodeSort, useDataTableSearch } from "@/hooks/use-data-table-search";
import { useLabels } from "@/hooks/use-labels";
import { useBulkDeleteSantaRules, useSantaRules } from "@/hooks/use-santa-rules";
import type { SantaRule } from "@/lib/api";
import { DEFAULT_PAGE_SIZE, MAX_PAGE_SIZE } from "@/lib/pagination";
import { RULE_TYPE_OPTIONS, ruleTypeLabel } from "@/lib/santa-rules";

const routeApi = getRouteApi("/_authenticated/santa/rules/");
const RULE_TYPE_FILTER_KEYS = [{ id: "rule_type" }] as const;

interface RuleTableRow {
  id: number;
  rule: SantaRule;
  isAdmin: boolean;
  labelsByID: ReadonlyMap<number, LabelChip>;
}

function RuleNameCell({ row }: CellContext<RuleTableRow, unknown>) {
  return row.original.isAdmin ? (
    <Link
      to="/santa/rules/$id"
      params={{ id: String(row.original.rule.id) }}
      className="font-medium hover:underline"
    >
      {row.original.rule.name}
    </Link>
  ) : (
    <span className="font-medium">{row.original.rule.name}</span>
  );
}

function RuleTargetsCell({ row }: CellContext<RuleTableRow, unknown>) {
  return row.original.rule.targets.include.length ? (
    <TargetLabelsCell targets={row.original.rule.targets} labelsByID={row.original.labelsByID} />
  ) : (
    <Badge variant="secondary">Inactive</Badge>
  );
}

const ruleColumns: ColumnDef<RuleTableRow>[] = [
  selectColumn<RuleTableRow>(),
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
    cell: ({ row }) => ruleTypeLabel(row.original.rule.rule_type),
    meta: { label: "Rule Type", options: RULE_TYPE_OPTIONS },
    enableColumnFilter: true,
  },
  {
    id: "identifier",
    accessorFn: (row) => row.rule.identifier,
    header: "Identifier",
    cell: ({ row }) => row.original.rule.identifier,
    meta: { label: "Identifier" },
  },
  {
    id: "targets",
    header: () => "Targets",
    enableSorting: false,
    cell: RuleTargetsCell,
    meta: { label: "Targets" },
  },
];

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
  const ruleType = search.rule_type;
  const query = useSantaRules({
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
    rule_type: ruleType,
  });
  const labels = useLabels({ per_page: MAX_PAGE_SIZE, sort: encodeSort("name") });
  const labelsByID = React.useMemo<ReadonlyMap<number, LabelChip>>(
    () => new Map((labels.data?.items ?? []).map((label) => [label.id, label])),
    [labels.data?.items],
  );
  const rules = query.data?.items ?? [];
  const tableRows: RuleTableRow[] = rules.map((rule) => ({
    id: rule.id,
    rule,
    isAdmin,
    labelsByID,
  }));
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const table = useDataTable({
    tableState: tableSearch,
    data: tableRows,
    columns: ruleColumns,
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
        <DataTableSkeleton columnCount={5} filterCount={1} />
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
          <div className="flex items-start justify-between gap-2 p-1">
            <div className="flex flex-1 flex-wrap items-center gap-2">
              <DataTableSearchInput
                className="h-8 w-40 lg:w-56"
                value={tableSearch.q ?? ""}
                onValueChange={tableSearch.onQueryChange}
              />
              <DataTableFacetedFilter
                column={table.getColumn("rule_type")}
                title="Rule Type"
                options={RULE_TYPE_OPTIONS}
              />
            </div>
          </div>
        </DataTable>
      )}
    </PageShell>
  );
}
