import { useNavigate, useParams } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Pencil, Trash2 } from "lucide-react";
import { useState } from "react";

import { DataTableStatic } from "@components/data-table/data-table-static";
import { KeyValueGrid, KeyValueItem } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryGate } from "@components/query-gate";
import { Button } from "@components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@components/ui/card";
import { useAuth } from "@features/auth/queries";
import { SoftwareArtwork } from "@features/software/software-icon";
import type { MunkiPackage } from "@lib/api";
import { parseRouteID } from "@lib/route-params";
import { formatRelative } from "@lib/utils";

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
        description={software.description || undefined}
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

      <Card>
        <CardContent>
          <KeyValueGrid>
            <KeyValueItem label="Name" value={software.name} />
            <KeyValueItem label="Display Name" value={software.display_name || "-"} />
            <KeyValueItem label="Category" value={software.category} />
            <KeyValueItem label="Developer" value={software.developer} />
            <KeyValueItem label="Packages" value={software.packages.length} />
            <KeyValueItem label="Updated" value={formatRelative(software.updated_at)} />
            <KeyValueItem label="Includes" value={software.targets.include.length} />
            <KeyValueItem label="Excludes" value={software.targets.exclude.length} />
          </KeyValueGrid>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Packages</CardTitle>
        </CardHeader>
        <CardContent>
          <DataTableStatic
            columns={packageColumns}
            data={software.packages}
            empty={<PanelEmptyState>No packages yet</PanelEmptyState>}
          />
        </CardContent>
      </Card>

      <MunkiSoftwareDeleteDialog
        software={software}
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        onDeleted={() => void navigate({ to: "/munki/software" })}
      />
    </PageShell>
  );
}
