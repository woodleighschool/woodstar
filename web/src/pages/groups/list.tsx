import { Link } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { UsersRound } from "lucide-react";
import * as React from "react";

import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableSearchInput } from "@/components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@/components/data-table/data-table-skeleton";
import { EnumBadge } from "@/components/enum-badge";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { QueryError } from "@/components/query-error";
import { DataTableEmpty } from "@/components/data-table/data-table-empty";
import { useDataTable } from "@/hooks/use-data-table";
import { useDataTableSearch } from "@/hooks/use-data-table-search";
import { DEFAULT_PAGE_SIZE } from "@/lib/pagination";
import { type Group, useGroups } from "@/hooks/use-groups";
import { DIRECTORY_SOURCES } from "@/lib/directory";
import { nonEmpty } from "@/lib/utils";

export function GroupListPage() {
  const tableSearch = useDataTableSearch();

  const query = useGroups({
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
  });

  const groups = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const hasFilters = !!tableSearch.q;

  const columns = React.useMemo<ColumnDef<Group>[]>(
    () => [
      {
        id: "display_name",
        accessorKey: "display_name",
        header: ({ column }) => <DataTableColumnHeader column={column} label="Name" />,
        cell: ({ row }) => (
          <Link
            to="/directory/users"
            search={{ group_id: row.original.id }}
            className="font-medium hover:underline"
          >
            {row.original.display_name}
          </Link>
        ),
        enableHiding: false,
        meta: { label: "Name" },
      },
      {
        id: "mail_nickname",
        accessorKey: "mail_nickname",
        header: ({ column }) => <DataTableColumnHeader column={column} label="Nickname" />,
        cell: ({ row }) => nonEmpty(row.original.mail_nickname) ?? "-",
        meta: { label: "Nickname" },
      },
      {
        id: "member_count",
        accessorKey: "member_count",
        header: ({ column }) => <DataTableColumnHeader column={column} label="Members" />,
        cell: ({ row }) => row.original.member_count,
        meta: { label: "Members" },
      },
      {
        id: "source",
        accessorKey: "source",
        header: ({ column }) => <DataTableColumnHeader column={column} label="Source" />,
        cell: ({ row }) => <EnumBadge value={row.original.source} metadata={DIRECTORY_SOURCES} />,
        meta: { label: "Source" },
      },
    ],
    [],
  );

  const { table } = useDataTable({
    data: groups,
    columns,
    pageCount,
    initialState: { pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE } },
    getRowId: (row) => String(row.id),
  });

  return (
    <PageShell>
      <PageHeader title="Groups" description="Browse directory groups." />

      {query.error ? (
        <QueryError
          title="Failed to load groups"
          error={query.error}
          onRetry={() => void query.refetch()}
        />
      ) : query.isLoading ? (
        <DataTableSkeleton columnCount={4} />
      ) : (
        <DataTable
          table={table}
          empty={
            <DataTableEmpty
              icon={<UsersRound />}
              filtered={hasFilters}
              title="No groups"
              description="Groups appear after directory sync."
              filteredDescription="No groups matched the current search."
            />
          }
        >
          <div className="flex items-start justify-between gap-2 p-1">
            <div className="flex flex-1 flex-wrap items-center gap-2">
              <DataTableSearchInput className="h-8 w-40 lg:w-56" />
            </div>
          </div>
        </DataTable>
      )}
    </PageShell>
  );
}
