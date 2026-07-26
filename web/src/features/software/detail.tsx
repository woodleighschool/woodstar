import { useParams } from "@tanstack/react-router";

import { TableSurface } from "@components/data-table/table-surface";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryGate } from "@components/query-gate";
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
import { useSoftwareTitle } from "@features/software/queries";
import { SoftwareIcon, softwareIconProps } from "@features/software/software-icon";
import { softwareSourceLabel } from "@features/software/software-source-labels";
import type { SoftwareTitle, SoftwareVersion } from "@lib/api";

export function SoftwareDetailPage() {
  const { id: softwareId } = useParams({
    from: "/_authenticated/software/titles/$id",
  });
  const query = useSoftwareTitle(Number(softwareId), {
    refetchInterval: 30000,
  });
  const title = query.data;

  if (query.error || !title) {
    return (
      <QueryGate
        title="Failed to load software title"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }

  return (
    <PageShell className="gap-6">
      <SoftwareHeader title={title} />
      <SoftwareOverview title={title} />
      <SoftwareSigningIdentities title={title} />
      <SoftwareVersions title={title} />
    </PageShell>
  );
}

function SoftwareHeader({ title }: { title: SoftwareTitle }) {
  const typeLabel = softwareSourceLabel(title.source, title.extension_for);

  return (
    <PageHeader
      title={title.name}
      icon={<SoftwareIcon {...softwareIconProps(title.source)} />}
      context={
        <Badge variant="secondary" className="font-normal">
          {typeLabel}
        </Badge>
      }
      actions={
        <Button
          variant="outline"
          size="sm"
          render={<Link to="/hosts" search={{ software_title_id: title.id }} />}
          nativeButton={false}
        >
          View hosts
        </Button>
      }
    />
  );
}

function SoftwareOverview({ title }: { title: SoftwareTitle }) {
  return (
    <KeyValueSection title="Overview">
      {title.browser ? <KeyValueRow label="Browser" value={title.browser} /> : null}
      {title.bundle_identifier ? (
        <KeyValueRow label="Bundle Identifier" value={title.bundle_identifier} />
      ) : null}
      {title.extension_for ? (
        <KeyValueRow label="Extension for" value={title.extension_for} />
      ) : null}
      <KeyValueRow
        label="Hosts"
        value={
          <Link to="/hosts" search={{ software_title_id: title.id }}>
            <span className="tabular-nums">{title.hosts_count}</span>{" "}
            {title.hosts_count === 1 ? "host" : "hosts"}
          </Link>
        }
      />
      <KeyValueRow
        label="Versions"
        value={<span className="tabular-nums">{title.versions.count}</span>}
      />
    </KeyValueSection>
  );
}

function SoftwareSigningIdentities({ title }: { title: SoftwareTitle }) {
  const identities = title.signing_identities.items;
  if (identities.length === 0) return null;

  return (
    <TableSurface heading="Signing identities">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Identifier</TableHead>
            <TableHead>Team ID</TableHead>
            <TableHead>Authority</TableHead>
            <TableHead className="text-right">Hosts</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {identities.map((identity) => (
            <TableRow key={`${identity.team_identifier}:${identity.identifier}`}>
              <TableCell className="font-medium">{identity.identifier || "-"}</TableCell>
              <TableCell>{identity.team_identifier}</TableCell>
              <TableCell>{identity.authorities.join(", ") || "-"}</TableCell>
              <TableCell className="text-right tabular-nums">{identity.hosts_count}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableSurface>
  );
}

function SoftwareVersions({ title }: { title: SoftwareTitle }) {
  const versions = title.versions.items;

  if (versions.length === 0) {
    return (
      <section className="flex min-w-0 flex-col gap-3">
        <h2 className="text-base/snug font-medium">Versions</h2>
        <PanelEmptyState>No versions yet</PanelEmptyState>
      </section>
    );
  }

  return (
    <TableSurface heading="Versions">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Version</TableHead>
            <TableHead className="text-right">Hosts</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {versions.map((version) => (
            <VersionRow key={version.id} title={title} version={version} />
          ))}
        </TableBody>
      </Table>
    </TableSurface>
  );
}

function VersionRow({ title, version }: { title: SoftwareTitle; version: SoftwareVersion }) {
  return (
    <TableRow>
      <TableCell className="font-medium">{version.version || "-"}</TableCell>
      <TableCell className="text-right tabular-nums">
        <Link to="/hosts" search={{ software_title_id: title.id, software_id: version.id }}>
          {version.hosts_count}
        </Link>
      </TableCell>
    </TableRow>
  );
}
