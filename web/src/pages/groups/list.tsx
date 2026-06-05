import { useSearch } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { UsersRound } from "lucide-react";

import { DataTable, DataTableColumnHeader, DataTableEmptyState, DataTableSearch } from "@/components/data-table";
import { EnumBadge } from "@/components/enum-badge";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import { useGroups, type Group } from "@/hooks/use-groups";
import { tableQueryParams, useTablePaginationParams } from "@/hooks/use-table-pagination-params";
import { nonEmpty } from "@/lib/utils";
import { DIRECTORY_SOURCES } from "@/pages/directory/shared";

export function GroupsPage() {
  const search = useSearch({ from: "/_authenticated/directory/groups/" });
  const { state, setters } = useTablePaginationParams();
  const [draft, setDraft] = useDebouncedSearchParam("q");

  const query = useGroups({
    q: search.q,
    ...tableQueryParams(state),
  });
  const data = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const hasFilters = !!search.q;

  const columns: ColumnDef<Group>[] = [
    {
      id: "display_name",
      accessorKey: "display_name",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
      cell: ({ row }) => row.original.display_name,
    },
    {
      id: "mail_nickname",
      accessorKey: "mail_nickname",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Nickname" />,
      cell: ({ row }) => nonEmpty(row.original.mail_nickname) ?? "-",
    },
    {
      id: "member_count",
      accessorKey: "member_count",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Members" align="right" />,
      cell: ({ row }) => row.original.member_count,
      meta: { headClassName: "text-right", cellClassName: "text-right" },
    },
    {
      id: "source",
      accessorKey: "source",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Source" />,
      cell: ({ row }) => <EnumBadge value={row.original.source} metadata={DIRECTORY_SOURCES} />,
    },
  ];

  return (
    <PageShell>
      <PageHeader title="Groups" description="Browse directory groups." />

      {query.error ? (
        <Alert variant="destructive">
          <AlertTitle>Failed to Load Groups</AlertTitle>
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
          pagination={state.pagination}
          sorting={state.sorting}
          onPaginationChange={setters.setPagination}
          onSortingChange={setters.setSorting}
          isLoading={query.isLoading}
          rowHref={(row) => ({ to: "/directory/users", search: { group_id: row.id } })}
          toolbar={
            <div className="flex flex-wrap items-center gap-2">
              <DataTableSearch value={draft} onChange={setDraft} placeholder="Search" />
            </div>
          }
          empty={
            <DataTableEmptyState
              icon={<UsersRound />}
              title={hasFilters ? "No Matches" : "No Groups"}
              description={hasFilters ? "No groups matched the current search." : "Groups appear after directory sync."}
            />
          }
        />
      )}
    </PageShell>
  );
}
