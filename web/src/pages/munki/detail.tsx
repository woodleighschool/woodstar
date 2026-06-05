import { Link, useParams } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Loader2, PackageCheck, Pencil, Plus } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import { DataTable, DataTableColumnHeader, DataTableEmptyState, DataTableRowDragHandle } from "@/components/data-table";
import type { LabelChip } from "@/components/labels/label-chip-utils";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { MunkiIcon } from "@/components/munki/munki-icon";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ButtonGroup } from "@/components/ui/button-group";
import { useReorderMunkiAssignments, type MunkiAssignment } from "@/hooks/munki/assignments";
import { type MunkiPackage } from "@/hooks/munki/packages";
import { useMunkiSoftwareTitle } from "@/hooks/munki/software-titles";
import { useLabels } from "@/hooks/use-labels";
import { MAX_PAGE_SIZE } from "@/lib/pagination";
import { formatRelative } from "@/lib/utils";

const actionLabels: Record<string, string> = {
  install: "Managed Installs",
  remove: "Managed Uninstalls",
  update_if_present: "Managed Updates",
  none: "None",
};

const packageSelectionLabels: Record<string, string> = {
  latest_eligible: "Latest compatible",
  specific_package: "Pinned package",
};

export function MunkiSoftwareTitleDetailPage() {
  const params = useParams({ strict: false });
  const softwareId = Number(params.softwareId);
  const query = useMunkiSoftwareTitle(Number.isFinite(softwareId) ? softwareId : null);
  const labels = useLabels({ page_size: MAX_PAGE_SIZE, sort: "name.asc" });
  const reorder = useReorderMunkiAssignments();
  const [reorderEnabled, setReorderEnabled] = useState(false);
  const [orderedAssignments, setOrderedAssignments] = useState<MunkiAssignment[]>([]);
  const software = query.data;
  const packages = software?.packages ?? [];
  const assignments = useMemo(() => software?.assignments ?? [], [software?.assignments]);
  const labelsByID = useMemo<ReadonlyMap<number, LabelChip>>(
    () => new Map((labels.data?.items ?? []).map((label) => [label.id, label])),
    [labels.data?.items],
  );
  let title = "Software";
  if (software?.name) title = software.name;
  if (software?.display_name) title = software.display_name;

  useEffect(() => {
    setOrderedAssignments(assignments);
  }, [assignments]);

  function moveAssignments(next: MunkiAssignment[]) {
    setOrderedAssignments(next.map((assignment, index) => ({ ...assignment, priority: index + 1 })));
  }

  function saveOrder() {
    reorder.mutate(
      { softwareId, orderedIds: orderedAssignments.map((assignment) => assignment.id) },
      {
        onSuccess: () => setReorderEnabled(false),
        onError: () => setOrderedAssignments(assignments),
      },
    );
  }

  const packageColumns: ColumnDef<MunkiPackage>[] = [
    {
      id: "version",
      accessorKey: "version",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Version" />,
      cell: ({ row }) => (
        <div className="flex min-w-0 items-center gap-3">
          <PackageIconView pkg={row.original} />
          <div className="min-w-0">
            <div className="truncate font-medium">{row.original.version}</div>
            <div className="text-muted-foreground truncate text-xs">
              {row.original.display_name || row.original.name}
            </div>
          </div>
        </div>
      ),
    },
    {
      id: "installer_type",
      header: "Installer",
      enableSorting: false,
      cell: ({ row }) => installerTypeLabel(row.original.installer_type),
    },
    {
      id: "behavior",
      header: "Behavior",
      enableSorting: false,
      cell: ({ row }) => <PackageBehavior pkg={row.original} />,
    },
    {
      id: "eligible",
      accessorKey: "eligible",
      header: "Available",
      enableSorting: false,
      cell: ({ row }) => (row.original.eligible ? "Yes" : "No"),
    },
    {
      id: "updated_at",
      accessorKey: "updated_at",
      header: "Updated",
      enableSorting: false,
      cell: ({ row }) => formatRelative(row.original.updated_at),
    },
  ];

  const assignmentColumns: ColumnDef<MunkiAssignment>[] = [
    ...(reorderEnabled
      ? ([
          {
            id: "drag",
            header: () => null,
            enableSorting: false,
            enableHiding: false,
            cell: () => <DataTableRowDragHandle />,
            meta: { headClassName: "w-10", cellClassName: "w-10" },
          },
        ] satisfies ColumnDef<MunkiAssignment>[])
      : []),
    {
      id: "priority",
      accessorKey: "priority",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Priority" />,
      cell: ({ row }) => row.original.priority,
      meta: { headClassName: "w-20", cellClassName: "w-20" },
    },
    {
      id: "effect",
      accessorKey: "effect",
      header: "Effect",
      enableSorting: false,
      cell: ({ row }) => effectLabel(row.original.effect),
    },
    {
      id: "label",
      accessorKey: "label_id",
      header: "Label",
      enableSorting: false,
      cell: ({ row }) => labelsByID.get(row.original.label_id)?.name ?? `Label ${row.original.label_id}`,
    },
    {
      id: "selection",
      accessorKey: "package_selection",
      header: "Package",
      enableSorting: false,
      cell: ({ row }) => assignmentPackageLabel(row.original),
    },
    {
      id: "action",
      accessorKey: "action",
      header: "Managed",
      enableSorting: false,
      cell: ({ row }) => assignmentActionLabel(row.original),
    },
    {
      id: "msc",
      header: "Availability",
      enableSorting: false,
      cell: ({ row }) => assignmentMSCSections(row.original).join(", ") || "None",
    },
  ];

  return (
    <PageShell>
      <PageHeader
        title={title}
        description="Software is targeted once. Pkginfos provide the versioned Munki candidates each assignment can render."
        leading={software ? <MunkiIcon iconUrl={software.icon_url} size="lg" /> : undefined}
        actions={
          software ? (
            <>
              <Button asChild size="sm" variant="outline">
                <Link to="/munki/software-titles/$softwareId/edit" params={{ softwareId: String(software.id) }}>
                  <Pencil data-icon="inline-start" />
                  Edit
                </Link>
              </Button>
              <Button asChild size="sm" variant="outline">
                <Link to="/munki/software-titles/$softwareId/packages/new" params={{ softwareId: String(software.id) }}>
                  <Plus data-icon="inline-start" />
                  Package
                </Link>
              </Button>
              <Button asChild size="sm">
                <Link
                  to="/munki/software-titles/$softwareId/assignments/new"
                  params={{ softwareId: String(software.id) }}
                >
                  <Plus data-icon="inline-start" />
                  Assignment
                </Link>
              </Button>
            </>
          ) : null
        }
      />

      {query.error ? (
        <Alert variant="destructive">
          <AlertTitle>Failed to Load Software</AlertTitle>
          <AlertDescription>{query.error.message}</AlertDescription>
        </Alert>
      ) : (
        <div className="flex flex-col gap-8">
          <section className="flex flex-col gap-3">
            <div className="flex items-start justify-between gap-3">
              <div className="flex flex-col gap-1">
                <h2 className="text-lg font-semibold">Packages</h2>
                <p className="text-muted-foreground max-w-3xl text-sm">
                  Each row is one pkginfo. Multiple versions can share the same Munki name.
                </p>
              </div>
            </div>
            <DataTable
              columns={packageColumns}
              data={packages}
              totalCount={packages.length}
              pagination={{ pageIndex: 0, pageSize: Math.max(packages.length, 1) }}
              sorting={[]}
              onPaginationChange={() => undefined}
              onSortingChange={() => undefined}
              isLoading={query.isLoading}
              clientSort
              rowHref={(row) => ({
                to: "/munki/software-titles/$softwareId/packages/$packageId/edit",
                params: { softwareId: String(row.software_id), packageId: String(row.id) },
              })}
              empty={
                <DataTableEmptyState
                  icon={<PackageCheck />}
                  title="No Packages"
                  description="Add at least one pkginfo before assigning this software."
                />
              }
            />
          </section>

          <section className="flex flex-col gap-3">
            <div className="flex items-start justify-between gap-3">
              <div className="flex flex-col gap-1">
                <h2 className="text-lg font-semibold">Assignments</h2>
                <p className="text-muted-foreground max-w-3xl text-sm">
                  Woodstar resolves this list top to bottom for each device. Latest-compatible assignments render the
                  Munki name and let the client choose from eligible pkginfos.
                </p>
              </div>
              <ButtonGroup>
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  disabled={reorderEnabled || orderedAssignments.length < 2 || query.isLoading}
                  onClick={() => setReorderEnabled(true)}
                >
                  Edit Order
                </Button>
                {reorderEnabled ? (
                  <>
                    <Button type="button" size="sm" disabled={reorder.isPending} onClick={saveOrder}>
                      {reorder.isPending ? <Loader2 data-icon="inline-start" className="animate-spin" /> : null}
                      Save
                    </Button>
                    <Button type="button" size="sm" variant="outline" onClick={() => setReorderEnabled(false)}>
                      Cancel
                    </Button>
                  </>
                ) : null}
              </ButtonGroup>
            </div>
            <DataTable
              columns={assignmentColumns}
              data={reorderEnabled ? orderedAssignments : assignments}
              totalCount={assignments.length}
              pagination={{ pageIndex: 0, pageSize: Math.max(assignments.length, 1) }}
              sorting={[]}
              onPaginationChange={() => undefined}
              onSortingChange={() => undefined}
              isLoading={query.isLoading}
              clientSort
              rowHref={
                reorderEnabled
                  ? undefined
                  : (row) => ({
                      to: "/munki/software-titles/$softwareId/assignments/$assignmentId/edit",
                      params: { softwareId: String(row.software_id), assignmentId: String(row.id) },
                    })
              }
              rowReorderDisabled={!reorderEnabled || reorder.isPending || orderedAssignments.length <= 1}
              onRowReorder={moveAssignments}
              empty={
                <DataTableEmptyState
                  icon={<PackageCheck />}
                  title="No Assignments"
                  description="Add an assignment to put this software into matching host manifests."
                />
              }
            />
          </section>
        </div>
      )}
    </PageShell>
  );
}

function effectLabel(effect: MunkiAssignment["effect"]) {
  return effect === "include" ? "Include" : "Exclude";
}

function assignmentPackageLabel(assignment: MunkiAssignment) {
  if (assignment.effect === "exclude") return <span className="text-muted-foreground">None</span>;
  if (assignment.package_selection === "specific_package") {
    return assignment.pinned_package_version
      ? `${assignment.pinned_package_name ?? "Pinned"} ${assignment.pinned_package_version}`
      : "Pinned package";
  }
  return packageSelectionLabels[assignment.package_selection ?? ""] ?? assignment.package_selection;
}

function assignmentActionLabel(assignment: MunkiAssignment) {
  if (assignment.effect === "exclude") return <span className="text-muted-foreground">Excluded</span>;
  return actionLabels[assignment.action ?? ""] ?? assignment.action;
}

function assignmentMSCSections(assignment: MunkiAssignment) {
  if (assignment.effect === "exclude") return [];
  const sections: string[] = [];
  if (assignment.optional_install) sections.push("Optional Installs");
  if (assignment.featured_item) sections.push("Featured Items");
  return sections;
}

function PackageIconView({ pkg }: { pkg: MunkiPackage }) {
  return <MunkiIcon iconUrl={pkg.icon_url} size="md" />;
}

function PackageBehavior({ pkg }: { pkg: MunkiPackage }) {
  const values = [
    pkg.unattended_install ? "Unattended" : "",
    pkg.uninstallable ? "Uninstallable" : "",
    pkg.on_demand ? "On demand" : "",
    pkg.restart_action && pkg.restart_action !== "None" ? pkg.restart_action : "",
  ].filter(Boolean);
  if (values.length === 0) {
    return <span className="text-muted-foreground">Standard</span>;
  }
  return (
    <div className="flex flex-wrap gap-1">
      {values.map((value) => (
        <Badge key={value} variant="secondary" className="font-normal">
          {value}
        </Badge>
      ))}
    </div>
  );
}

function installerTypeLabel(value: MunkiPackage["installer_type"]) {
  if (value === "pkg") return "Package";
  return value;
}
