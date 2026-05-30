import { Link, useSearch } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { MoreHorizontal, Plus, Tags } from "lucide-react";
import { useState } from "react";

import {
  DataTable,
  DataTableColumnHeader,
  DataTableEmptyState,
  DataTableFacetedFilter,
  DataTableSearch,
} from "@/components/data-table";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
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
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import { useDeleteLabel, useLabels, type Label } from "@/hooks/use-labels";
import { tableQueryParams, useTablePaginationParams } from "@/hooks/use-table-pagination-params";
import { formatRelative } from "@/lib/utils";
import { LABEL_MEMBERSHIP_OPTIONS, labelMembershipLabel } from "@/pages/labels/shared";

export function LabelsPage() {
  const search = useSearch({ from: "/_authenticated/labels/" });
  const { state, setters } = useTablePaginationParams();
  const [draft, setDraft] = useDebouncedSearchParam("q");
  const [deleting, setDeleting] = useState<Label | null>(null);

  const query = useLabels({
    q: search.q,
    ...tableQueryParams(state),
    label_type: "regular",
    label_membership_type: search.label_membership_type,
  });

  const data = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const hasFilters = !!search.q || !!search.label_membership_type;

  const columns: ColumnDef<Label>[] = [
    {
      id: "name",
      accessorKey: "name",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
      cell: ({ row }) => row.original.name,
    },
    {
      id: "label_membership_type",
      accessorKey: "label_membership_type",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Membership" />,
      cell: ({ row }) => labelMembershipLabel(row.original.label_membership_type),
    },
    {
      id: "hosts_count",
      accessorKey: "hosts_count",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Hosts" />,
      cell: ({ row }) => row.original.hosts_count,
    },
    {
      id: "updated_at",
      accessorKey: "updated_at",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Updated" />,
      cell: ({ row }) => (row.original.updated_at ? formatRelative(row.original.updated_at) : "-"),
    },
    {
      id: "actions",
      header: () => null,
      enableSorting: false,
      cell: ({ row }) => <LabelRowActions label={row.original} onDelete={setDeleting} />,
      meta: { headClassName: "w-12" },
    },
  ];

  return (
    <PageShell>
      <PageHeader
        title="Labels"
        description="Group hosts for targeting, reporting, and Santa scope."
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
        <Alert variant="destructive">
          <AlertTitle>Failed to Load Labels</AlertTitle>
          <AlertDescription>{query.error.message}</AlertDescription>
          <Button variant="outline" size="sm" onClick={() => void query.refetch()} className="mt-2 w-fit">
            Retry
          </Button>
        </Alert>
      ) : (
        <DataTable
          columns={columns}
          data={data}
          totalCount={totalCount}
          pagination={state.pagination}
          sorting={state.sorting}
          onPaginationChange={setters.setPagination}
          onSortingChange={setters.setSorting}
          isLoading={query.isLoading}
          rowHref={(row) => ({ to: "/labels/$labelId/edit", params: { labelId: String(row.id) } })}
          toolbar={
            <div className="flex items-center gap-2">
              <DataTableSearch value={draft} onChange={setDraft} placeholder="Search" />
              <DataTableFacetedFilter
                title="Membership"
                options={LABEL_MEMBERSHIP_OPTIONS}
                selected={search.label_membership_type ? [search.label_membership_type] : []}
                onChange={(next) => setters.setFilter("label_membership_type", next[0])}
                singleSelect
              />
            </div>
          }
          empty={
            <DataTableEmptyState
              icon={<Tags />}
              title={hasFilters ? "No Matches" : "No Host Groups"}
              description={hasFilters ? "No labels matched the current filters." : "Create labels for host targeting."}
            />
          }
        />
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
