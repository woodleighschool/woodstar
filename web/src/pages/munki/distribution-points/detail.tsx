import { Link, useNavigate, useParams } from "@tanstack/react-router";
import { KeyRound, Pencil, Trash2 } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

import { ConfirmDialog } from "@/components/confirm-dialog";
import { EmptyPanel } from "@/components/empty-panel";
import { KeyValueGrid, KeyValueItem } from "@/components/key-value";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { MunkiIcon } from "@/components/munki/munki-icon";
import { QueryGate } from "@/components/query-gate";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
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
  const { distributionPointId } = useParams({
    from: "/_authenticated/munki/distribution-points/$distributionPointId",
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
                    to="/munki/distribution-points/$distributionPointId/edit"
                    params={{ distributionPointId: String(point.id) }}
                  />
                }
                nativeButton={false}
              >
                <Pencil data-icon="inline-start" />
                Edit
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={rotate.isPending}
                onClick={() => void rotateKey()}
              >
                <KeyRound data-icon="inline-start" />
                Rotate Key
              </Button>
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
        {packages.length === 0 ? (
          <EmptyPanel>No mirrored packages.</EmptyPanel>
        ) : (
          <div className="overflow-hidden rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Package</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Error</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {packages.map((pkg) => (
                  <TableRow key={pkg.package_id}>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <MunkiIcon iconUrl={pkg.software_icon_url} />
                        <Link
                          to="/munki/packages/$packageId/edit"
                          params={{ packageId: String(pkg.package_id) }}
                          className="min-w-0 truncate font-medium hover:underline"
                        >
                          {pkg.name} {pkg.version}
                        </Link>
                      </div>
                    </TableCell>
                    <TableCell>
                      <PackageStatusBadge status={pkg.status} />
                    </TableCell>
                    <TableCell>{packageErrorText(pkg.error)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
function packageErrorText(error: string | undefined) {
  if (error === undefined || error === "") {
    return <span className="text-muted-foreground">-</span>;
  }
  return error;
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
