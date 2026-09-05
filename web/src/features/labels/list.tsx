import { getRouteApi } from "@tanstack/react-router";
import { MoreHorizontal, Plus, Tags } from "lucide-react";
import * as React from "react";

import { DataTable } from "@components/data-table/data-table";
import { DataTableEmpty } from "@components/data-table/data-table-empty";
import { DataTableFacetedFilter } from "@components/data-table/data-table-faceted-filter";
import { DataTableSearchInput } from "@components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@components/data-table/data-table-skeleton";
import type { DataTableCellContext, DataTableColumnDef } from "@components/data-table/types";
import { useDataTable } from "@components/data-table/use-data-table";
import { useDataTableSearch } from "@components/data-table/use-data-table-search";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link, TextLink } from "@components/link";
import { QueryError } from "@components/query-error";
import { Button } from "@components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@components/ui/dropdown-menu";
import { useCan } from "@features/authz/access";
import { LABEL_MEMBERSHIP_OPTIONS, labelMembershipLabel } from "@features/labels/model";
import { useLabels } from "@features/labels/queries";
import type { Label } from "@lib/api";
import { DEFAULT_PAGE_SIZE } from "@lib/pagination";
import { countLabel, formatRelative } from "@lib/utils";

import { LabelDeleteDialog } from "./delete-dialog";

const routeApi = getRouteApi("/_authenticated/labels/");
const MEMBERSHIP_FILTER_KEYS = [{ id: "label_membership_type", multiple: true }] as const;

interface LabelTableRow {
  label: Label;
  onDelete: (label: Label) => void;
}

function LabelNameCell({ row }: DataTableCellContext<LabelTableRow>) {
  return (
    <TextLink
      to="/labels/$id"
      params={{ id: String(row.original.label.id) }}
      className="font-medium"
    >
      {row.original.label.name}
    </TextLink>
  );
}

function LabelActionsCell({ row }: DataTableCellContext<LabelTableRow>) {
  return <LabelRowActions label={row.original.label} onDelete={row.original.onDelete} />;
}

function LabelHostCountCell({ row }: DataTableCellContext<LabelTableRow>) {
  const label = row.original.label;
  return (
    <TextLink
      to="/hosts"
      search={{ label_id: label.id }}
      className="whitespace-nowrap tabular-nums"
    >
      {countLabel(label.hosts_count, "host")}
    </TextLink>
  );
}

const labelColumns: DataTableColumnDef<LabelTableRow>[] = [
  {
    id: "name",
    accessorFn: (row) => row.label.name,
    header: "Name",
    cell: LabelNameCell,
    enableHiding: false,
    meta: { label: "Name" },
  },
  {
    id: "label_membership_type",
    accessorFn: (row) => row.label.label_membership_type,
    header: "Membership",
    cell: ({ row }) => labelMembershipLabel(row.original.label.label_membership_type),
    meta: { label: "Membership", options: LABEL_MEMBERSHIP_OPTIONS },
    enableColumnFilter: true,
  },
  {
    id: "hosts_count",
    accessorFn: (row) => row.label.hosts_count,
    header: "Hosts",
    cell: LabelHostCountCell,
    meta: { label: "Hosts" },
  },
  {
    id: "updated_at",
    accessorFn: (row) => row.label.updated_at,
    header: "Updated",
    cell: ({ row }) =>
      row.original.label.updated_at ? formatRelative(row.original.label.updated_at) : "-",
    meta: { label: "Updated" },
  },
  {
    id: "actions",
    header: () => null,
    enableSorting: false,
    enableHiding: false,
    size: 44,
    minSize: 44,
    maxSize: 44,
    enableResizing: false,
    cell: LabelActionsCell,
  },
];

const labelViewerColumns = labelColumns.filter((column) => column.id !== "actions");

export function LabelListPage() {
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const tableSearch = useDataTableSearch({
    search,
    onSearchChange: (updater) => void navigate({ search: updater, replace: true }),
    filterKeys: MEMBERSHIP_FILTER_KEYS,
  });
  const canEdit = useCan({ resource: "labels", access: "edit" });
  const [deleting, setDeleting] = React.useState<Label | null>(null);
  const membership = search.label_membership_type;
  const query = useLabels(
    {
      q: tableSearch.q,
      page: tableSearch.page,
      per_page: tableSearch.per_page,
      sort: tableSearch.sort,
      label_type: "regular",
      label_membership_type: membership,
    },
    { refetchInterval: 30000 },
  );
  const tableRows = React.useMemo<LabelTableRow[]>(
    () => query.data?.items.map((label) => ({ label, onDelete: setDeleting })) ?? [],
    [query.data?.items],
  );
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const table = useDataTable({
    tableState: tableSearch,
    data: tableRows,
    columns: canEdit ? labelColumns : labelViewerColumns,
    pageCount,
    rowCount: totalCount,
    initialState: { pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE } },
    getRowId: (row) => String(row.label.id),
  });
  return (
    <PageShell>
      <PageHeader
        title="Labels"
        description="Group hosts for targeting, reporting, and Santa rules."
        actions={
          canEdit ? (
            <Button size="sm" render={<Link to="/labels/new" />} nativeButton={false}>
              <Plus data-icon="inline-start" />
              Create
            </Button>
          ) : null
        }
      />
      {query.error ? (
        <QueryError
          title="Failed to Load Labels"
          error={query.error}
          onRetry={() => void query.refetch()}
        />
      ) : query.isLoading ? (
        <DataTableSkeleton columnCount={canEdit ? 5 : 4} filterCount={1} />
      ) : (
        <DataTable
          table={table}
          pending={query.isPlaceholderData}
          empty={
            <DataTableEmpty
              icon={<Tags />}
              filtered={tableSearch.isFiltered}
              title="No Labels"
              description="Create labels for host targeting."
              filteredDescription="No labels matched the current filters."
            />
          }
        >
          <DataTableSearchInput
            loading={query.isPlaceholderData}
            value={tableSearch.q ?? ""}
            onValueChange={tableSearch.onQueryChange}
          />
          <DataTableFacetedFilter
            column={table.getColumn("label_membership_type")}
            title="Membership"
            options={LABEL_MEMBERSHIP_OPTIONS}
          />
        </DataTable>
      )}

      {canEdit ? (
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
            render={<Link to="/labels/$id/edit" params={{ id: String(label.id) }} />}
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
