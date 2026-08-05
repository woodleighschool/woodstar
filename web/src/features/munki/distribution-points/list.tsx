import { getRouteApi } from "@tanstack/react-router";
import { GripVertical, HardDrive, MoreHorizontal, Plus } from "lucide-react";
import * as React from "react";

import { AsyncButton } from "@components/async-button";
import { ConfirmDialog } from "@components/confirm-dialog";
import { DataTable } from "@components/data-table/data-table";
import { DataTableEmpty } from "@components/data-table/data-table-empty";
import { DataTableSearchInput } from "@components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@components/data-table/data-table-skeleton";
import {
  DraggableTableRow,
  DraggableTableRowHandle,
  DraggableTableRows,
} from "@components/data-table/draggable-table-rows";
import { TableSurface } from "@components/data-table/table-surface";
import type { DataTableColumnDef } from "@components/data-table/types";
import { useDataTable } from "@components/data-table/use-data-table";
import { encodeSort, useDataTableSearch } from "@components/data-table/use-data-table-search";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { QueryError } from "@components/query-error";
import { Button } from "@components/ui/button";
import { ButtonGroup } from "@components/ui/button-group";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@components/ui/dropdown-menu";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@components/ui/table";
import { toast } from "@components/ui/toast";
import { useAuth } from "@features/auth/queries";
import type { MunkiDistributionPoint } from "@lib/api";
import { DEFAULT_PAGE_SIZE, MAX_PAGE_SIZE } from "@lib/pagination";

import { DistributionPointDeleteDialog } from "./delete-dialog";
import { BoolBadge, WorkerStatus } from "./distribution-point-badges";
import { useMunkiDistributionPoints, useReorderMunkiDistributionPoints } from "./queries";

const routeApi = getRouteApi("/_authenticated/munki/distribution-points/");

export function DistributionPointListPage() {
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const tableSearch = useDataTableSearch({
    search,
    onSearchChange: (updater) => void navigate({ search: updater, replace: true }),
  });
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const [reorderEnabled, setReorderEnabled] = React.useState(false);
  const [reorderWarningOpen, setReorderWarningOpen] = React.useState(false);
  const [deleting, setDeleting] = React.useState<MunkiDistributionPoint | null>(null);
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
  const reorderTruncated = reorderEnabled && totalCount > MAX_PAGE_SIZE;
  const canEnableReorder = isAdmin && !tableSearch.isFiltered && totalCount > 1 && !query.isLoading;
  const columns = React.useMemo<DataTableColumnDef<MunkiDistributionPoint>[]>(
    () => distributionPointColumns(isAdmin, setDeleting),
    [isAdmin],
  );
  const table = useDataTable({
    tableState: tableSearch,
    data: serverRows,
    columns,
    pageCount,
    rowCount: totalCount,
    initialState: { pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE } },
    getRowId: (row) => String(row.id),
  });
  const emptyState = (
    <DataTableEmpty
      icon={<HardDrive />}
      filtered={tableSearch.isFiltered}
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
                <Button
                  size="sm"
                  render={<Link to="/munki/distribution-points/new" />}
                  nativeButton={false}
                >
                  <Plus data-icon="inline-start" />
                  Create
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
        <DataTableSkeleton columnCount={isAdmin ? 7 : 6} />
      ) : (
        <DataTable table={table} pending={query.isPlaceholderData} empty={emptyState}>
          <DataTableSearchInput
            loading={query.isPlaceholderData}
            value={tableSearch.q ?? ""}
            onValueChange={tableSearch.onQueryChange}
          />
        </DataTable>
      )}

      {isAdmin ? (
        <>
          <ReorderWarningDialog
            open={reorderWarningOpen}
            onOpenChange={setReorderWarningOpen}
            onConfirm={() => {
              setReorderEnabled(true);
              setReorderWarningOpen(false);
            }}
          />
          <DistributionPointDeleteDialog
            point={deleting}
            open={deleting !== null}
            onOpenChange={(open) => {
              if (!open) setDeleting(null);
            }}
          />
        </>
      ) : null}
    </PageShell>
  );
}
function distributionPointColumns(
  isAdmin: boolean,
  onDelete: (point: MunkiDistributionPoint) => void,
): DataTableColumnDef<MunkiDistributionPoint>[] {
  const columns: DataTableColumnDef<MunkiDistributionPoint>[] = [
    {
      id: "position",
      accessorKey: "position",
      header: "Order",
      cell: ({ row }) => row.original.position + 1,
      meta: { label: "Order" },
    },
    {
      id: "name",
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => (
        <Link
          to="/munki/distribution-points/$id"
          params={{ id: String(row.original.id) }}
          className="font-medium"
        >
          {row.original.name}
        </Link>
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
      id: "worker",
      header: () => "Connection",
      enableSorting: false,
      cell: ({ row }) => <WorkerStatus worker={row.original.worker} />,
      meta: { label: "Connection" },
    },
    {
      id: "worker_version",
      header: () => "Worker Version",
      enableSorting: false,
      cell: ({ row }) =>
        row.original.worker?.build_version ?? <span className="text-muted-foreground">-</span>,
      meta: { label: "Worker Version" },
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
  if (!isAdmin) return columns;
  return [
    ...columns,
    {
      id: "actions",
      header: () => null,
      enableSorting: false,
      enableHiding: false,
      size: 44,
      minSize: 44,
      maxSize: 44,
      enableResizing: false,
      cell: ({ row }) => (
        <DistributionPointRowActions point={row.original} onDelete={() => onDelete(row.original)} />
      ),
    },
  ];
}

function DistributionPointRowActions({
  point,
  onDelete,
}: {
  point: MunkiDistributionPoint;
  onDelete: () => void;
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
              <Link to="/munki/distribution-points/$id/edit" params={{ id: String(point.id) }} />
            }
          >
            Edit
          </DropdownMenuItem>
          <DropdownMenuItem variant="destructive" onClick={onDelete}>
            Delete
          </DropdownMenuItem>
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
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
          toast.add({ title: "Saved order", type: "success" });
          onDone();
        },
        onError: () => setOrdered(rows),
      },
    );
  }
  return (
    <div className="flex flex-col gap-2.5">
      <div className="flex items-center justify-end gap-2">
        <AsyncButton
          type="button"
          size="sm"
          isPending={reorder.isPending}
          disabled={truncated}
          onClick={saveOrder}
        >
          Save
        </AsyncButton>
        <Button type="button" variant="outline" size="sm" onClick={onDone}>
          Cancel
        </Button>
      </div>

      <DraggableTableRows value={ordered} onValueChange={setOrdered} getRowId={(row) => row.id}>
        <TableSurface>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-10" />
                <TableHead className="w-20">Order</TableHead>
                <TableHead>Name</TableHead>
                <TableHead>Enabled</TableHead>
                <TableHead>Connection</TableHead>
                <TableHead>Worker Version</TableHead>
                <TableHead>Base URL</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {ordered.map((row, index) => (
                <DraggableTableRow key={row.id} id={row.id}>
                  <TableCell className="w-10">
                    <DraggableTableRowHandle disabled={dragDisabled}>
                      <GripVertical className="text-muted-foreground" />
                    </DraggableTableRowHandle>
                  </TableCell>
                  <TableCell className="w-20">{index + 1}</TableCell>
                  <TableCell className="font-medium">{row.name}</TableCell>
                  <TableCell>
                    <BoolBadge value={row.enabled} label="Enabled" />
                  </TableCell>
                  <TableCell>
                    <WorkerStatus worker={row.worker} />
                  </TableCell>
                  <TableCell>
                    {row.worker?.build_version ?? <span className="text-muted-foreground">-</span>}
                  </TableCell>
                  <TableCell>
                    {row.client_base_url || <span className="text-muted-foreground">-</span>}
                  </TableCell>
                </DraggableTableRow>
              ))}
            </TableBody>
          </Table>
        </TableSurface>
      </DraggableTableRows>

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
