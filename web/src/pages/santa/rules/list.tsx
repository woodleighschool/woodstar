import { Link, useNavigate, useSearch } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { ListChecks, Plus, Trash2 } from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";

import {
  BulkDeleteDialog,
  DataTable,
  DataTableColumnHeader,
  DataTableEmptyState,
  DataTableFacetedFilter,
  DataTableSearch,
} from "@/components/data-table";
import type { LabelChip } from "@/components/labels/label-chip-utils";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { QueryError } from "@/components/query-error";
import { TargetLabelsCell } from "@/components/targeting/target-labels-cell";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import { useLabels } from "@/hooks/use-labels";
import { useBulkDeleteSantaRules, useSantaRules, type SantaRule } from "@/hooks/use-santa-rules";
import { tableQueryParams, useTablePaginationParams } from "@/hooks/use-table-pagination-params";
import { MAX_PAGE_SIZE } from "@/lib/pagination";

import { RULE_TYPE_OPTIONS, ruleTypeLabel } from "@/lib/santa-rules";

export function RuleListPage() {
  const search = useSearch({ from: "/_authenticated/santa/rules/" });
  const navigate = useNavigate();
  const { state, setters } = useTablePaginationParams();
  const [draft, setDraft] = useDebouncedSearchParam("q");
  const ruleType = search.rule_type;
  const query = useSantaRules({
    q: typeof search.q === "string" ? search.q : undefined,
    rule_type: ruleType,
    ...tableQueryParams(state),
  });
  const labels = useLabels({
    page_size: MAX_PAGE_SIZE,
    sort: "name.asc",
  });
  const bulkDelete = useBulkDeleteSantaRules();
  const [selectedRuleIds, setSelectedRuleIds] = useState<string[]>([]);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const rows = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const hasFilters = !!search.q || !!ruleType;
  const selectedIDs = selectedRuleIds.map(Number);
  const labelsByID = useMemo<ReadonlyMap<number, LabelChip>>(
    () => new Map((labels.data?.items ?? []).map((label) => [label.id, label])),
    [labels.data?.items],
  );

  function deleteSelectedRules() {
    const count = selectedIDs.length;
    bulkDelete.mutate(selectedIDs, {
      onSuccess: () => {
        setSelectedRuleIds([]);
        setDeleteOpen(false);
        toast.success(`Deleted ${count} ${count === 1 ? "rule" : "rules"}`);
      },
    });
  }

  const columns: ColumnDef<SantaRule>[] = [
    {
      id: "name",
      accessorKey: "name",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
      cell: ({ row }) => row.original.name,
    },
    {
      id: "rule_type",
      accessorKey: "rule_type",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Rule Type" />,
      cell: ({ row }) => ruleTypeLabel(row.original.rule_type),
    },
    {
      id: "identifier",
      accessorKey: "identifier",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Identifier" />,
      cell: ({ row }) => row.original.identifier,
    },
    {
      id: "targets",
      header: "Targets",
      enableSorting: false,
      cell: ({ row }) => {
        const targets = row.original.targets;
        return targets.include.length ? (
          <TargetLabelsCell targets={targets} labelsByID={labelsByID} />
        ) : (
          <Badge variant="secondary">Inactive</Badge>
        );
      },
    },
  ];

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
          rowHref={(row) => ({ to: "/santa/rules/$ruleId", params: { ruleId: String(row.id) } })}
          toolbar={
            <div className="flex flex-wrap items-center gap-2">
              <DataTableSearch value={draft} onChange={setDraft} placeholder="Search" />
              <DataTableFacetedFilter
                title="Rule Type"
                selected={ruleType ? [ruleType] : []}
                options={RULE_TYPE_OPTIONS}
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
            <DataTableEmptyState
              icon={<ListChecks />}
              title={hasFilters ? "No Matches" : "No Execution Rules"}
              description={hasFilters ? "No rules matched these filters." : "Create a rule, then attach label targets."}
            />
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
        description="Deleted rules stop syncing."
        pending={bulkDelete.isPending}
        onConfirm={deleteSelectedRules}
      />
    </PageShell>
  );
}
