import { useParams, useSearch } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Loader2, Users } from "lucide-react";

import {
  DataTable,
  DataTableColumnHeader,
  DataTableEmptyState,
  DataTableFacetedFilter,
  DataTableSearch,
} from "@/components/data-table";
import { EnumBadge } from "@/components/enum-badge";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { DetailSettings, SettingItem } from "@/components/queries/query-ui";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { USER_ACCESS_ROLE_OPTIONS, USER_ACCESS_ROLES, userAccessRole } from "@/components/users/user-role";
import { useAuth } from "@/hooks/use-auth";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import { useGroup, useGroupMembers } from "@/hooks/use-groups";
import { tableQueryParams, useTablePaginationParams } from "@/hooks/use-table-pagination-params";
import type { User } from "@/hooks/use-users";
import { formatRelative, nonEmpty } from "@/lib/utils";

const MEMBER_STATUS_OPTIONS = [
  { value: "active", label: "Active" },
  { value: "inactive", label: "Inactive" },
];

export function GroupDetailPage() {
  const { groupId } = useParams({ from: "/_authenticated/directory/groups/$groupId" });
  const groupID = Number(groupId);
  const group = useGroup(Number.isFinite(groupID) ? groupID : null);

  if (group.error) {
    return (
      <PageShell>
        <Alert variant="destructive">
          <AlertTitle>Failed to Load Group</AlertTitle>
          <AlertDescription>{group.error.message}</AlertDescription>
        </Alert>
      </PageShell>
    );
  }

  if (!group.data) {
    return (
      <PageShell className="text-muted-foreground flex-row items-center gap-2 text-sm">
        <Loader2 className="size-4 animate-spin" /> Loading Group...
      </PageShell>
    );
  }

  return (
    <PageShell>
      <PageHeader
        title={group.data.display_name}
        description={nonEmpty(group.data.mail_nickname) ?? group.data.external_id}
      />

      <DetailSettings>
        <SettingItem label="Members">{group.data.member_count}</SettingItem>
        <SettingItem label="Synced">{formatRelative(group.data.last_synced_at)}</SettingItem>
        <SettingItem label="External ID">{group.data.external_id}</SettingItem>
      </DetailSettings>

      <GroupMembersTable groupId={groupID} />
    </PageShell>
  );
}

function GroupMembersTable({ groupId }: { groupId: number }) {
  const search = useSearch({ from: "/_authenticated/directory/groups/$groupId" });
  const { state, setters } = useTablePaginationParams();
  const [draft, setDraft] = useDebouncedSearchParam("q");
  const { user: currentUser } = useAuth();

  const query = useGroupMembers(groupId, {
    q: search.q,
    role: search.role,
    status: search.status,
    ...tableQueryParams(state),
  });
  const data = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const hasFilters = !!search.q || !!search.role || !!search.status;

  const columns: ColumnDef<User>[] = [
    {
      id: "name",
      accessorKey: "name",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
      cell: ({ row }) => nonEmpty(row.original.name) ?? row.original.email,
    },
    {
      id: "email",
      accessorKey: "email",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Email" />,
      cell: ({ row }) => row.original.email,
    },
    {
      id: "role",
      accessorKey: "role",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Role" />,
      cell: ({ row }) => <EnumBadge value={userAccessRole(row.original.role)} metadata={USER_ACCESS_ROLES} />,
    },
    {
      id: "status",
      accessorFn: (row) => (row.active ? "active" : "inactive"),
      header: ({ column }) => <DataTableColumnHeader column={column} title="Status" />,
      cell: ({ row }) => (
        <Badge variant={row.original.active ? "outline" : "secondary"}>
          {row.original.active ? "Active" : "Inactive"}
        </Badge>
      ),
    },
    {
      id: "department",
      accessorKey: "department",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Department" />,
      cell: ({ row }) => nonEmpty(row.original.department) ?? "-",
    },
  ];

  if (query.error) {
    return (
      <Alert variant="destructive">
        <AlertTitle>Failed to Load Members</AlertTitle>
        <AlertDescription>{query.error.message}</AlertDescription>
        <Button variant="outline" size="sm" onClick={() => void query.refetch()} className="mt-2 w-fit">
          Retry
        </Button>
      </Alert>
    );
  }

  return (
    <DataTable
      columns={columns}
      data={data}
      totalCount={totalCount}
      pagination={state.pagination}
      sorting={state.sorting}
      onPaginationChange={setters.setPagination}
      onSortingChange={setters.setSorting}
      isLoading={query.isLoading}
      rowHref={(row) =>
        row.id === currentUser?.id
          ? { to: "/account" }
          : { to: "/directory/users/$userId/edit", params: { userId: String(row.id) } }
      }
      toolbar={
        <div className="flex flex-wrap items-center gap-2">
          <DataTableSearch value={draft} onChange={setDraft} placeholder="Search" />
          <DataTableFacetedFilter
            title="Role"
            options={USER_ACCESS_ROLE_OPTIONS}
            selected={search.role ? [search.role] : []}
            onChange={(next) => setters.setFilter("role", next[0])}
            singleSelect
          />
          <DataTableFacetedFilter
            title="Status"
            options={MEMBER_STATUS_OPTIONS}
            selected={search.status ? [search.status] : []}
            onChange={(next) => setters.setFilter("status", next[0])}
            singleSelect
          />
        </div>
      }
      empty={
        <DataTableEmptyState
          icon={<Users />}
          title={hasFilters ? "No Matches" : "No Members"}
          description={hasFilters ? "No members matched the current filters." : "This group has no synced members."}
        />
      }
    />
  );
}
