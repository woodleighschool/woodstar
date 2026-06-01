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

const intentLabels: Record<string, string> = {
  ensure_installed: "Install and update",
  ensure_absent: "Remove",
  update_if_present: "Update if present",
  optional: "Self Service",
  featured: "Featured",
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
      id: "package",
      accessorKey: "package_version",
      header: "Package",
      enableSorting: false,
      cell: ({ row }) => row.original.package_version,
    },
    {
      id: "intent",
      accessorKey: "intent",
      header: "Intent",
      enableSorting: false,
      cell: ({ row }) => intentLabels[row.original.intent] ?? row.original.intent,
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
        description="Packages define the versioned Munki metadata. Deployments decide which package and intent each device receives."
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
                  Deployment
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
                  Package rows are typed pkginfo inputs. Woodstar renders them into Munki catalogs.
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
                  description="Add a package version before deploying this software."
                />
              }
            />
          </section>

          <section className="flex flex-col gap-3">
            <div className="flex items-start justify-between gap-3">
              <div className="flex flex-col gap-1">
                <h2 className="text-lg font-semibold">Deployments</h2>
                <p className="text-muted-foreground max-w-3xl text-sm">
                  Woodstar resolves this list top to bottom for each device. Put narrow removals above broad installs
                  when they should win.
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
                  title="No Deployments"
                  description="Add a deployment to put this package into a host manifest."
                />
              }
            />
          </section>
        </div>
      )}
    </PageShell>
  );
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
