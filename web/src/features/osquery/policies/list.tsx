import { getRouteApi } from "@tanstack/react-router";
import {
  CircleAlert,
  CircleCheck,
  CircleDashed,
  Info,
  MoreHorizontal,
  Plus,
  ShieldCheck,
} from "lucide-react";
import { useMemo, useState } from "react";

import { BulkDeleteActionBar } from "@components/bulk-delete-action-bar";
import { DataTable } from "@components/data-table/data-table";
import { DataTableEmpty } from "@components/data-table/data-table-empty";
import { DataTableSearchInput } from "@components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@components/data-table/data-table-skeleton";
import { selectColumn } from "@components/data-table/select-column";
import type { DataTableCellContext, DataTableColumnDef } from "@components/data-table/types";
import { useDataTable } from "@components/data-table/use-data-table";
import { useDataTableSearch } from "@components/data-table/use-data-table-search";
import { EnumBadge } from "@components/enum-badge";
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
import { useAuth } from "@features/auth/queries";
import type { OsqueryPolicy } from "@lib/api";
import { DEFAULT_PAGE_SIZE } from "@lib/pagination";
import { formatRelative } from "@lib/utils";

import { PolicyDeleteDialog } from "./delete-dialog";
import { POLICY_REMEDIATION_MODES, policyRemediationMode } from "./model";
import { useBulkDeletePolicies, usePolicies } from "./queries";
import { PolicyResultCountLink } from "./result-count-link";

const routeApi = getRouteApi("/_authenticated/osquery/policies/");

type PolicyTableRow = OsqueryPolicy & {
  onDelete: (policy: OsqueryPolicy) => void;
};

export function PolicyListPage() {
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const tableSearch = useDataTableSearch({
    search,
    onSearchChange: (updater) => void navigate({ search: updater, replace: true }),
  });
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const bulkDelete = useBulkDeletePolicies();
  const [deleting, setDeleting] = useState<OsqueryPolicy | null>(null);
  const query = usePolicies({
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
  });
  const tableRows = useMemo<PolicyTableRow[]>(
    () => query.data?.items.map((policy) => ({ ...policy, onDelete: setDeleting })) ?? [],
    [query.data?.items],
  );
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const table = useDataTable({
    tableState: tableSearch,
    data: tableRows,
    columns: isAdmin ? policyAdminColumns : policyColumns,
    pageCount,
    rowCount: totalCount,
    initialState: { pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE } },
    getRowId: (row) => String(row.id),
    enableRowSelection: isAdmin,
  });
  return (
    <PageShell>
      <PageHeader
        title="Policies"
        actions={
          isAdmin ? (
            <Button size="sm" render={<Link to="/osquery/policies/new" />} nativeButton={false}>
              <Plus data-icon="inline-start" />
              Create
            </Button>
          ) : null
        }
      />
      {query.error ? (
        <QueryError
          title="Failed to Load Policies"
          error={query.error}
          onRetry={() => void query.refetch()}
        />
      ) : query.isLoading ? (
        <DataTableSkeleton columnCount={isAdmin ? 9 : 7} />
      ) : (
        <DataTable
          table={table}
          pending={query.isPlaceholderData}
          actionBar={
            isAdmin ? (
              <BulkDeleteActionBar table={table} bulkDelete={bulkDelete} noun="policy" />
            ) : undefined
          }
          empty={
            <DataTableEmpty
              icon={<ShieldCheck />}
              filtered={tableSearch.isFiltered}
              title="No Health Policies"
              description="Create a policy from SQL."
              filteredDescription="No policies matched the current search."
            />
          }
        >
          <DataTableSearchInput
            loading={query.isPlaceholderData}
            value={tableSearch.q ?? ""}
            onValueChange={tableSearch.onQueryChange}
          />
        </DataTable>
      )}

      {isAdmin ? (
        <PolicyDeleteDialog
          open={deleting !== null}
          onOpenChange={(open) => {
            if (!open) setDeleting(null);
          }}
          policy={deleting}
        />
      ) : null}
    </PageShell>
  );
}

const policyColumns: DataTableColumnDef<PolicyTableRow>[] = [
  {
    id: "name",
    accessorKey: "name",
    header: "Name",
    cell: ({ row }) => (
      <TextLink
        to="/osquery/policies/$id"
        params={{ id: String(row.original.id) }}
        className="font-medium"
      >
        {row.original.name}
      </TextLink>
    ),
    enableHiding: false,
    size: 360,
    minSize: 240,
    meta: { label: "Name" },
  },
  {
    id: "passing_host_count",
    accessorKey: "passing_host_count",
    header: () => (
      <span className="flex items-center gap-1.5">
        <CircleCheck className="size-4 text-status-online" />
        Pass
      </span>
    ),
    cell: ({ row }) => (
      <PolicyResultCountLink
        policyId={row.original.id}
        count={row.original.passing_host_count}
        status="pass"
      />
    ),
    size: 130,
    minSize: 130,
    maxSize: 130,
    enableResizing: false,
    meta: { label: "Pass" },
  },
  {
    id: "failing_host_count",
    accessorKey: "failing_host_count",
    header: () => (
      <span className="flex items-center gap-1.5">
        <CircleAlert className="size-4 text-destructive" />
        Fail
      </span>
    ),
    cell: ({ row }) => (
      <PolicyResultCountLink
        policyId={row.original.id}
        count={row.original.failing_host_count}
        status="fail"
      />
    ),
    size: 130,
    minSize: 130,
    maxSize: 130,
    enableResizing: false,
    meta: { label: "Fail" },
  },
  {
    id: "error_host_count",
    accessorKey: "error_host_count",
    header: () => (
      <span className="flex items-center gap-1.5">
        <Info className="size-4 text-warning" />
        Error
      </span>
    ),
    cell: ({ row }) => (
      <PolicyResultCountLink
        policyId={row.original.id}
        count={row.original.error_host_count}
        status="error"
      />
    ),
    size: 130,
    minSize: 130,
    maxSize: 130,
    enableResizing: false,
    meta: { label: "Error" },
  },
  {
    id: "pending_host_count",
    accessorKey: "pending_host_count",
    header: () => (
      <span className="flex items-center gap-1.5">
        <CircleDashed className="size-4 text-muted-foreground" />
        Pending
      </span>
    ),
    cell: ({ row }) => (
      <PolicyResultCountLink
        policyId={row.original.id}
        count={row.original.pending_host_count}
        status="pending"
      />
    ),
    size: 130,
    minSize: 130,
    maxSize: 130,
    enableResizing: false,
    meta: { label: "Pending" },
  },
  {
    id: "remediation",
    accessorFn: (row) => policyRemediationMode(row.remediation),
    header: "Remediation",
    cell: ({ row }) => (
      <EnumBadge
        value={policyRemediationMode(row.original.remediation)}
        metadata={POLICY_REMEDIATION_MODES}
      />
    ),
    size: 144,
    minSize: 144,
    maxSize: 144,
    enableResizing: false,
    meta: { label: "Remediation" },
  },
  {
    id: "updated_at",
    accessorKey: "updated_at",
    header: "Updated",
    cell: ({ row }) => formatRelative(row.original.updated_at),
    enableResizing: false,
    meta: { label: "Updated" },
  },
];

function PolicyActionsCell({ row }: DataTableCellContext<PolicyTableRow>) {
  const policy = row.original;
  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button type="button" size="icon" variant="ghost" />}>
        <MoreHorizontal />
        <span className="sr-only">Open policy actions</span>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuGroup>
          <DropdownMenuItem
            render={<Link to="/osquery/policies/$id/edit" params={{ id: String(policy.id) }} />}
          >
            Edit
          </DropdownMenuItem>
          <DropdownMenuItem variant="destructive" onClick={() => policy.onDelete(policy)}>
            Delete
          </DropdownMenuItem>
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

const policyAdminColumns: DataTableColumnDef<PolicyTableRow>[] = [
  selectColumn<PolicyTableRow>(),
  ...policyColumns,
  {
    id: "actions",
    header: () => null,
    enableSorting: false,
    enableHiding: false,
    size: 44,
    minSize: 44,
    maxSize: 44,
    enableResizing: false,
    cell: PolicyActionsCell,
  },
];
