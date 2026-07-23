import { Link, useNavigate, useParams } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { KeyRound, Pencil, Trash2 } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

import { ConfirmDialog } from "@/components/confirm-dialog";
import { DataTableStatic } from "@/components/data-table/data-table-static";
import { EmptyPanel } from "@/components/empty-panel";
import { KeyValueGrid, KeyValueItem } from "@/components/key-value";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { PendingButton } from "@/components/pending-button";
import { QueryGate } from "@/components/query-gate";
import { SoftwareArtwork } from "@/components/software/software-icon";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useAuth } from "@/hooks/use-auth";
import {
  useDeleteMunkiDistributionPoint,
  useLiveMunkiDistributionPoint,
  useRotateMunkiDistributionPointKey,
} from "@/hooks/use-munki-distribution-points";
import type { MunkiDistributionPointDetail, MunkiPackageState } from "@/lib/api";
import {
  BoolBadge,
  ConnectionBadge,
  MirrorBadge,
  PackageStatusBadge,
} from "@/pages/munki/distribution-points/distribution-point-badges";
import { KeyRevealDialog } from "@/pages/munki/distribution-points/key-reveal-dialog";
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
  const remove = useDeleteMunkiDistributionPoint();
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
    toast.success("Key rotated");
  }
  async function deletePoint() {
    await remove.mutateAsync(point.id);
    setDeleteOpen(false);
    toast.success("Distribution point deleted");
    void navigate({ to: "/munki/distribution-points" });
  }
  return (
    <PageShell className="gap-6">
      <PageHeader
        title={point.name}
        context={<ConnectionBadge online={point.online} />}
        actions={
          isAdmin ? (
            <>
              <Button
                variant="outline"
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
              <PendingButton
                type="button"
                variant="outline"
                size="sm"
                isPending={rotate.isPending}
                onClick={() => void rotateKey()}
                icon={<KeyRound data-icon="inline-start" />}
              >
                Rotate Key
              </PendingButton>
              <Button type="button" variant="outline" size="sm" onClick={() => setDeleteOpen(true)}>
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
            <KeyValueItem
              label="Enabled"
              value={<BoolBadge value={point.enabled} label="Enabled" />}
            />
            <KeyValueItem label="Mirror" value={<MirrorBadge packages={point.packages} />} />
            <KeyValueItem label="Base URL" value={point.client_base_url} />
            <KeyValueItem
              label="Client CIDRs"
              value={<CidrList cidrs={point.client_cidrs} />}
              className="sm:col-span-2"
            />
          </KeyValueGrid>
        </CardContent>
      </Card>

      <PackageStateCard packages={point.packages} />

      <KeyRevealDialog
        title="Rotated Distribution Point Key"
        description="Copy this key into the worker configuration. It is shown only once."
        value={rotatedKey ?? ""}
        open={rotatedKey !== null}
        onOpenChange={(open) => {
          if (!open) setRotatedKey(null);
        }}
      />

      <DeleteDialog
        open={deleteOpen}
        pending={remove.isPending}
        onOpenChange={setDeleteOpen}
        onConfirm={() => void deletePoint()}
      />
    </PageShell>
  );
}
function CidrList({ cidrs }: { cidrs: MunkiDistributionPointDetail["client_cidrs"] }) {
  if (cidrs.length === 0) return <span className="text-muted-foreground">-</span>;
  return (
    <div className="flex flex-wrap gap-1.5">
      {cidrs.map((cidr) => (
        <code key={cidr} className="rounded-sm bg-muted px-1.5 py-0.5 font-mono text-xs">
          {cidr}
        </code>
      ))}
    </div>
  );
}
function PackageStateCard({ packages }: { packages: MunkiPackageState[] }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Packages</CardTitle>
      </CardHeader>
      <CardContent>
        <DataTableStatic
          columns={packageStateColumns}
          data={packages}
          empty={<EmptyPanel>No mirrored packages.</EmptyPanel>}
        />
      </CardContent>
    </Card>
  );
}

const packageStateColumns: ColumnDef<MunkiPackageState>[] = [
  {
    id: "package",
    header: () => "Package",
    cell: ({ row }) => (
      <div className="flex max-w-xl min-w-0 items-center gap-2">
        <SoftwareArtwork src={row.original.software_icon_url} />
        <Link
          to="/munki/packages/$id/edit"
          params={{ id: String(row.original.package_id) }}
          className="min-w-0 truncate font-medium"
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
function DeleteDialog({
  open,
  pending,
  onOpenChange,
  onConfirm,
}: {
  open: boolean;
  pending: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
}) {
  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Delete distribution point?"
      description="Clients stop being redirected to this distribution point."
      confirmLabel="Delete"
      variant="destructive"
      pending={pending}
      onConfirm={onConfirm}
    />
  );
}
