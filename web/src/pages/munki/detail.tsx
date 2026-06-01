import { Link, useParams } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Loader2, PackageCheck, Package as PackageIcon, Plus } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import { DataTable, DataTableColumnHeader, DataTableEmptyState, DataTableRowDragHandle } from "@/components/data-table";
import type { LabelChip } from "@/components/labels/label-chip-utils";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { TargetLabelsCell } from "@/components/santa/target-labels-cell";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ButtonGroup } from "@/components/ui/button-group";
import { useLabels } from "@/hooks/use-labels";
import {
  useMunkiSoftwareTitle,
  useReorderMunkiDeployments,
  type MunkiDeployment,
  type MunkiPackage,
} from "@/hooks/use-munki";
import { MAX_PAGE_SIZE } from "@/lib/pagination";
import { formatRelative } from "@/lib/utils";

const actionLabels: Record<string, string> = {
  install: "Install",
  remove: "Remove",
  update_if_present: "Update if present",
  none: "No automatic action",
};

const selfServiceLabels: Record<string, string> = {
  hidden: "Hidden",
  available: "Available",
  featured: "Featured",
  default: "Default",
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
  const reorder = useReorderMunkiDeployments();
  const [reorderEnabled, setReorderEnabled] = useState(false);
  const [orderedDeployments, setOrderedDeployments] = useState<MunkiDeployment[]>([]);
  const software = query.data;
  const packages = software?.packages ?? [];
  const deployments = useMemo(() => software?.deployments ?? [], [software?.deployments]);
  const labelsByID = useMemo<ReadonlyMap<number, LabelChip>>(
    () => new Map((labels.data?.items ?? []).map((label) => [label.id, label])),
    [labels.data?.items],
  );
  let title = "Software";
  if (software?.name) title = software.name;
  if (software?.display_name) title = software.display_name;

  useEffect(() => {
    setOrderedDeployments(deployments);
  }, [deployments]);

  function moveDeployments(next: MunkiDeployment[]) {
    setOrderedDeployments(next.map((deployment, position) => ({ ...deployment, position })));
  }

  function saveOrder() {
    reorder.mutate(
      { softwareId, orderedIds: orderedDeployments.map((deployment) => deployment.id) },
      {
        onSuccess: () => setReorderEnabled(false),
        onError: () => setOrderedDeployments(deployments),
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
            <Link
              to="/munki/software-titles/$softwareId/packages/$packageId/edit"
              params={{ softwareId: String(row.original.software_id), packageId: String(row.original.id) }}
              className="hover:text-foreground truncate font-medium underline-offset-4 hover:underline"
            >
              {row.original.version}
            </Link>
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

  const deploymentColumns: ColumnDef<MunkiDeployment>[] = [
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
        ] satisfies ColumnDef<MunkiDeployment>[])
      : []),
    {
      id: "position",
      accessorKey: "position",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Order" />,
      cell: ({ row }) => row.original.position + 1,
      meta: { headClassName: "w-20", cellClassName: "w-20" },
    },
    {
      id: "selection",
      accessorKey: "package_selection",
      header: "Package",
      enableSorting: false,
      cell: ({ row }) => (
        <Link
          to="/munki/software-titles/$softwareId/deployments/$deploymentId/edit"
          params={{ softwareId: String(row.original.software_id), deploymentId: String(row.original.id) }}
          className="hover:text-foreground underline-offset-4 hover:underline"
        >
          {deploymentPackageLabel(row.original)}
        </Link>
      ),
    },
    {
      id: "action",
      accessorKey: "action",
      header: "Action",
      enableSorting: false,
      cell: ({ row }) => actionLabels[row.original.action] ?? row.original.action,
    },
    {
      id: "self_service",
      accessorKey: "self_service",
      header: "Self Service",
      enableSorting: false,
      cell: ({ row }) => selfServiceLabels[row.original.self_service] ?? row.original.self_service,
    },
    {
      id: "targets",
      header: "Targets",
      enableSorting: false,
      cell: ({ row }) =>
        row.original.all_hosts ? (
          "All devices"
        ) : (
          <TargetLabelsCell labelIDs={row.original.include_label_ids ?? []} labelsByID={labelsByID} />
        ),
    },
    {
      id: "exclusions",
      header: "Exclusions",
      enableSorting: false,
      cell: ({ row }) =>
        (row.original.exclude_label_ids ?? []).length === 0 ? (
          "None"
        ) : (
          <TargetLabelsCell labelIDs={row.original.exclude_label_ids ?? []} labelsByID={labelsByID} />
        ),
    },
  ];

  return (
    <PageShell>
      <PageHeader
        title={title}
        description="Software is targeted once. Pkginfos provide the versioned Munki candidates each assignment can render."
        actions={
          software ? (
            <>
              <Button asChild size="sm" variant="outline">
                <Link to="/munki/software-titles/$softwareId/packages/new" params={{ softwareId: String(software.id) }}>
                  <Plus data-icon="inline-start" />
                  Package
                </Link>
              </Button>
              <Button asChild size="sm">
                <Link
                  to="/munki/software-titles/$softwareId/deployments/new"
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
                  disabled={reorderEnabled || orderedDeployments.length < 2 || query.isLoading}
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
              columns={deploymentColumns}
              data={reorderEnabled ? orderedDeployments : deployments}
              totalCount={deployments.length}
              pagination={{ pageIndex: 0, pageSize: Math.max(deployments.length, 1) }}
              sorting={[]}
              onPaginationChange={() => undefined}
              onSortingChange={() => undefined}
              isLoading={query.isLoading}
              clientSort
              rowReorderDisabled={!reorderEnabled || reorder.isPending || orderedDeployments.length <= 1}
              onRowReorder={moveDeployments}
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

function deploymentPackageLabel(deployment: MunkiDeployment) {
  if (deployment.package_selection === "specific_package") {
    return deployment.pinned_package_version
      ? `${deployment.pinned_package_name ?? "Pinned"} ${deployment.pinned_package_version}`
      : "Pinned package";
  }
  return packageSelectionLabels[deployment.package_selection] ?? deployment.package_selection;
}

function PackageIconView({ pkg }: { pkg: MunkiPackage }) {
  const label = pkg.display_name || pkg.name;
  if (pkg.icon_url) {
    return (
      <img
        src={pkg.icon_url}
        alt=""
        className="bg-muted size-9 shrink-0 rounded-md border object-contain p-1"
        loading="lazy"
      />
    );
  }
  return (
    <div className="bg-muted text-muted-foreground flex size-9 shrink-0 items-center justify-center rounded-md border">
      <PackageIcon aria-label={`${label} package`} className="size-4" />
    </div>
  );
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
