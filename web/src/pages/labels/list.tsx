import { Link } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { MoreHorizontal, Plus, Tags } from "lucide-react";
import * as React from "react";
import { toast } from "sonner";

import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableFacetedFilter } from "@/components/data-table/data-table-faceted-filter";
import { DataTableSearchInput } from "@/components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@/components/data-table/data-table-skeleton";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { QueryError } from "@/components/query-error";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { useDataTable } from "@/hooks/use-data-table";
import { DEFAULT_PAGE_SIZE, useDataTableSearch } from "@/hooks/use-data-table-search";
import { useDeleteLabel, useLabels, type Label, type LabelListParams } from "@/hooks/use-labels";
import { LABEL_MEMBERSHIP_OPTIONS, labelMembershipLabel } from "@/lib/labels";
import { formatRelative } from "@/lib/utils";

const MEMBERSHIP_FILTER_KEYS = [{ id: "label_membership_type" }] as const;

export function LabelListPage() {
  const tableSearch = useDataTableSearch(MEMBERSHIP_FILTER_KEYS);
  const [deleting, setDeleting] = React.useState<Label | null>(null);

  const membership = tableSearch.filters.label_membership_type?.[0];

  const query = useLabels({
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
    label_type: "regular",
    label_membership_type: membership as LabelListParams["label_membership_type"],
  });

  const labels = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const hasFilters = !!tableSearch.q || !!membership;

  const columns = React.useMemo<ColumnDef<Label>[]>(
    () => [
      {
        id: "name",
        accessorKey: "name",
        header: ({ column }) => <DataTableColumnHeader column={column} label="Name" />,
        cell: ({ row }) => (
          <Link
            to="/labels/$labelId/edit"
            params={{ labelId: String(row.original.id) }}
            className="font-medium hover:underline"
          >
            {row.original.name}
          </Link>
        ),
        enableHiding: false,
        meta: { label: "Name" },
      },
      {
        id: "label_membership_type",
        accessorKey: "label_membership_type",
        header: ({ column }) => <DataTableColumnHeader column={column} label="Membership" />,
        cell: ({ row }) => labelMembershipLabel(row.original.label_membership_type),
        meta: { label: "Membership", variant: "select", options: LABEL_MEMBERSHIP_OPTIONS },
        enableColumnFilter: true,
      },
      {
        id: "hosts_count",
        accessorKey: "hosts_count",
        header: ({ column }) => <DataTableColumnHeader column={column} label="Hosts" />,
        cell: ({ row }) => row.original.hosts_count,
        meta: { label: "Hosts" },
      },
      {
        id: "updated_at",
        accessorKey: "updated_at",
        header: ({ column }) => <DataTableColumnHeader column={column} label="Updated" />,
        cell: ({ row }) => (row.original.updated_at ? formatRelative(row.original.updated_at) : "-"),
        meta: { label: "Updated" },
      },
      {
        id: "actions",
        header: () => null,
        enableSorting: false,
        enableHiding: false,
        size: 48,
        cell: ({ row }) => <LabelRowActions label={row.original} onDelete={setDeleting} />,
      },
    ],
    [],
  );

  const { table } = useDataTable({
    data: labels,
    columns,
    pageCount,
    initialState: { pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE } },
    getRowId: (row) => String(row.id),
  });

  return (
    <PageShell>
      <PageHeader
        title="Labels"
        description="Group hosts for targeting, reporting, and Santa rules."
        actions={
          <Button asChild size="sm">
            <Link to="/labels/new">
              <Plus data-icon="inline-start" />
              Create
            </Link>
          </Button>
        }
      />
      {query.error ? (
        <QueryError title="Failed to load labels" error={query.error} onRetry={() => void query.refetch()} />
      ) : query.isLoading ? (
        <DataTableSkeleton columnCount={5} filterCount={1} />
      ) : (
        <DataTable
          table={table}
          empty={
            <Empty className="min-h-72 border-0">
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <Tags />
                </EmptyMedia>
                <EmptyTitle>{hasFilters ? "No matches" : "No labels"}</EmptyTitle>
                <EmptyDescription>
                  {hasFilters ? "No labels matched the current filters." : "Create labels for host targeting."}
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
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

      <LabelDeleteDialog
        label={deleting}
        open={deleting !== null}
        onOpenChange={(open) => {
          if (!open) setDeleting(null);
        }}
      />
    </PageShell>
  );
}

function LabelRowActions({ label, onDelete }: { label: Label; onDelete: (label: Label) => void }) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button type="button" size="icon" variant="ghost">
          <MoreHorizontal />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuGroup>
          <DropdownMenuItem asChild>
            <Link to="/labels/$labelId/edit" params={{ labelId: String(label.id) }}>
              Edit
            </Link>
          </DropdownMenuItem>
          <DropdownMenuItem variant="destructive" onSelect={() => onDelete(label)}>
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
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete Label</AlertDialogTitle>
          <AlertDialogDescription>
            {label ? `${label.name} will be removed from hosts and filters.` : "This label will be removed."}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel variant="ghost" size="sm" disabled={remove.isPending}>
            Cancel
          </AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            size="sm"
            disabled={remove.isPending}
            onClick={(event) => {
              event.preventDefault();
              void handleDelete();
            }}
          >
            Delete Label
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
