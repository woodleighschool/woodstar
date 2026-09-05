import { useNavigate, useParams } from "@tanstack/react-router";
import { AppWindow, Pencil, Trash2 } from "lucide-react";
import { useMemo, useState } from "react";

import { DataTableStatic } from "@components/data-table/data-table-static";
import type { DataTableColumnDef } from "@components/data-table/types";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link, TextLink } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryGate } from "@components/query-gate";
import { TargetBadge, TargetDetails } from "@components/targeting/target-details";
import { Badge } from "@components/ui/badge";
import { Button } from "@components/ui/button";
import { useCan } from "@features/authz/access";
import { useLabelNameMap } from "@features/labels/components/label-ref-list";
import { SoftwareArtwork } from "@features/software/software-icon";
import type { MunkiInclude, MunkiPackage, MunkiSoftwareDetail } from "@lib/api";
import { parseRouteID } from "@lib/route-params";
import { countLabel, formatRelative } from "@lib/utils";

import { MUNKI_SOFTWARE_ACTIONS } from "./actions";
import { MunkiSoftwareDeleteDialog } from "./delete-dialog";
import { useMunkiSoftwareDetail } from "./queries";

const packageColumns: DataTableColumnDef<MunkiPackage>[] = [
  {
    id: "version",
    accessorKey: "version",
    header: () => "Version",
    cell: ({ row }) => (
      <div className="flex min-w-0 items-center gap-2">
        <SoftwareArtwork src={row.original.software.icon_url} />
        <TextLink
          to="/munki/packages/$id"
          params={{ id: String(row.original.id) }}
          className="min-w-0 truncate font-medium"
          title={row.original.version}
        >
          {row.original.version}
        </TextLink>
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
  const canEdit = useCan({ resource: "munki.software", access: "edit" });
  const softwareID = parseRouteID(params.id);
  const query = useMunkiSoftwareDetail(softwareID);
  const [deleteOpen, setDeleteOpen] = useState(false);

  if (softwareID === null) {
    return (
      <QueryGate
        title="Failed to Load Software"
        error={{ message: "Software route is invalid." }}
      />
    );
  }
  if (query.error || !query.data) {
    return (
      <QueryGate
        title="Failed to Load Software"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }

  const software = query.data;
  return (
    <PageShell className="gap-6">
      <PageHeader
        title="Software Details"
        icon={
          <SoftwareArtwork
            src={software.icon_url}
            fallbackIcon={AppWindow}
            size="md"
            loading="eager"
          />
        }
        actions={
          canEdit ? (
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
        <KeyValueRow
          label="Packages"
          value={countLabel(software.packages.length, "package", "packages")}
        />
      </KeyValueSection>

      <MunkiSoftwareTargets software={software} />

      <DataTableStatic
        heading="Packages"
        columns={packageColumns}
        data={software.packages}
        empty={<PanelEmptyState>No Packages Yet</PanelEmptyState>}
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
          <div className="flex flex-wrap gap-1.5">
            {software.targets.include.map((target) => {
              const label = labelsByID.get(target.label_id) ?? `Label ${target.label_id}`;
              return (
                <TargetBadge
                  key={target.label_id}
                  labelID={target.label_id}
                  label={label}
                  details={[
                    { label: "Package", value: munkiPackageLabel(target.package, packagesByID) },
                    {
                      label: "Actions",
                      value: (
                        <div className="flex flex-wrap justify-end gap-1">
                          {target.actions.map((action) => (
                            <Badge key={action} variant="secondary" className="font-normal">
                              {MUNKI_SOFTWARE_ACTIONS[action].name}
                            </Badge>
                          ))}
                        </div>
                      ),
                    },
                  ]}
                />
              );
            })}
          </div>
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
