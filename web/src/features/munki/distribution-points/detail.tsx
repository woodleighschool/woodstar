import { useNavigate, useParams } from "@tanstack/react-router";
import { KeyRound, Pencil, Smile, Trash2 } from "lucide-react";
import { useMemo, useState } from "react";

import { AsyncButton } from "@components/async-button";
import { BooleanIndicator } from "@components/boolean-indicator";
import { DataTableStatic } from "@components/data-table/data-table-static";
import type { DataTableColumnDef } from "@components/data-table/types";
import { EnumStatusIndicator } from "@components/enum-status-indicator";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link, TextLink } from "@components/link";
import { QueryGate } from "@components/query-gate";
import { TokenList } from "@components/token-list";
import { Button } from "@components/ui/button";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@components/ui/empty";
import { toast } from "@components/ui/toast";
import { useCan } from "@features/authz/access";
import { SoftwareArtwork } from "@features/software/software-icon";
import type { MunkiDistributionPointDetail, MunkiPackageState } from "@lib/api";

import { DistributionPointDeleteDialog } from "./delete-dialog";
import { KeyRevealDialog } from "./key-reveal-dialog";
import {
  DISTRIBUTION_POINT_WORKER_STATUSES,
  distributionPointWorkerStatus,
  packageStateLabel,
} from "./model";
import { useLiveMunkiDistributionPoint, useRotateMunkiDistributionPointKey } from "./queries";
export function DistributionPointDetailPage() {
  const { id: distributionPointId } = useParams({
    from: "/_authenticated/munki/distribution-points/$id",
  });
  const navigate = useNavigate();
  const canEdit = useCan({ resource: "munki.distribution-points", access: "edit" });
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
    toast.add({ title: "Key Rotated", type: "success" });
  }
  return (
    <PageShell className="gap-6">
      <PageHeader
        title="Distribution Point Details"
        actions={
          canEdit ? (
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
        <KeyValueRow
          label="Status"
          value={
            <EnumStatusIndicator
              value={distributionPointWorkerStatus(point.worker)}
              metadata={DISTRIBUTION_POINT_WORKER_STATUSES}
              showIndicator
            />
          }
        />
        <KeyValueRow label="Enabled" value={<BooleanIndicator value={point.enabled} />} />
        <KeyValueRow
          label="Version"
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
  return <TokenList values={cidrs} />;
}
function PackageStateTable({ packages }: { packages: MunkiPackageState[] }) {
  const nonCurrentPackages = useMemo(
    () => packages.filter((pkg) => pkg.status !== "current"),
    [packages],
  );
  return (
    <DataTableStatic
      heading="Package Sync"
      columns={packageStateColumns}
      data={nonCurrentPackages}
      empty={
        <Empty className="min-h-40 border-0">
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <Smile />
            </EmptyMedia>
            <EmptyTitle>Everything is up to date</EmptyTitle>
            <EmptyDescription>
              Nothing is waiting to sync, and there are no errors to fix.
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      }
    />
  );
}

const packageStateColumns: DataTableColumnDef<MunkiPackageState>[] = [
  {
    id: "package",
    header: () => "Package",
    cell: ({ row }) => (
      <div className="flex items-center gap-2">
        <SoftwareArtwork src={row.original.software_icon_url} />
        <TextLink
          to="/munki/packages/$id"
          params={{ id: String(row.original.package_id) }}
          className="min-w-0 truncate font-medium"
          title={`${row.original.name} ${row.original.version}`}
        >
          {row.original.name} {row.original.version}
        </TextLink>
      </div>
    ),
  },
  {
    id: "status",
    header: () => "Status",
    cell: ({ row }) => packageStateLabel(row.original.status),
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
