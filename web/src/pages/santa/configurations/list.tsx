import { Link, useSearch } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { FileSliders, Loader2, Plus, Trash2 } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import { BulkDeleteDialog } from "@/components/data-table/bulk-delete-dialog";
import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableEmptyState } from "@/components/data-table/data-table-empty-state";
import { DataTableSearch } from "@/components/data-table/data-table-search";
import { DraggableDataTable, DraggableDataTableRowDragHandle } from "@/components/data-table/draggable-data-table";
import { labelsFromIDs, type LabelChip } from "@/components/labels/label-chip-utils";
import { LabelChips } from "@/components/labels/label-chips";
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
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ButtonGroup } from "@/components/ui/button-group";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import { useLabels } from "@/hooks/use-labels";
import {
  useBulkDeleteSantaConfigurations,
  useReorderSantaConfigurations,
  useSantaConfigurations,
  type SantaConfiguration,
} from "@/hooks/use-santa";
import { tableQueryParams, useTablePaginationParams } from "@/hooks/use-table-pagination-params";
import { MAX_PAGE_SIZE } from "@/lib/pagination";
import { formatRelative } from "@/lib/utils";

import { clientModeLabel } from "./shared";

export function SantaConfigurationsPage() {
  const search = useSearch({ strict: false });
  const { state, setters } = useTablePaginationParams();
  const [draft, setDraft] = useDebouncedSearchParam("q");
  const [selectedConfigurationIds, setSelectedConfigurationIds] = useState<string[]>([]);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [reorderWarningOpen, setReorderWarningOpen] = useState(false);
  const [reorderEnabled, setReorderEnabled] = useState(false);
  const query = useSantaConfigurations({
    q: typeof search.q === "string" ? search.q : undefined,
    ...(reorderEnabled ? { page_size: MAX_PAGE_SIZE, sort: "position.asc" } : tableQueryParams(state)),
  });
  const labels = useLabels({
    page_size: MAX_PAGE_SIZE,
    sort: "name.asc",
    label_type: "regular",
  });
  const bulkDelete = useBulkDeleteSantaConfigurations();
  const reorder = useReorderSantaConfigurations();
  const totalCount = query.data?.count ?? 0;
  const reorderTruncated = reorderEnabled && totalCount > MAX_PAGE_SIZE;
  const serverRows = useMemo(() => query.data?.items ?? [], [query.data?.items]);
  const labelsByID = useMemo<ReadonlyMap<number, LabelChip>>(
    () => new Map((labels.data?.items ?? []).map((label) => [label.id, label])),
    [labels.data?.items],
  );
  const [orderedRows, setOrderedRows] = useState<SantaConfiguration[]>([]);
  const hasFilters = !!search.q;
  const canEnableReorder = !hasFilters && orderedRows.length > 0 && !query.isLoading;
  const selectedIDs = selectedConfigurationIds.map(Number);

  useEffect(() => {
    setOrderedRows(serverRows);
  }, [serverRows]);

  function enableReorder() {
    setSelectedConfigurationIds([]);
    setReorderEnabled(true);
    setReorderWarningOpen(false);
  }

  function deleteSelectedConfigurations() {
    bulkDelete.mutate(selectedIDs, {
      onSuccess: () => {
        setSelectedConfigurationIds([]);
        setDeleteOpen(false);
      },
    });
  }

  function moveOrder(next: SantaConfiguration[]) {
    const nextRows = next.map((row, position) => ({ ...row, position }));
    setOrderedRows(nextRows);
  }

  function saveOrder() {
    reorder.mutate(
      orderedRows.map((row) => row.id),
      {
        onSuccess: () => {
          setReorderEnabled(false);
        },
        onError: () => setOrderedRows(serverRows),
      },
    );
  }

  const columns: ColumnDef<SantaConfiguration>[] = [
    ...(reorderEnabled
      ? ([
          {
            id: "drag",
            header: () => null,
            enableSorting: false,
            enableHiding: false,
            cell: () => <DraggableDataTableRowDragHandle label="Reorder configuration" />,
            meta: { headClassName: "w-10", cellClassName: "w-10" },
          },
        ] satisfies ColumnDef<SantaConfiguration>[])
      : []),
    {
      id: "position",
      accessorKey: "position",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Order" />,
      cell: ({ row }) => <span className="text-muted-foreground tabular-nums">{row.original.position + 1}</span>,
      meta: { headClassName: "w-20", cellClassName: "w-20" },
    },
    {
      id: "name",
      accessorKey: "name",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
      cell: ({ row }) => <span className="font-medium">{row.original.name}</span>,
    },
    {
      id: "client_mode",
      accessorKey: "client_mode",
      header: "Client mode",
      enableSorting: false,
      cell: ({ row }) => <Badge variant="secondary">{clientModeLabel(row.original.client_mode)}</Badge>,
    },
    {
      id: "labels",
      header: "Targets",
      enableSorting: false,
      cell: ({ row }) => <TargetLabelsCell labelIDs={row.original.label_ids ?? []} labelsByID={labelsByID} />,
    },
    {
      id: "updated_at",
      accessorKey: "updated_at",
      header: "Updated",
      enableSorting: false,
      cell: ({ row }) => (
        <span className="text-muted-foreground" title={new Date(row.original.updated_at).toLocaleString()}>
          {formatRelative(row.original.updated_at)}
        </span>
      ),
    },
  ];

  return (
    <PageShell>
      <PageHeader
        title="Santa configurations"
        description="Configurations are evaluated in list order; each label can belong to one configuration."
        actions={
          <>
            <ButtonGroup>
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={reorderEnabled || !canEnableReorder}
                onClick={() => setReorderWarningOpen(true)}
              >
                Edit order
              </Button>
              {reorderEnabled ? (
                <>
                  <Button
                    type="button"
                    variant="destructive"
                    size="sm"
                    disabled={reorder.isPending || query.isLoading || reorderTruncated}
                    onClick={saveOrder}
                  >
                    {reorder.isPending ? <Loader2 data-icon="inline-start" className="animate-spin" /> : null}
                    Save
                  </Button>
                  <Button type="button" variant="outline" size="sm" onClick={() => setReorderEnabled(false)}>
                    Cancel
                  </Button>
                </>
              ) : null}
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
        }
      />

      {query.error ? (
        <Alert variant="destructive">
          <AlertTitle>Failed to load configurations</AlertTitle>
          <AlertDescription>{query.error.message}</AlertDescription>
        </Alert>
      ) : reorderEnabled ? (
        <DraggableDataTable
          columns={columns}
          data={orderedRows}
          isLoading={query.isLoading}
          disabled={reorder.isPending || reorderTruncated || orderedRows.length <= 1}
          onRowReorder={moveOrder}
          empty={<ConfigurationsEmptyState hasFilters={hasFilters} />}
          footer={
            reorderTruncated ? (
              <div className="text-muted-foreground rounded-md border px-3 py-2 text-sm">
                Showing the first {MAX_PAGE_SIZE} of {totalCount} configurations. Narrow the list before editing order.
              </div>
            ) : null
          }
        />
      ) : (
        <DataTable
          columns={columns}
          data={serverRows}
          totalCount={totalCount}
          pagination={state.pagination}
          sorting={state.sorting}
          onPaginationChange={setters.setPagination}
          onSortingChange={setters.setSorting}
          isLoading={query.isLoading}
          enableRowSelection
          selectedRowIds={selectedConfigurationIds}
          onSelectedRowIdsChange={setSelectedConfigurationIds}
          bulkActions={
            <Button variant="destructive" size="sm" onClick={() => setDeleteOpen(true)} disabled={bulkDelete.isPending}>
              <Trash2 data-icon="inline-start" />
              Delete
            </Button>
          }
          rowHref={(row) => ({
            to: "/santa/configurations/$configurationId/edit",
            params: { configurationId: String(row.id) },
          })}
          toolbar={
            <div className="flex items-center gap-2">
              <DataTableSearch value={draft} onChange={setDraft} placeholder="Search" label="Search configurations" />
            </div>
          }
          empty={<ConfigurationsEmptyState hasFilters={hasFilters} />}
        />
      )}

      <BulkDeleteDialog
        open={deleteOpen}
        onOpenChange={(open) => {
          if (!open) bulkDelete.reset();
          setDeleteOpen(open);
        }}
        count={selectedIDs.length}
        noun="configuration"
        description="Deleted configurations stop applying to matching hosts."
        pending={bulkDelete.isPending}
        onConfirm={deleteSelectedConfigurations}
      />
      <ReorderWarningDialog open={reorderWarningOpen} onOpenChange={setReorderWarningOpen} onConfirm={enableReorder} />
    </PageShell>
  );
}

function ConfigurationsEmptyState({ hasFilters }: { hasFilters: boolean }) {
  return (
    <DataTableEmptyState
      icon={<FileSliders />}
      title={hasFilters ? "No matches" : "No client configurations"}
      description={
        hasFilters
          ? "No Santa configurations matched the current filters."
          : "Create a configuration to start sending Santa client settings."
      }
    />
  );
}

function TargetLabelsCell({
  labelIDs,
  labelsByID,
}: {
  labelIDs: number[];
  labelsByID: ReadonlyMap<number, LabelChip>;
}) {
  const countText = `${labelIDs.length} label${labelIDs.length === 1 ? "" : "s"}`;

  if (labelIDs.length === 0) {
    return <span className="text-muted-foreground text-sm tabular-nums">{countText}</span>;
  }

  const labels = labelsFromIDs(labelIDs, labelsByID);

  return (
    <HoverCard openDelay={150} closeDelay={150}>
      <HoverCardTrigger asChild>
        <button
          type="button"
          className="text-muted-foreground rounded-sm text-sm tabular-nums underline-offset-4 hover:underline focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
        >
          {countText}
        </button>
      </HoverCardTrigger>
      <HoverCardContent align="start" side="top" className="w-auto max-w-80 p-2">
        <LabelChips labels={labels} />
      </HoverCardContent>
    </HoverCard>
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
          <AlertDialogTitle>Reorder Santa configurations?</AlertDialogTitle>
          <AlertDialogDescription>
            Santa uses the first matching configuration for each host. Reordering can change client behavior
            immediately, so make sure you know what you&apos;re doing before continuing.
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
