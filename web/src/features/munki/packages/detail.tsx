import { useNavigate, useParams } from "@tanstack/react-router";
import { filesize } from "filesize";
import { Package as PackageIcon, Pencil, Trash2 } from "lucide-react";
import { useState } from "react";

import { BooleanIndicator } from "@components/boolean-indicator";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link, TextLink } from "@components/link";
import { QueryGate } from "@components/query-gate";
import { TokenList } from "@components/token-list";
import { Button } from "@components/ui/button";
import { useCan } from "@features/authz/access";
import { SoftwareArtwork } from "@features/software/software-icon";
import type { MunkiPackageReference } from "@lib/api";
import { parseRouteID } from "@lib/route-params";

import { MunkiPackageDeleteDialog } from "./delete-dialog";
import { useMunkiPackage } from "./queries";

export function MunkiPackageDetailPage() {
  const params = useParams({ strict: false });
  const navigate = useNavigate();
  const canEdit = useCan({ resource: "munki.packages", access: "edit" });
  const packageID = parseRouteID(params.id);
  const query = useMunkiPackage(packageID);
  const [deleteOpen, setDeleteOpen] = useState(false);

  if (packageID === null) {
    return (
      <QueryGate title="Failed to Load Package" error={{ message: "Package route is invalid." }} />
    );
  }
  if (query.error || !query.data) {
    return (
      <QueryGate
        title="Failed to Load Package"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }

  const pkg = query.data;
  return (
    <PageShell className="gap-6">
      <PageHeader
        title="Package Details"
        icon={
          <SoftwareArtwork
            src={pkg.software.icon_url}
            fallbackIcon={PackageIcon}
            size="md"
            loading="eager"
          />
        }
        actions={
          canEdit ? (
            <>
              <Button
                size="sm"
                render={<Link to="/munki/packages/$id/edit" params={{ id: String(pkg.id) }} />}
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
        <KeyValueRow
          label="Software"
          value={
            <TextLink
              to="/munki/software/$id"
              params={{ id: String(pkg.software.id) }}
              className="font-medium"
            >
              {pkg.software.name}
            </TextLink>
          }
        />
        <KeyValueRow label="Version" value={pkg.version} />
        <KeyValueRow label="Notes" value={pkg.notes} />
        <KeyValueRow label="Installer" value={pkg.installer_type} />
        <KeyValueRow
          label="Installer File"
          value={
            pkg.installer_file ? (
              <span className="flex flex-col gap-0.5">
                <span>{pkg.installer_file.filename}</span>
                <span className="text-muted-foreground">
                  {filesize(pkg.installer_file.size_bytes)}
                </span>
              </span>
            ) : (
              "-"
            )
          }
        />
        <KeyValueRow label="Minimum macOS" value={pkg.minimum_os_version || "Any"} />
        <KeyValueRow label="Maximum macOS" value={pkg.maximum_os_version || "Any"} />
        <KeyValueRow
          label="Architecture"
          value={<TokenList values={pkg.supported_architectures} empty="Any" />}
        />
      </KeyValueSection>

      <KeyValueSection title="Behaviour">
        <KeyValueRow label="On Demand" value={<BooleanIndicator value={pkg.on_demand} />} />
        <KeyValueRow label="Precache" value={<BooleanIndicator value={pkg.precache} />} />
        <KeyValueRow
          label="Unattended Install"
          value={<BooleanIndicator value={pkg.unattended_install} />}
        />
        <KeyValueRow
          label="Unattended Uninstall"
          value={<BooleanIndicator value={pkg.unattended_uninstall} />}
        />
        <KeyValueRow label="Uninstallable" value={<BooleanIndicator value={pkg.uninstallable} />} />
        <KeyValueRow label="Restart Action" value={pkg.restart_action || "None"} />
      </KeyValueSection>

      <KeyValueSection title="Relationships">
        <KeyValueRow label="Requires" value={<PackageReferences values={pkg.requires} />} />
        <KeyValueRow label="Updates" value={<PackageReferences values={pkg.update_for} />} />
      </KeyValueSection>

      <MunkiPackageDeleteDialog
        pkg={pkg}
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        onDeleted={() => void navigate({ to: "/munki/packages" })}
      />
    </PageShell>
  );
}

function PackageReferences({ values }: { values: MunkiPackageReference[] }) {
  if (values.length === 0) return "-";
  return (
    <div className="flex flex-col items-start gap-1">
      {values.map((reference) => (
        <TextLink
          key={`${reference.software_id}:${reference.package_id ?? "latest"}`}
          to={reference.package_id === undefined ? "/munki/software/$id" : "/munki/packages/$id"}
          params={{ id: String(reference.package_id ?? reference.software_id) }}
          className="font-medium"
        >
          {reference.software_name} {reference.package_version ?? "latest"}
        </TextLink>
      ))}
    </div>
  );
}
