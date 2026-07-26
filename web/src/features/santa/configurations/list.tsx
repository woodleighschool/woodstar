import { getRouteApi } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { FileSliders, GripVertical, MoreHorizontal, Plus } from "lucide-react";
import * as React from "react";
import { toast } from "sonner";

import { AsyncButton } from "@components/async-button";
import { BulkDeleteActionBar } from "@components/bulk-delete-action-bar";
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
import { selectColumn } from "@components/data-table/select-column";
import { TableSurface } from "@components/data-table/table-surface";
import { useDataTable } from "@components/data-table/use-data-table";
import { encodeSort, useDataTableSearch } from "@components/data-table/use-data-table-search";
import { EnumStatusIndicator } from "@components/enum-status-indicator";
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
import { useAuth } from "@features/auth/queries";
import type { SantaConfiguration } from "@lib/api";
import { DEFAULT_PAGE_SIZE, MAX_PAGE_SIZE } from "@lib/pagination";
import { formatRelative } from "@lib/utils";

import { ConfigurationDeleteDialog } from "./delete-dialog";
import { CLIENT_MODES } from "./metadata";
import {
  useBulkDeleteSantaConfigurations,
  useReorderSantaConfigurations,
  useSantaConfigurations,
} from "./queries";

const routeApi = getRouteApi("/_authenticated/santa/configurations/");

export function ConfigurationListPage() {
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
  const [deleting, setDeleting] = React.useState<SantaConfiguration | null>(null);
  const query = useSantaConfigurations(
    reorderEnabled
      ? {
          q: tableSearch.q,
          per_page: MAX_PAGE_SIZE,
          sort: encodeSort("position"),
        }
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
  const columns = React.useMemo<ColumnDef<SantaConfiguration>[]>(
    () => configurationColumns(isAdmin, setDeleting),
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
    enableRowSelection: isAdmin,
  });
  const emptyState = (
    <DataTableEmpty
      icon={<FileSliders />}
      filtered={tableSearch.isFiltered}
      title="No client configurations"
      description="Create a configuration for Santa clients."
      filteredDescription="No configurations matched the current filters."
    />
  );
  return (
    <PageShell>
      <PageHeader
        title="Configurations"
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
                  render={<Link to="/santa/configurations/new" />}
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
          title="Failed to load configurations"
          error={query.error}
          onRetry={() => void query.refetch()}
        />
      ) : reorderEnabled && isAdmin ? (
        <ConfigurationReorder
          key={serverRows.map((row) => row.id).join(",")}
          rows={serverRows}
          truncated={reorderTruncated}
          totalCount={totalCount}
          onDone={() => setReorderEnabled(false)}
        />
      ) : query.isLoading ? (
        <DataTableSkeleton columnCount={isAdmin ? 6 : 4} />
      ) : (
        <DataTable
          table={table}
          actionBar={
            isAdmin ? (
              <BulkDeleteActionBar
                table={table}
                useBulkDelete={useBulkDeleteSantaConfigurations}
                noun="configuration"
                description="Deleted configurations stop applying."
              />
            ) : undefined
          }
          empty={emptyState}
        >
          <div className="flex items-start justify-between gap-2 p-1">
            <div className="flex flex-1 flex-wrap items-center gap-2">
              <DataTableSearchInput
                className="h-8 w-40 lg:w-56"
                value={tableSearch.q ?? ""}
                onValueChange={tableSearch.onQueryChange}
              />
            </div>
          </div>
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
          <ConfigurationDeleteDialog
            configuration={deleting}
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
function configurationColumns(
  isAdmin: boolean,
  onDelete: (configuration: SantaConfiguration) => void,
): ColumnDef<SantaConfiguration>[] {
  const columns: ColumnDef<SantaConfiguration>[] = [
    ...(isAdmin ? [selectColumn<SantaConfiguration>()] : []),
    {
      id: "position",
      accessorKey: "position",
      header: "Order",
      cell: ({ row }) => row.original.position + 1,
      meta: { label: "Order" },
      size: 80,
    },
    {
      id: "name",
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => (
        <Link
          to="/santa/configurations/$id"
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
      id: "client_mode",
      accessorKey: "client_mode",
      header: () => "Client Mode",
      enableSorting: false,
      cell: ({ row }) => (
        <EnumStatusIndicator value={row.original.client_mode} metadata={CLIENT_MODES} />
      ),
      meta: { label: "Client Mode" },
    },
    {
      id: "updated_at",
      accessorKey: "updated_at",
      header: () => "Updated",
      enableSorting: false,
      cell: ({ row }) => formatRelative(row.original.updated_at),
      meta: { label: "Updated" },
    },
    ...(isAdmin
      ? [
          {
            id: "actions",
            header: () => null,
            enableSorting: false,
            enableHiding: false,
            size: 48,
            cell: ({ row }) => (
              <ConfigurationRowActions configuration={row.original} onDelete={onDelete} />
            ),
          } satisfies ColumnDef<SantaConfiguration>,
        ]
      : []),
  ];
  return columns;
}

function ConfigurationRowActions({
  configuration,
  onDelete,
}: {
  configuration: SantaConfiguration;
  onDelete: (configuration: SantaConfiguration) => void;
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
              <Link to="/santa/configurations/$id/edit" params={{ id: String(configuration.id) }} />
            }
          >
            Edit
          </DropdownMenuItem>
          <DropdownMenuItem variant="destructive" onClick={() => onDelete(configuration)}>
            Delete
          </DropdownMenuItem>
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
function ConfigurationReorder({
  rows,
  truncated,
  totalCount,
  onDone,
}: {
  rows: SantaConfiguration[];
  truncated: boolean;
  totalCount: number;
  onDone: () => void;
}) {
  const [ordered, setOrdered] = React.useState<SantaConfiguration[]>(rows);
  const reorder = useReorderSantaConfigurations();
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
                <TableHead>Client Mode</TableHead>
                <TableHead>Updated</TableHead>
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
                    <EnumStatusIndicator value={row.client_mode} metadata={CLIENT_MODES} />
                  </TableCell>
                  <TableCell>{formatRelative(row.updated_at)}</TableCell>
                </DraggableTableRow>
              ))}
            </TableBody>
          </Table>
        </TableSurface>
      </DraggableTableRows>

      {truncated ? (
        <div className="rounded-md border px-3 py-2 text-sm text-muted-foreground">
          Showing the first {MAX_PAGE_SIZE} of {totalCount} configurations. Narrow the list before
          editing order.
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
      title="Reorder configurations?"
      description="Santa uses the first matching configuration for each host. Reordering can change client behavior immediately."
      confirmLabel="Continue"
      onConfirm={onConfirm}
    />
  );
}
