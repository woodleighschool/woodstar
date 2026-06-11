import { Link } from "@tanstack/react-router";
import type { ColumnDef, Table as TanStackTable } from "@tanstack/react-table";
import { ListChecks, Plus, Trash2 } from "lucide-react";
import * as React from "react";
import { toast } from "sonner";

import { BulkDeleteDialog } from "@/components/bulk-delete-dialog";
import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableFacetedFilter } from "@/components/data-table/data-table-faceted-filter";
import { DataTableSearchInput } from "@/components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@/components/data-table/data-table-skeleton";
import type { LabelChip } from "@/components/labels/label-chip-utils";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { QueryError } from "@/components/query-error";
import { TargetLabelsCell } from "@/components/targeting/target-labels-cell";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { useDataTable } from "@/hooks/use-data-table";
import { DEFAULT_PAGE_SIZE, encodeSort, MAX_PAGE_SIZE, useDataTableSearch } from "@/hooks/use-data-table-search";
import { useLabels } from "@/hooks/use-labels";
import { useBulkDeleteSantaRules, useSantaRules, type SantaRule, type SantaRuleType } from "@/hooks/use-santa-rules";
import { RULE_TYPE_OPTIONS, ruleTypeLabel } from "@/lib/santa-rules";

const RULE_TYPE_FILTER_KEYS = [{ id: "rule_type" }] as const;

export function RuleListPage() {
  const tableSearch = useDataTableSearch(RULE_TYPE_FILTER_KEYS);

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

  const columns = React.useMemo<ColumnDef<SantaRule>[]>(
    () => [
      {
        id: "select",
        header: ({ table }) => (
          <Checkbox
            checked={table.getIsAllPageRowsSelected() || (table.getIsSomePageRowsSelected() && "indeterminate")}
            onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
            aria-label="Select all"
          />
        ),
        cell: ({ row }) => (
          <Checkbox
            checked={row.getIsSelected()}
            onCheckedChange={(value) => row.toggleSelected(!!value)}
            aria-label="Select row"
          />
        ),
        enableSorting: false,
        enableHiding: false,
        size: 32,
      },
      {
        id: "name",
        accessorKey: "name",
        header: ({ column }) => <DataTableColumnHeader column={column} label="Name" />,
        cell: ({ row }) => (
          <Link
            to="/santa/rules/$ruleId"
            params={{ ruleId: String(row.original.id) }}
            className="font-medium hover:underline"
          >
            {row.original.name}
          </Link>
        ),
        enableHiding: false,
        meta: { label: "Name" },
      },
      {
        id: "rule_type",
        accessorKey: "rule_type",
        header: ({ column }) => <DataTableColumnHeader column={column} label="Rule Type" />,
        cell: ({ row }) => ruleTypeLabel(row.original.rule_type),
        meta: { label: "Rule Type", variant: "select", options: RULE_TYPE_OPTIONS },
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
    ],
    [labelsByID],
  );

  const { table } = useDataTable({
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
          <Button asChild size="sm">
            <Link to="/santa/rules/new">
              <Plus data-icon="inline-start" />
              Create
            </Link>
          </Button>
        }
      />

      {query.error ? (
        <QueryError title="Failed to load rules" error={query.error} onRetry={() => void query.refetch()} />
      ) : query.isLoading ? (
        <DataTableSkeleton columnCount={5} filterCount={1} />
      ) : (
        <DataTable
          table={table}
          actionBar={<RulesActionBar table={table} />}
          empty={
            <Empty className="min-h-72 border-0">
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <ListChecks />
                </EmptyMedia>
                <EmptyTitle>{hasFilters ? "No matches" : "No execution rules"}</EmptyTitle>
                <EmptyDescription>
                  {hasFilters ? "No rules matched these filters." : "Create a rule, then attach label targets."}
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
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

function RulesActionBar({ table }: { table: TanStackTable<SantaRule> }) {
  const rows = table.getFilteredSelectedRowModel().rows;
  const ids = React.useMemo(() => rows.map((row) => Number(row.original.id)), [rows]);
  const [open, setOpen] = React.useState(false);
  const bulkDelete = useBulkDeleteSantaRules();

  const onConfirm = () => {
    const count = ids.length;
    bulkDelete.mutate(ids, {
      onSuccess: () => {
        toast.success(`Deleted ${count} ${count === 1 ? "rule" : "rules"}`);
        table.toggleAllRowsSelected(false);
        setOpen(false);
      },
    });
  };

  return (
    <div className="flex items-center gap-3 rounded-md border bg-background p-1 pl-3 shadow-sm">
      <span className="text-sm text-muted-foreground">{ids.length} selected</span>
      <Button variant="destructive" size="sm" onClick={() => setOpen(true)} disabled={bulkDelete.isPending}>
        <Trash2 />
        Delete
      </Button>
      <BulkDeleteDialog
        open={open}
        onOpenChange={(next) => {
          if (!next) bulkDelete.reset();
          setOpen(next);
        }}
        count={ids.length}
        noun="rule"
        description="Deleted rules stop syncing."
        pending={bulkDelete.isPending}
        onConfirm={onConfirm}
      />
    </div>
  );
}
