import { Link, useSearch } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { MoreHorizontal, Plus, Search, Tags, X } from "lucide-react";
import { useState } from "react";

import { PageLead } from "@/components/queries/query-ui";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { DataTable } from "@/components/ui/data-table";
import { DataTableColumnHeader } from "@/components/ui/data-table-column-header";
import { DataTableFacetedFilter } from "@/components/ui/data-table-faceted-filter";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import { useDeleteLabel, useLabels, type Label } from "@/hooks/use-labels";
import { useTablePaginationParams } from "@/hooks/use-table-pagination-params";
import { PLATFORM_LABELS, QUERYABLE_PLATFORMS, platformLabel } from "@/lib/targeting";
import { formatRelative } from "@/lib/utils";

const MEMBERSHIP_OPTIONS = [
  { value: "dynamic", label: "Dynamic" },
  { value: "manual", label: "Manual" },
  { value: "host_vitals", label: "Host vitals" },
];

const PLATFORM_OPTIONS = QUERYABLE_PLATFORMS.map((platform) => ({ value: platform, label: PLATFORM_LABELS[platform] }));

export function LabelsPage() {
  const search = useSearch({ strict: false });
  const { state, setters } = useTablePaginationParams();
  const [draft, setDraft] = useDebouncedSearchParam("q");
  const [deleting, setDeleting] = useState<Label | null>(null);

  const query = useLabels({
    q: search.q,
    page: state.page,
    per_page: state.perPage,
    order_key: state.orderKey,
    order_direction: state.orderDirection,
    label_type: "regular",
    label_membership_type: search.label_membership_type,
    platform: search.platform,
  });

  const data = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const hasFilters = !!search.q || !!search.label_membership_type || !!search.platform;

  const columns: ColumnDef<Label>[] = [
    {
      id: "name",
      accessorKey: "name",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
      cell: ({ row }) => (
        <div className="grid gap-1">
          <span className="font-medium">{row.original.name}</span>
          {row.original.description ? (
            <span className="text-muted-foreground line-clamp-2 text-xs">{row.original.description}</span>
          ) : null}
        </div>
      ),
    },
    {
      id: "label_membership_type",
      accessorKey: "label_membership_type",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Membership" />,
      cell: ({ row }) => <span className="text-muted-foreground">{row.original.label_membership_type}</span>,
    },
    {
      id: "platform",
      accessorKey: "platform",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Platform" />,
      cell: ({ row }) => <span className="text-muted-foreground">{platformLabel(row.original.platform)}</span>,
    },
    {
      id: "hosts_count",
      accessorKey: "hosts_count",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Hosts" />,
      cell: ({ row }) => <span className="tabular-nums">{row.original.hosts_count}</span>,
    },
    {
      id: "updated_at",
      accessorKey: "updated_at",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Updated" />,
      cell: ({ row }) => (
        <span
          className="text-muted-foreground"
          title={row.original.updated_at ? new Date(row.original.updated_at).toLocaleString() : ""}
        >
          {row.original.updated_at ? formatRelative(row.original.updated_at) : "-"}
        </span>
      ),
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
    <div className="flex flex-col gap-5 p-6">
      <PageLead
        title="Labels"
        description="Group hosts for filtering, reports, checks, and future Santa/Munki targeting."
        actions={
          <Button asChild size="sm">
            <Link to="/labels/new">
              <Plus className="size-4" />
              Add label
            </Link>
          </Button>
        }
      />
      {query.error ? (
        <Alert variant="destructive">
          <AlertTitle>Failed to load labels</AlertTitle>
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
          page={state.page}
          perPage={state.perPage}
          sort={{ orderKey: state.orderKey, orderDirection: state.orderDirection }}
          onPageChange={setters.setPage}
          onPerPageChange={setters.setPerPage}
          onSortChange={(s) => setters.setSort(s.orderKey, s.orderDirection)}
          isLoading={query.isLoading}
          rowHref={(row) => ({ to: "/labels/$labelId/edit", params: { labelId: String(row.id) } })}
          toolbar={
            <div className="flex items-center gap-2">
              <div className="relative max-w-md flex-1">
                <Search
                  className="text-muted-foreground pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2"
                  aria-hidden
                />
                <Input
                  value={draft}
                  onChange={(e) => setDraft(e.target.value)}
                  placeholder="Search labels"
                  className="pr-8 pl-8"
                  aria-label="Search labels"
                />
                {draft ? (
                  <button
                    type="button"
                    onClick={() => setDraft("")}
                    className="text-muted-foreground hover:text-foreground absolute top-1/2 right-2 -translate-y-1/2 rounded p-0.5"
                    aria-label="Clear search"
                  >
                    <X className="size-3.5" />
                  </button>
                ) : null}
              </div>
              <DataTableFacetedFilter
                title="Membership"
                options={MEMBERSHIP_OPTIONS}
                selected={search.label_membership_type ? [search.label_membership_type] : []}
                onChange={(next) => setters.setFilter("label_membership_type", next[0])}
                singleSelect
              />
              <DataTableFacetedFilter
                title="Platform"
                options={PLATFORM_OPTIONS}
                selected={search.platform ? [search.platform] : []}
                onChange={(next) => setters.setFilter("platform", next[0])}
                singleSelect
              />
              <div className="text-muted-foreground ml-auto text-xs tabular-nums">
                {query.isFetching ? "Loading..." : `${totalCount} ${totalCount === 1 ? "label" : "labels"}`}
              </div>
            </div>
          }
          empty={
            <Empty>
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <Tags />
                </EmptyMedia>
                <EmptyTitle>{hasFilters ? "No matches" : "No labels yet"}</EmptyTitle>
                <EmptyDescription>
                  {hasFilters
                    ? "No labels matched the current filters."
                    : "Add a label to target hosts in queries, reports, and checks."}
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
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
    </div>
  );
}

function LabelRowActions({ label, onDelete }: { label: Label; onDelete: (label: Label) => void }) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button type="button" size="icon" variant="ghost" aria-label={`Actions for ${label.name}`}>
          <MoreHorizontal className="size-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem asChild>
          <Link to="/labels/$labelId/edit" params={{ labelId: String(label.id) }}>
            Edit
          </Link>
        </DropdownMenuItem>
        <DropdownMenuItem className="text-destructive focus:text-destructive" onSelect={() => onDelete(label)}>
          Delete
        </DropdownMenuItem>
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
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Delete label</DialogTitle>
          <DialogDescription>
            {label ? `${label.name} will be removed from hosts and filters.` : "This label will be removed."}
          </DialogDescription>
        </DialogHeader>
        {remove.error ? <p className="text-destructive text-sm">{remove.error.message}</p> : null}
        <DialogFooter>
          <DialogClose asChild>
            <Button type="button" variant="ghost" size="sm" disabled={remove.isPending}>
              Cancel
            </Button>
          </DialogClose>
          <Button
            type="button"
            variant="destructive"
            size="sm"
            disabled={remove.isPending}
            onClick={() => void handleDelete()}
          >
            Delete label
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
