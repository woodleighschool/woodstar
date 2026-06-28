import { Link } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { GripVertical, HardDrive, Plus } from "lucide-react";
import * as React from "react";
import { toast } from "sonner";

import { ConfirmDialog } from "@/components/confirm-dialog";
import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableEmpty } from "@/components/data-table/data-table-empty";
import { DataTableSearchInput } from "@/components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@/components/data-table/data-table-skeleton";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { QueryError } from "@/components/query-error";
import { Button } from "@/components/ui/button";
import { ButtonGroup } from "@/components/ui/button-group";
import {
  Sortable,
  SortableContent,
  SortableItem,
  SortableItemHandle,
} from "@/components/ui/sortable";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useAuth } from "@/hooks/use-auth";
import { useDataTable } from "@/hooks/use-data-table";
import { encodeSort, useDataTableSearch } from "@/hooks/use-data-table-search";
import {
  useMunkiDistributionPoints,
  useReorderMunkiDistributionPoints,
} from "@/hooks/use-munki-distribution-points";
import type { MunkiDistributionPoint } from "@/lib/api";
import { DEFAULT_PAGE_SIZE, MAX_PAGE_SIZE } from "@/lib/pagination";
import {
  BoolBadge,
  ConnectionBadge,
} from "@/pages/munki/distribution-points/distribution-point-badges";

export function DistributionPointListPage() {
  const tableSearch = useDataTableSearch();
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const [reorderEnabled, setReorderEnabled] = React.useState(false);
  const [reorderWarningOpen, setReorderWarningOpen] = React.useState(false);

  const query = useMunkiDistributionPoints(
    reorderEnabled
      ? { q: tableSearch.q, per_page: MAX_PAGE_SIZE, sort: encodeSort("position") }
      : {
          q: tableSearch.q,
          page: tableSearch.page,
          per_page: tableSearch.per_page,
          sort: tableSearch.sort,
        },
  );

  const serverRows = React.useMemo(() => query.data?.items ?? [], [query.data?.items]);
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const hasFilters = !!tableSearch.q;
  const reorderTruncated = reorderEnabled && totalCount > MAX_PAGE_SIZE;
  const canEnableReorder = isAdmin && !hasFilters && totalCount > 1 && !query.isLoading;

  const columns = React.useMemo<ColumnDef<MunkiDistributionPoint>[]>(
    () => distributionPointColumns(isAdmin),
    [isAdmin],
  );

  const table = useDataTable({
    tableState: tableSearch,
    data: serverRows,
    columns,
    pageCount,
    initialState: { pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE } },
    getRowId: (row) => String(row.id),
  });

  const emptyState = (
    <DataTableEmpty
      icon={<HardDrive />}
      filtered={hasFilters}
      title="No distribution points"
      description="Create a distribution point for Munki clients."
      filteredDescription="No distribution points matched the current filters."
    />
  );

  return (
    <PageShell>
      <PageHeader
        title="Distribution Points"
        actions={
          isAdmin ? (
            <>
              <ButtonGroup>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={reorderEnabled || !canEnableReorder}
                  onClick={() => setReorderWarningOpen(true)}
                >
                  Edit Order
                </Button>
              </ButtonGroup>
              {reorderEnabled ? null : (
                <Button asChild size="sm">
                  <Link to="/munki/distribution-points/new">
                    <Plus data-icon="inline-start" />
                    Create
                  </Link>
                </Button>
              )}
            </>
          ) : null
        }
      />

      {query.error ? (
        <QueryError
          title="Failed to load distribution points"
          error={query.error}
          onRetry={() => void query.refetch()}
        />
      ) : reorderEnabled && isAdmin ? (
        <DistributionPointReorder
          key={serverRows.map((row) => row.id).join(",")}
          rows={serverRows}
          truncated={reorderTruncated}
          totalCount={totalCount}
          onDone={() => setReorderEnabled(false)}
        />
      ) : query.isLoading ? (
        <DataTableSkeleton columnCount={5} />
      ) : (
        <DataTable table={table} empty={emptyState}>
          <div className="flex items-start justify-between gap-2 p-1">
            <div className="flex flex-1 flex-wrap items-center gap-2">
              <DataTableSearchInput className="h-8 w-40 lg:w-56" />
            </div>
          </div>
        </DataTable>
      )}

      {isAdmin ? (
        <ReorderWarningDialog
          open={reorderWarningOpen}
          onOpenChange={setReorderWarningOpen}
          onConfirm={() => {
            setReorderEnabled(true);
            setReorderWarningOpen(false);
          }}
        />
      ) : null}
    </PageShell>
  );
}

function distributionPointColumns(isAdmin: boolean): ColumnDef<MunkiDistributionPoint>[] {
  return [
    {
      id: "position",
      accessorKey: "position",
      header: ({ column }) => <DataTableColumnHeader column={column} label="Order" />,
      cell: ({ row }) => row.original.position + 1,
      meta: { label: "Order" },
      size: 80,
    },
    {
      id: "name",
      accessorKey: "name",
      header: ({ column }) => <DataTableColumnHeader column={column} label="Name" />,
      cell: ({ row }) =>
        isAdmin ? (
          <Link
            to="/munki/distribution-points/$distributionPointId"
            params={{ distributionPointId: String(row.original.id) }}
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
      id: "enabled",
      accessorKey: "enabled",
      header: () => "Enabled",
      enableSorting: false,
      cell: ({ row }) => <BoolBadge value={row.original.enabled} label="Enabled" />,
      meta: { label: "Enabled" },
    },
    {
      id: "online",
      accessorKey: "online",
      header: () => "Connection",
      enableSorting: false,
      cell: ({ row }) => <ConnectionBadge online={row.original.online} />,
      meta: { label: "Connection" },
    },
    {
      id: "client_base_url",
      accessorKey: "client_base_url",
      header: () => "Base URL",
      enableSorting: false,
      cell: ({ row }) =>
        row.original.client_base_url || <span className="text-muted-foreground">-</span>,
      meta: { label: "Base URL" },
    },
  ];
}

function DistributionPointReorder({
  rows,
  truncated,
  totalCount,
  onDone,
}: {
  rows: MunkiDistributionPoint[];
  truncated: boolean;
  totalCount: number;
  onDone: () => void;
}) {
  const [ordered, setOrdered] = React.useState<MunkiDistributionPoint[]>(rows);
  const reorder = useReorderMunkiDistributionPoints();

  const dragDisabled = reorder.isPending || truncated || ordered.length <= 1;

  function saveOrder() {
    reorder.mutate(
      ordered.map((row) => row.id),
      {
        onSuccess: () => {
          toast.success("Saved order");
          onDone();
        },
        onError: () => setOrdered(rows),
      },
    );
  }

  return (
    <div className="flex flex-col gap-2.5">
      <div className="flex items-center justify-end gap-2">
        <Button
          type="button"
          size="sm"
          disabled={reorder.isPending || truncated}
          onClick={saveOrder}
        >
          Save
        </Button>
        <Button type="button" variant="outline" size="sm" onClick={onDone}>
          Cancel
        </Button>
      </div>

      <Sortable value={ordered} onValueChange={setOrdered} getItemValue={(row) => row.id}>
        <div className="overflow-hidden rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-10" />
                <TableHead className="w-20">Order</TableHead>
                <TableHead>Name</TableHead>
                <TableHead>Enabled</TableHead>
                <TableHead>Connection</TableHead>
                <TableHead>Base URL</TableHead>
              </TableRow>
            </TableHeader>
            <SortableContent asChild>
              <TableBody>
                {ordered.map((row, index) => (
                  <SortableItem key={row.id} value={row.id} asChild>
                    <TableRow>
                      <TableCell className="w-10">
                        <SortableItemHandle asChild disabled={dragDisabled}>
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            aria-label="Drag to reorder"
                          >
                            <GripVertical className="text-muted-foreground" />
                          </Button>
                        </SortableItemHandle>
                      </TableCell>
                      <TableCell className="w-20">{index + 1}</TableCell>
                      <TableCell className="font-medium">{row.name}</TableCell>
                      <TableCell>
                        <BoolBadge value={row.enabled} label="Enabled" />
                      </TableCell>
                      <TableCell>
                        <ConnectionBadge online={row.online} />
                      </TableCell>
                      <TableCell>
                        {row.client_base_url || <span className="text-muted-foreground">-</span>}
                      </TableCell>
                    </TableRow>
                  </SortableItem>
                ))}
              </TableBody>
            </SortableContent>
          </Table>
        </div>
      </Sortable>

      {truncated ? (
        <div className="rounded-md border px-3 py-2 text-sm text-muted-foreground">
          Showing the first {MAX_PAGE_SIZE} of {totalCount} distribution points. Narrow the list
          before editing order.
        </div>
      ) : null}
    </div>
  );
}

function ReorderWarningDialog({
  open,
  onOpenChange,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
}) {
  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Reorder distribution points?"
      description="Woodstar redirects to the first matching distribution point. Reordering changes client behavior immediately."
      confirmLabel="Continue"
      onConfirm={onConfirm}
    />
  );
}
