import { getRouteApi } from "@tanstack/react-router";
import { UsersRound } from "lucide-react";
import { useMemo } from "react";

import { DataTable } from "@components/data-table/data-table";
import { DataTableEmpty } from "@components/data-table/data-table-empty";
import { DataTableSearchInput } from "@components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@components/data-table/data-table-skeleton";
import type { DataTableColumnDef } from "@components/data-table/types";
import { useDataTable } from "@components/data-table/use-data-table";
import { useDataTableSearch } from "@components/data-table/use-data-table-search";
import { EnumBadge } from "@components/enum-badge";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { TextLink } from "@components/link";
import { QueryError } from "@components/query-error";
import { useGroups } from "@features/directory/groups/queries";
import { DIRECTORY_SOURCES } from "@features/directory/source";
import type { Group } from "@lib/api";
import { DEFAULT_PAGE_SIZE } from "@lib/pagination";
import { countLabel, nonEmpty } from "@lib/utils";

const routeApi = getRouteApi("/_authenticated/directory/groups/");

const groupColumns: DataTableColumnDef<Group>[] = [
  {
    id: "display_name",
    accessorKey: "display_name",
    header: "Name",
    cell: ({ row }) => (
      <TextLink
        to="/directory/users"
        search={{ group_id: row.original.id }}
        className="font-medium"
      >
        {row.original.display_name}
      </TextLink>
    ),
    enableHiding: false,
    meta: { label: "Name" },
  },
  {
    id: "mail_nickname",
    accessorKey: "mail_nickname",
    header: "Nickname",
    cell: ({ row }) => nonEmpty(row.original.mail_nickname) ?? "-",
    meta: { label: "Nickname" },
  },
  {
    id: "member_count",
    accessorKey: "member_count",
    header: "Members",
    cell: ({ row }) => countLabel(row.original.member_count, "member"),
    meta: { label: "Members" },
  },
  {
    id: "source",
    accessorKey: "source",
    header: "Source",
    cell: ({ row }) => <EnumBadge value={row.original.source} metadata={DIRECTORY_SOURCES} />,
    meta: { label: "Source" },
  },
];

export function GroupListPage() {
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const tableSearch = useDataTableSearch({
    search,
    onSearchChange: (updater) => void navigate({ search: updater, replace: true }),
  });

  const query = useGroups({
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
  });

  const groups = useMemo(() => query.data?.items ?? [], [query.data?.items]);
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const table = useDataTable({
    tableState: tableSearch,
    data: groups,
    columns: groupColumns,
    pageCount,
    rowCount: totalCount,
    initialState: { pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE } },
    getRowId: (row) => String(row.id),
  });

  return (
    <PageShell>
      <PageHeader title="Groups" description="Browse directory groups." />

      {query.error ? (
        <QueryError
          title="Failed to Load Groups"
          error={query.error}
          onRetry={() => void query.refetch()}
        />
      ) : query.isLoading ? (
        <DataTableSkeleton columnCount={4} />
      ) : (
        <DataTable
          table={table}
          pending={query.isPlaceholderData}
          empty={
            <DataTableEmpty
              icon={<UsersRound />}
              filtered={tableSearch.isFiltered}
              title="No Groups"
              description="Groups appear after directory sync."
              filteredDescription="No groups matched the current search."
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
    </PageShell>
  );
}
