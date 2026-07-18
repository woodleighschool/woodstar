import { Link } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { MoreHorizontal, Plus, Tags } from "lucide-react";
import * as React from "react";
import { toast } from "sonner";

import { ConfirmDialog } from "@/components/confirm-dialog";
import { DataTable } from "@/components/data-table/data-table";
import { DataTableEmpty } from "@/components/data-table/data-table-empty";
import { DataTableFacetedFilter } from "@/components/data-table/data-table-faceted-filter";
import { DataTableSearchInput } from "@/components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@/components/data-table/data-table-skeleton";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { QueryError } from "@/components/query-error";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useAuth } from "@/hooks/use-auth";
import { useDataTable } from "@/hooks/use-data-table";
import { useDataTableSearch } from "@/hooks/use-data-table-search";
import { type LabelListParams, useDeleteLabel, useLabels } from "@/hooks/use-labels";
import type { Label } from "@/lib/api";
import { LABEL_MEMBERSHIP_OPTIONS, labelMembershipLabel } from "@/lib/labels";
import { DEFAULT_PAGE_SIZE } from "@/lib/pagination";
import { formatRelative } from "@/lib/utils";
const MEMBERSHIP_FILTER_KEYS = [{ id: "label_membership_type" }] as const;
export function LabelListPage() {
  const tableSearch = useDataTableSearch(MEMBERSHIP_FILTER_KEYS);
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const [deleting, setDeleting] = React.useState<Label | null>(null);
  const membership = tableSearch.filters.label_membership_type?.[0];
  const query = useLabels(
    {
      q: tableSearch.q,
      page: tableSearch.page,
      per_page: tableSearch.per_page,
      sort: tableSearch.sort,
      label_type: "regular",
      label_membership_type: membership as LabelListParams["label_membership_type"],
    },
    { refetchInterval: 30000 },
  );
  const labels = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const hasFilters = !!tableSearch.q || !!membership;
  const columns = React.useMemo<ColumnDef<Label>[]>(() => {
    const baseColumns: ColumnDef<Label>[] = [
      {
        id: "name",
        accessorKey: "name",
        header: "Name",
        cell: ({ row }) =>
          isAdmin ? (
            <Link
              to="/labels/$labelId/edit"
              params={{ labelId: String(row.original.id) }}
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
        id: "label_membership_type",
        accessorKey: "label_membership_type",
        header: "Membership",
        cell: ({ row }) => labelMembershipLabel(row.original.label_membership_type),
        meta: { label: "Membership", options: LABEL_MEMBERSHIP_OPTIONS },
        enableColumnFilter: true,
      },
      {
        id: "hosts_count",
        accessorKey: "hosts_count",
        header: "Hosts",
        cell: ({ row }) => row.original.hosts_count,
        meta: { label: "Hosts" },
      },
      {
        id: "updated_at",
        accessorKey: "updated_at",
        header: "Updated",
        cell: ({ row }) =>
          row.original.updated_at ? formatRelative(row.original.updated_at) : "-",
        meta: { label: "Updated" },
      },
      {
        id: "actions",
        header: () => null,
        enableSorting: false,
        enableHiding: false,
        size: 48,
        cell: ({ row }) =>
          isAdmin ? <LabelRowActions label={row.original} onDelete={setDeleting} /> : null,
      },
    ];
    return isAdmin ? baseColumns : baseColumns.filter((column) => column.id !== "actions");
  }, [isAdmin]);
  const table = useDataTable({
    tableState: tableSearch,
    data: labels,
    columns,
    pageCount,
    rowCount: totalCount,
    initialState: { pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE } },
    getRowId: (row) => String(row.id),
  });
  return (
    <PageShell>
      <PageHeader
        title="Labels"
        description="Group hosts for targeting, reporting, and Santa rules."
        actions={
          isAdmin ? (
            <Button size="sm" render={<Link to="/labels/new" />} nativeButton={false}>
              <Plus data-icon="inline-start" />
              Create
            </Button>
          ) : null
        }
      />
      {query.error ? (
        <QueryError
          title="Failed to load labels"
          error={query.error}
          onRetry={() => void query.refetch()}
        />
      ) : query.isLoading ? (
        <DataTableSkeleton columnCount={5} filterCount={1} />
      ) : (
        <DataTable
          table={table}
          empty={
            <DataTableEmpty
              icon={<Tags />}
              filtered={hasFilters}
              title="No labels"
              description="Create labels for host targeting."
              filteredDescription="No labels matched the current filters."
            />
          }
        >
          <div className="flex items-start justify-between gap-2 p-1">
            <div className="flex flex-1 flex-wrap items-center gap-2">
              <DataTableSearchInput className="h-8 w-40 lg:w-56" />
              <DataTableFacetedFilter
                column={table.getColumn("label_membership_type")}
                title="Membership"
                options={LABEL_MEMBERSHIP_OPTIONS}
              />
            </div>
          </div>
        </DataTable>
      )}

      {isAdmin ? (
        <LabelDeleteDialog
          label={deleting}
          open={deleting !== null}
          onOpenChange={(open) => {
            if (!open) setDeleting(null);
          }}
        />
      ) : null}
    </PageShell>
  );
}
function LabelRowActions({ label, onDelete }: { label: Label; onDelete: (label: Label) => void }) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button type="button" size="icon" variant="ghost" />}>
        <MoreHorizontal />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuGroup>
          <DropdownMenuItem
            render={<Link to="/labels/$labelId/edit" params={{ labelId: String(label.id) }} />}
          >
            Edit
          </DropdownMenuItem>
          <DropdownMenuItem variant="destructive" onClick={() => onDelete(label)}>
            Delete
          </DropdownMenuItem>
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
function LabelDeleteDialog({
  label,
  open,
  onOpenChange,
}: {
  label: Label | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const remove = useDeleteLabel();
  async function handleDelete() {
    if (!label) return;
    await remove.mutateAsync(label.id);
    onOpenChange(false);
    toast.success("Deleted label");
  }
  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Delete Label"
      description={
        label
          ? `${label.name} will be removed from hosts and filters.`
          : "This label will be removed."
      }
      confirmLabel="Delete Label"
      variant="destructive"
      pending={remove.isPending}
      onConfirm={() => void handleDelete()}
    />
  );
}
