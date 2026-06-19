import { Link } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { ListChecks, Plus } from "lucide-react";
import * as React from "react";

import { BulkDeleteActionBar } from "@/components/bulk-delete-action-bar";
import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
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
import {
  type SantaRule,
  type SantaRuleType,
  useBulkDeleteSantaRules,
  useSantaRules,
} from "@/hooks/use-santa-rules";
import { DEFAULT_PAGE_SIZE, MAX_PAGE_SIZE } from "@/lib/pagination";
import { RULE_TYPE_OPTIONS, ruleTypeLabel } from "@/lib/santa-rules";

const RULE_TYPE_FILTER_KEYS = [{ id: "rule_type" }] as const;

export function RuleListPage() {
  const tableSearch = useDataTableSearch(RULE_TYPE_FILTER_KEYS);
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";

  const ruleType = tableSearch.filters.rule_type?.[0] as SantaRuleType | undefined;

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
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const hasFilters = !!tableSearch.q || !!ruleType;

  const columns = React.useMemo<ColumnDef<SantaRule>[]>(() => {
    const baseColumns: ColumnDef<SantaRule>[] = [
      selectColumn<SantaRule>(),
      {
        id: "name",
        accessorKey: "name",
        header: ({ column }) => <DataTableColumnHeader column={column} label="Name" />,
        cell: ({ row }) =>
          isAdmin ? (
            <Link
              to="/santa/rules/$ruleId"
              params={{ ruleId: String(row.original.id) }}
              className="font-medium hover:underline"
            >
              {row.original.name}
            </Link>
          ) : (
            <span className="font-medium">{row.original.name}</span>
          ),
        enableHiding: false,
        meta: { label: "Name" },
      },
      {
        id: "rule_type",
        accessorKey: "rule_type",
        header: ({ column }) => <DataTableColumnHeader column={column} label="Rule Type" />,
        cell: ({ row }) => ruleTypeLabel(row.original.rule_type),
        meta: { label: "Rule Type", options: RULE_TYPE_OPTIONS },
        enableColumnFilter: true,
      },
      {
        id: "identifier",
        accessorKey: "identifier",
        header: ({ column }) => <DataTableColumnHeader column={column} label="Identifier" />,
        cell: ({ row }) => row.original.identifier,
        meta: { label: "Identifier" },
      },
      {
        id: "targets",
        header: () => "Targets",
        enableSorting: false,
        cell: ({ row }) =>
          row.original.targets.include.length ? (
            <TargetLabelsCell targets={row.original.targets} labelsByID={labelsByID} />
          ) : (
            <Badge variant="secondary">Inactive</Badge>
          ),
        meta: { label: "Targets" },
      },
    ];
    return isAdmin ? baseColumns : baseColumns.filter((column) => column.id !== "select");
  }, [isAdmin, labelsByID]);

  const table = useDataTable({
    tableState: tableSearch,
    data: rules,
    columns,
    pageCount,
    initialState: { pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE } },
    getRowId: (row) => String(row.id),
  });

  return (
    <PageShell>
      <PageHeader
        title="Rules"
        actions={
          isAdmin ? (
            <Button asChild size="sm">
              <Link to="/santa/rules/new">
                <Plus data-icon="inline-start" />
                Create
              </Link>
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
              filtered={hasFilters}
              title="No execution rules"
              description="Create a rule, then attach label targets."
              filteredDescription="No rules matched these filters."
            />
          }
        >
          <div className="flex items-start justify-between gap-2 p-1">
            <div className="flex flex-1 flex-wrap items-center gap-2">
              <DataTableSearchInput className="h-8 w-40 lg:w-56" />
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
