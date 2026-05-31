import type { ColumnDef } from "@tanstack/react-table";
import { ClipboardList, PackageCheck, PackageSearch } from "lucide-react";
import type { ComponentProps } from "react";

import { DataTable, DataTableEmptyState } from "@/components/data-table";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  useMunkiAssignments,
  useMunkiReleases,
  useMunkiSoftwareTitles,
  type MunkiAssignment,
  type MunkiRelease,
  type MunkiSoftwareTitle,
} from "@/hooks/use-munki";
import { tableQueryParams, useTablePaginationParams } from "@/hooks/use-table-pagination-params";
import { formatRelative } from "@/lib/utils";

type BadgeVariant = ComponentProps<typeof Badge>["variant"];

const intentLabels: Record<MunkiAssignment["intent"], string> = {
  ensure_installed: "Install",
  ensure_absent: "Remove",
  update_if_present: "Update",
  optional: "Optional",
  featured: "Featured",
};

const intentVariants: Record<MunkiAssignment["intent"], BadgeVariant> = {
  ensure_installed: "success",
  ensure_absent: "destructive",
  update_if_present: "info",
  optional: "secondary",
  featured: "warning",
};

export function MunkiSoftwareTitlesPage() {
  const { state, setters } = useTablePaginationParams();
  const query = useMunkiSoftwareTitles(tableQueryParams(state));
  const rows = query.data?.items ?? [];

  const columns: ColumnDef<MunkiSoftwareTitle>[] = [
    {
      id: "name",
      accessorFn: (row) => row.display_name || row.name,
      header: "Name",
      enableSorting: false,
      cell: ({ row }) => row.original.display_name || row.original.name,
    },
    {
      id: "category",
      accessorKey: "category",
      header: "Category",
      enableSorting: false,
      cell: ({ row }) => row.original.category || "-",
    },
    {
      id: "developer",
      accessorKey: "developer",
      header: "Developer",
      enableSorting: false,
      cell: ({ row }) => row.original.developer || "-",
    },
    {
      id: "updated_at",
      accessorKey: "updated_at",
      header: "Updated",
      enableSorting: false,
      cell: ({ row }) => formatRelative(row.original.updated_at),
    },
  ];

  return (
    <PageShell>
      <PageHeader title="Munki Software" description="Desired software titles managed by Munki." />
      <MunkiTableError message={query.error?.message} onRetry={() => void query.refetch()} />
      {!query.error ? (
        <DataTable
          columns={columns}
          data={rows}
          totalCount={query.data?.count ?? 0}
          pagination={state.pagination}
          sorting={state.sorting}
          onPaginationChange={setters.setPagination}
          onSortingChange={setters.setSorting}
          isLoading={query.isLoading}
          empty={
            <DataTableEmptyState
              icon={<PackageSearch />}
              title="No Munki Software"
              description="No desired software titles yet."
            />
          }
        />
      ) : null}
    </PageShell>
  );
}

export function MunkiReleasesPage() {
  const { state, setters } = useTablePaginationParams();
  const query = useMunkiReleases(tableQueryParams(state));
  const rows = query.data?.items ?? [];

  const columns: ColumnDef<MunkiRelease>[] = [
    {
      id: "name",
      accessorKey: "name",
      header: "Name",
      enableSorting: false,
    },
    {
      id: "version",
      accessorKey: "version",
      header: "Version",
      enableSorting: false,
    },
    {
      id: "software_id",
      accessorKey: "software_id",
      header: "Software ID",
      enableSorting: false,
    },
    {
      id: "eligible",
      accessorKey: "eligible",
      header: "Eligible",
      enableSorting: false,
      cell: ({ row }) => (
        <Badge variant={row.original.eligible ? "success" : "secondary"}>{row.original.eligible ? "Yes" : "No"}</Badge>
      ),
    },
    {
      id: "updated_at",
      accessorKey: "updated_at",
      header: "Updated",
      enableSorting: false,
      cell: ({ row }) => formatRelative(row.original.updated_at),
    },
  ];

  return (
    <PageShell>
      <PageHeader title="Munki Releases" description="Pkginfo-backed releases available to assignments." />
      <MunkiTableError message={query.error?.message} onRetry={() => void query.refetch()} />
      {!query.error ? (
        <DataTable
          columns={columns}
          data={rows}
          totalCount={query.data?.count ?? 0}
          pagination={state.pagination}
          sorting={state.sorting}
          onPaginationChange={setters.setPagination}
          onSortingChange={setters.setSorting}
          isLoading={query.isLoading}
          empty={
            <DataTableEmptyState icon={<PackageCheck />} title="No Munki Releases" description="No releases yet." />
          }
        />
      ) : null}
    </PageShell>
  );
}

export function MunkiAssignmentsPage() {
  const { state, setters } = useTablePaginationParams();
  const query = useMunkiAssignments(tableQueryParams(state));
  const rows = query.data?.items ?? [];

  const columns: ColumnDef<MunkiAssignment>[] = [
    {
      id: "release_id",
      accessorKey: "release_id",
      header: "Release ID",
      enableSorting: false,
    },
    {
      id: "intent",
      accessorKey: "intent",
      header: "Intent",
      enableSorting: false,
      cell: ({ row }) => <IntentBadge intent={row.original.intent} />,
    },
    {
      id: "scope",
      header: "Scope",
      enableSorting: false,
      cell: ({ row }) => assignmentScope(row.original),
    },
    {
      id: "updated_at",
      accessorKey: "updated_at",
      header: "Updated",
      enableSorting: false,
      cell: ({ row }) => formatRelative(row.original.updated_at),
    },
  ];

  return (
    <PageShell>
      <PageHeader title="Munki Assignments" description="Desired state resolved by host and label scope." />
      <MunkiTableError message={query.error?.message} onRetry={() => void query.refetch()} />
      {!query.error ? (
        <DataTable
          columns={columns}
          data={rows}
          totalCount={query.data?.count ?? 0}
          pagination={state.pagination}
          sorting={state.sorting}
          onPaginationChange={setters.setPagination}
          onSortingChange={setters.setSorting}
          isLoading={query.isLoading}
          empty={
            <DataTableEmptyState
              icon={<ClipboardList />}
              title="No Munki Assignments"
              description="No assignments yet."
            />
          }
        />
      ) : null}
    </PageShell>
  );
}

function IntentBadge({ intent }: { intent: MunkiAssignment["intent"] }) {
  return <Badge variant={intentVariants[intent]}>{intentLabels[intent]}</Badge>;
}

function assignmentScope(assignment: MunkiAssignment) {
  const include = scopePart(
    assignment.all_hosts,
    assignment.include_label_ids?.length ?? 0,
    assignment.include_host_ids?.length ?? 0,
  );
  const excludes = (assignment.exclude_label_ids?.length ?? 0) + (assignment.exclude_host_ids?.length ?? 0);
  return excludes > 0 ? `${include}; ${excludes} excluded` : include;
}

function scopePart(allHosts: boolean, labelCount: number, hostCount: number) {
  if (allHosts) return "All Hosts";
  const parts: string[] = [];
  if (labelCount > 0) parts.push(`${labelCount} label${labelCount === 1 ? "" : "s"}`);
  if (hostCount > 0) parts.push(`${hostCount} host${hostCount === 1 ? "" : "s"}`);
  return parts.length > 0 ? parts.join(", ") : "-";
}

function MunkiTableError({ message, onRetry }: { message?: string; onRetry: () => void }) {
  if (!message) return null;
  return (
    <Alert variant="destructive">
      <AlertTitle>Failed to Load Munki Data</AlertTitle>
      <AlertDescription>{message}</AlertDescription>
      <Button variant="outline" size="sm" onClick={onRetry} className="mt-2 w-fit">
        Retry
      </Button>
    </Alert>
  );
}
