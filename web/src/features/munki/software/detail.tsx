import { useNavigate, useParams } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { AppWindow, Pencil, Trash2 } from "lucide-react";
import { useMemo, useState } from "react";

import { DataTableStatic } from "@components/data-table/data-table-static";
import { TableSurface } from "@components/data-table/table-surface";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryGate } from "@components/query-gate";
import { TargetDetails } from "@components/targeting/target-details";
import { Badge } from "@components/ui/badge";
import { Button } from "@components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@components/ui/table";
import { useAuth } from "@features/auth/queries";
import { useLabelNameMap } from "@features/labels/components/label-ref-list";
import { SoftwareArtwork } from "@features/software/software-icon";
import type { MunkiInclude, MunkiPackage, MunkiSoftwareDetail } from "@lib/api";
import { parseRouteID } from "@lib/route-params";
import { formatRelative } from "@lib/utils";

import { MUNKI_SOFTWARE_ACTIONS } from "./actions";
import { MunkiSoftwareDeleteDialog } from "./delete-dialog";
import { useMunkiSoftwareDetail } from "./queries";

const packageColumns: ColumnDef<MunkiPackage>[] = [
  {
    id: "version",
    accessorKey: "version",
    header: () => "Version",
    cell: ({ row }) => (
      <div className="flex min-w-0 items-center gap-2">
        <SoftwareArtwork src={row.original.software.icon_url} />
        <Link
          to="/munki/packages/$id"
          params={{ id: String(row.original.id) }}
          className="min-w-0 truncate font-medium"
        >
          {row.original.version}
        </Link>
      </div>
    ),
  },
  {
    id: "installer_type",
    accessorKey: "installer_type",
    header: () => "Installer",
  },
  {
    id: "updated_at",
    accessorKey: "updated_at",
    header: () => "Updated",
    cell: ({ row }) => formatRelative(row.original.updated_at),
  },
];

export function MunkiSoftwareDetailPage() {
  const params = useParams({ strict: false });
  const navigate = useNavigate();
  const { user } = useAuth();
  const softwareID = parseRouteID(params.id);
  const query = useMunkiSoftwareDetail(softwareID);
  const [deleteOpen, setDeleteOpen] = useState(false);

  if (softwareID === null) {
    return (
      <QueryGate
        title="Failed to load software"
        error={{ message: "Software route is invalid." }}
      />
    );
  }
  if (query.error || !query.data) {
    return (
      <QueryGate
        title="Failed to load software"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }

  const software = query.data;
  const isAdmin = user?.role === "admin";
  return (
    <PageShell className="gap-6">
      <PageHeader
        title="Software Details"
        icon={<SoftwareArtwork src={software.icon_url} fallbackIcon={AppWindow} loading="eager" />}
        meta={`Edited ${formatRelative(software.updated_at)}`}
        actions={
          isAdmin ? (
            <>
              <Button
                size="sm"
                render={<Link to="/munki/software/$id/edit" params={{ id: String(software.id) }} />}
                nativeButton={false}
              >
                <Pencil data-icon="inline-start" />
                Edit
              </Button>
              <Button
                type="button"
                variant="destructive"
                size="sm"
                onClick={() => setDeleteOpen(true)}
              >
                <Trash2 data-icon="inline-start" />
                Delete
              </Button>
            </>
          ) : null
        }
      />

      <KeyValueSection title="Overview">
        <KeyValueRow label="Name" value={software.name} />
        <KeyValueRow label="Display Name" value={software.display_name || "-"} />
        <KeyValueRow label="Description" value={software.description} />
        <KeyValueRow label="Category" value={software.category} />
        <KeyValueRow label="Developer" value={software.developer} />
        <KeyValueRow label="Packages" value={software.packages.length} />
      </KeyValueSection>

      <MunkiSoftwareTargets software={software} />

      <DataTableStatic
        heading="Packages"
        columns={packageColumns}
        data={software.packages}
        empty={<PanelEmptyState>No packages yet</PanelEmptyState>}
      />

      <MunkiSoftwareDeleteDialog
        software={software}
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        onDeleted={() => void navigate({ to: "/munki/software" })}
      />
    </PageShell>
  );
}

function MunkiSoftwareTargets({ software }: { software: MunkiSoftwareDetail }) {
  const labelsByID = useLabelNameMap();
  const packagesByID = useMemo(
    () => new Map(software.packages.map((pkg) => [pkg.id, pkg])),
    [software.packages],
  );

  return (
    <TargetDetails
      include={
        software.targets.include.length > 0 ? (
          <TableSurface variant="embedded">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Label</TableHead>
                  <TableHead>Package</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {software.targets.include.map((target) => (
                  <TableRow key={target.label_id}>
                    <TableCell>
                      <Link
                        to="/labels/$id"
                        params={{ id: String(target.label_id) }}
                        className="font-medium"
                      >
                        {labelsByID.get(target.label_id) ?? `Label ${target.label_id}`}
                      </Link>
                    </TableCell>
                    <TableCell>{munkiPackageLabel(target.package, packagesByID)}</TableCell>
                    <TableCell>
                      <div className="flex flex-wrap gap-1">
                        {target.actions.map((action) => (
                          <Badge key={action} variant="secondary" className="font-normal">
                            {MUNKI_SOFTWARE_ACTIONS[action].name}
                          </Badge>
                        ))}
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableSurface>
        ) : (
          "-"
        )
      }
      excludeLabelIDs={software.targets.exclude.map((target) => target.label_id)}
    />
  );
}

function munkiPackageLabel(
  selector: MunkiInclude["package"],
  packagesByID: ReadonlyMap<number, MunkiPackage>,
) {
  if (selector.strategy === "latest") return "Latest";
  return packagesByID.get(selector.package_id ?? 0)?.version ?? `Package ${selector.package_id}`;
}
