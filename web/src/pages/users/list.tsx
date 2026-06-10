import { useSearch } from "@tanstack/react-router";
import { UserPlus } from "lucide-react";
import { useState } from "react";

import { FilterChip } from "@/components/filter-controls";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { Button } from "@/components/ui/button";
import { UserDeleteDialog } from "@/components/users/user-delete-dialog";
import { UserFormDialog } from "@/components/users/user-form-dialog";
import { UsersTable, UsersToolbar } from "@/components/users/users-table";
import { useAuth } from "@/hooks/use-auth";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import { useGroup, type Group } from "@/hooks/use-groups";
import { tableQueryParams, useTablePaginationParams } from "@/hooks/use-table-pagination-params";
import { useUsers, type User } from "@/hooks/use-users";

export function UserListPage() {
  const search = useSearch({ from: "/_authenticated/directory/users/" });
  const { state, setters } = useTablePaginationParams();
  const [draft, setDraft] = useDebouncedSearchParam("q");
  const { user: currentUser } = useAuth();
  const [createOpen, setCreateOpen] = useState(false);
  const [deleting, setDeleting] = useState<User | null>(null);
  const groupID = search.group_id;
  const group = useGroup(groupID ?? null);

  const query = useUsers({
    q: search.q,
    role: search.role,
    source: search.source,
    group_id: groupID,
    ...tableQueryParams(state),
  });
  const data = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const hasFilters = !!search.q || !!search.role || !!search.source || groupID !== undefined;
  const groupLabel = groupFilterLabel({ group: group.data, groupID });

  return (
    <PageShell>
      <PageHeader
        title="Users"
        description="Manage directory and local users."
        context={
          groupLabel ? (
            <FilterChip label="Group" value={groupLabel} onRemove={() => setters.setFilter("group_id", undefined)} />
          ) : null
        }
        actions={
          <Button size="sm" onClick={() => setCreateOpen(true)}>
            <UserPlus data-icon="inline-start" />
            Create
          </Button>
        }
      />

      <div>
        <UsersTable
          data={data}
          totalCount={totalCount}
          query={query}
          currentUserId={currentUser?.id ?? null}
          pagination={state.pagination}
          sorting={state.sorting}
          onPaginationChange={setters.setPagination}
          onSortingChange={setters.setSorting}
          toolbar={
            <UsersToolbar
              draft={draft}
              onDraftChange={setDraft}
              role={search.role}
              source={search.source}
              onFilterChange={setters.setFilter}
            />
          }
          hasFilters={hasFilters}
          onDelete={setDeleting}
        />
      </div>

      <UserFormDialog open={createOpen} onOpenChange={setCreateOpen} />

      <UserDeleteDialog
        open={deleting !== null}
        onOpenChange={(open) => {
          if (!open) setDeleting(null);
        }}
        user={deleting}
      />
    </PageShell>
  );
}

function groupFilterLabel({ group, groupID }: { group: Group | undefined; groupID: number | undefined }) {
  if (groupID === undefined) return undefined;
  return group?.display_name ?? `Group #${groupID}`;
}
