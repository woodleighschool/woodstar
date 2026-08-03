import { useNavigate, useParams } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { KeyRound, Pencil, Trash2 } from "lucide-react";
import { useState } from "react";

import { AsyncButton } from "@components/async-button";
import { DataTableStatic } from "@components/data-table/data-table-static";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryGate } from "@components/query-gate";
import { Button } from "@components/ui/button";
import { toast } from "@components/ui/toast";
import { useAuth } from "@features/auth/queries";
import { SoftwareArtwork } from "@features/software/software-icon";
import type { MunkiDistributionPointDetail, MunkiPackageState } from "@lib/api";

import { DistributionPointDeleteDialog } from "./delete-dialog";
import {
  BoolBadge,
  MirrorBadge,
  PackageStatusBadge,
  WorkerStatus,
} from "./distribution-point-badges";
import { KeyRevealDialog } from "./key-reveal-dialog";
import { useLiveMunkiDistributionPoint, useRotateMunkiDistributionPointKey } from "./queries";
export function DistributionPointDetailPage() {
  const { id: distributionPointId } = useParams({
    from: "/_authenticated/munki/distribution-points/$id",
  });
  const navigate = useNavigate();
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const id = Number(distributionPointId);
  const query = useLiveMunkiDistributionPoint(Number.isFinite(id) ? id : null);
  const rotate = useRotateMunkiDistributionPointKey();
  const [rotatedKey, setRotatedKey] = useState<string | null>(null);
  const [deleteOpen, setDeleteOpen] = useState(false);
  if (query.error || !query.data) {
    return (
      <QueryGate
        title="Failed to load distribution point"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }
  const point = query.data;
  async function rotateKey() {
    const result = await rotate.mutateAsync(point.id);
    setRotatedKey(result.key);
    toast.add({ title: "Key rotated", type: "success" });
  }
  return (
    <PageShell className="gap-6">
      <PageHeader
        title="Distribution Point Details"
        actions={
          isAdmin ? (
            <>
              <Button
                size="sm"
                render={
                  <Link
                    to="/munki/distribution-points/$id/edit"
                    params={{ id: String(point.id) }}
                  />
                }
                nativeButton={false}
              >
                <Pencil data-icon="inline-start" />
                Edit
              </Button>
              <AsyncButton
                type="button"
                variant="outline"
                size="sm"
                isPending={rotate.isPending}
                onClick={() => void rotateKey()}
                icon={<KeyRound data-icon="inline-start" />}
              >
                Rotate Key
              </AsyncButton>
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
        <KeyValueRow label="Name" value={point.name} />
        <KeyValueRow label="Worker Status" value={<WorkerStatus worker={point.worker} />} />
        <KeyValueRow label="Enabled" value={<BoolBadge value={point.enabled} label="Enabled" />} />
        <KeyValueRow label="Mirror" value={<MirrorBadge packages={point.packages} />} />
        <KeyValueRow
          label="Worker Version"
          value={point.worker?.build_version ?? <span className="text-muted-foreground">-</span>}
        />
        <KeyValueRow label="Base URL" value={point.client_base_url} />
        <KeyValueRow label="Client CIDRs" value={<CidrList cidrs={point.client_cidrs} />} />
      </KeyValueSection>

      <PackageStateTable packages={point.packages} />

      <KeyRevealDialog
        title="Rotated Distribution Point Key"
        description="Copy this key into the worker configuration. It is shown only once."
        value={rotatedKey ?? ""}
        open={rotatedKey !== null}
        onOpenChange={(open) => {
          if (!open) setRotatedKey(null);
        }}
      />

      <DistributionPointDeleteDialog
        point={point}
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        onDeleted={() => void navigate({ to: "/munki/distribution-points" })}
      />
    </PageShell>
  );
}
function CidrList({ cidrs }: { cidrs: MunkiDistributionPointDetail["client_cidrs"] }) {
  if (cidrs.length === 0) return <span className="text-muted-foreground">-</span>;
  return (
    <div className="flex flex-wrap gap-1.5">
      {cidrs.map((cidr) => (
        <span key={cidr} className="text-sm">
          {cidr}
        </span>
      ))}
    </div>
  );
}
function PackageStateTable({ packages }: { packages: MunkiPackageState[] }) {
  return (
    <DataTableStatic
      heading="Packages"
      columns={packageStateColumns}
      data={packages}
      empty={<PanelEmptyState>No mirrored packages.</PanelEmptyState>}
    />
  );
}

const packageStateColumns: ColumnDef<MunkiPackageState>[] = [
  {
    id: "package",
    header: () => "Package",
    cell: ({ row }) => (
      <div className="flex items-center gap-2">
        <SoftwareArtwork src={row.original.software_icon_url} />
        <Link
          to="/munki/packages/$id"
          params={{ id: String(row.original.package_id) }}
          className="min-w-0 truncate font-medium"
          title={`${row.original.name} ${row.original.version}`}
        >
          {row.original.name} {row.original.version}
        </Link>
      </div>
    ),
  },
  {
    id: "status",
    header: () => "Status",
    cell: ({ row }) => <PackageStatusBadge status={row.original.status} />,
  },
  {
    id: "error",
    header: () => "Error",
    cell: ({ row }) => packageErrorText(row.original.error),
  },
];

function packageErrorText(error: string | undefined) {
  if (error === undefined || error === "") {
    return <span className="text-muted-foreground">-</span>;
  }
  return <span className="block max-w-xl wrap-break-word whitespace-normal">{error}</span>;
}
