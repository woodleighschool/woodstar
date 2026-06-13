import { Link } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { FileSliders, GripVertical, Plus } from "lucide-react";
import * as React from "react";
import { toast } from "sonner";

import { BulkDeleteActionBar } from "@/components/bulk-delete-action-bar";
import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableSearchInput } from "@/components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@/components/data-table/data-table-skeleton";
import { EnumStatus } from "@/components/enum-status";
import type { LabelChip } from "@/components/labels/label-chip-utils";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { QueryError } from "@/components/query-error";
import { SubmitButton } from "@/components/submit-button";
import { TargetLabelsCell } from "@/components/targeting/target-labels-cell";
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
import { ButtonGroup } from "@/components/ui/button-group";
import { Checkbox } from "@/components/ui/checkbox";
import { DataTableEmpty } from "@/components/data-table/data-table-empty";
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
import {
  DEFAULT_PAGE_SIZE,
  encodeSort,
  MAX_PAGE_SIZE,
  useDataTableSearch,
} from "@/hooks/use-data-table-search";
import { useLabels } from "@/hooks/use-labels";
import {
  type SantaConfiguration,
  useBulkDeleteSantaConfigurations,
  useReorderSantaConfigurations,
  useSantaConfigurations,
} from "@/hooks/use-santa-configurations";
import { CLIENT_MODES } from "@/lib/santa-configurations";
import { formatRelative } from "@/lib/utils";

export function ConfigurationListPage() {
  const tableSearch = useDataTableSearch();
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const [reorderEnabled, setReorderEnabled] = React.useState(false);
  const [reorderWarningOpen, setReorderWarningOpen] = React.useState(false);

  const query = useSantaConfigurations(
    reorderEnabled
      ? { q: tableSearch.q, per_page: MAX_PAGE_SIZE, sort: encodeSort("position") }
      : {
          q: tableSearch.q,
          page: tableSearch.page,
          per_page: tableSearch.per_page,
          sort: tableSearch.sort,
        },
  );

  const labels = useLabels({ per_page: MAX_PAGE_SIZE, sort: encodeSort("name") });
  const labelsByID = React.useMemo<ReadonlyMap<number, LabelChip>>(
    () => new Map((labels.data?.items ?? []).map((label) => [label.id, label])),
    [labels.data?.items],
  );

  const serverRows = React.useMemo(() => query.data?.items ?? [], [query.data?.items]);
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const hasFilters = !!tableSearch.q;
  const reorderTruncated = reorderEnabled && totalCount > MAX_PAGE_SIZE;
  const canEnableReorder = isAdmin && !hasFilters && totalCount > 1 && !query.isLoading;

  const columns = React.useMemo<ColumnDef<SantaConfiguration>[]>(
    () => configurationColumns(labelsByID, isAdmin),
    [isAdmin, labelsByID],
  );

  const { table } = useDataTable({
    data: serverRows,
    columns,
    pageCount,
    initialState: { pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE } },
    getRowId: (row) => String(row.id),
  });

  const emptyState = (
    <DataTableEmpty
      icon={<FileSliders />}
      filtered={hasFilters}
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
                <Button asChild size="sm">
                  <Link to="/santa/configurations/new">
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
          title="Failed to load configurations"
          error={query.error}
          onRetry={() => void query.refetch()}
        />
      ) : reorderEnabled && isAdmin ? (
        <ConfigurationReorder
          key={serverRows.map((row) => row.id).join(",")}
          rows={serverRows}
          labelsByID={labelsByID}
          truncated={reorderTruncated}
          totalCount={totalCount}
          onDone={() => setReorderEnabled(false)}
        />
      ) : query.isLoading ? (
        <DataTableSkeleton columnCount={6} />
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

function configurationColumns(
  labelsByID: ReadonlyMap<number, LabelChip>,
  isAdmin: boolean,
): ColumnDef<SantaConfiguration>[] {
  const columns: ColumnDef<SantaConfiguration>[] = [
    {
      id: "select",
      header: ({ table }) => (
        <Checkbox
          checked={
            table.getIsAllPageRowsSelected() ||
            (table.getIsSomePageRowsSelected() && "indeterminate")
          }
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
            to="/santa/configurations/$configurationId"
            params={{ configurationId: String(row.original.id) }}
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
      id: "client_mode",
      accessorKey: "client_mode",
      header: () => "Client Mode",
      enableSorting: false,
      cell: ({ row }) => <EnumStatus value={row.original.client_mode} metadata={CLIENT_MODES} />,
      meta: { label: "Client Mode" },
    },
    {
      id: "labels",
      header: () => "Targets",
      enableSorting: false,
      cell: ({ row }) => (
        <TargetLabelsCell targets={row.original.targets} labelsByID={labelsByID} />
      ),
      meta: { label: "Targets" },
    },
    {
      id: "updated_at",
      accessorKey: "updated_at",
      header: () => "Updated",
      enableSorting: false,
      cell: ({ row }) => formatRelative(row.original.updated_at),
      meta: { label: "Updated" },
    },
  ];
  return isAdmin ? columns : columns.filter((column) => column.id !== "select");
}

function ConfigurationReorder({
  rows,
  labelsByID,
  truncated,
  totalCount,
  onDone,
}: {
  rows: SantaConfiguration[];
  labelsByID: ReadonlyMap<number, LabelChip>;
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
        <SubmitButton
          type="button"
          pending={reorder.isPending}
          size="sm"
          disabled={truncated}
          onClick={saveOrder}
        >
          Save
        </SubmitButton>
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
                <TableHead>Client Mode</TableHead>
                <TableHead>Targets</TableHead>
                <TableHead>Updated</TableHead>
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
                        <EnumStatus value={row.client_mode} metadata={CLIENT_MODES} />
                      </TableCell>
                      <TableCell>
                        <TargetLabelsCell targets={row.targets} labelsByID={labelsByID} />
                      </TableCell>
                      <TableCell>{formatRelative(row.updated_at)}</TableCell>
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
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Reorder configurations?</AlertDialogTitle>
          <AlertDialogDescription>
            Santa uses the first matching configuration for each host. Reordering can change client
            behavior immediately.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel variant="ghost" size="sm">
            Cancel
          </AlertDialogCancel>
          <AlertDialogAction
            size="sm"
            onClick={(event) => {
              event.preventDefault();
              onConfirm();
            }}
          >
            Continue
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
