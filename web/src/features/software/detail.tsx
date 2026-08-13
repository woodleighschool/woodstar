import { useParams } from "@tanstack/react-router";

import { TableSurface } from "@components/data-table/table-surface";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { TextLink } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryGate } from "@components/query-gate";
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
import { hostCountLabel, softwareSourceLabel } from "@features/software/software-source-labels";
import type { SoftwareTitle } from "@lib/api";

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
      <SoftwareSummary title={title} />
      <SoftwareVersions title={title} />
      <SoftwareSigningIdentities title={title} />
    </PageShell>
  );
}

function SoftwareHeader({ title }: { title: SoftwareTitle }) {
  return (
    <PageHeader title={title.name} icon={<SoftwareIcon {...softwareIconProps(title.source)} />} />
  );
}

function SoftwareSummary({ title }: { title: SoftwareTitle }) {
  return (
    <KeyValueSection title="Overview">
      <KeyValueRow label="Type" value={softwareSourceLabel(title.source, title.extension_for)} />
      {title.bundle_identifier ? (
        <KeyValueRow label="Bundle ID" value={title.bundle_identifier} />
      ) : null}
      <KeyValueRow
        label="Hosts"
        value={
          <TextLink to="/hosts" search={{ software_title_id: title.id }}>
            {hostCountLabel(title.hosts_count)}
          </TextLink>
        }
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
            <TableRow
              key={`${identity.team_identifier}:${identity.identifier}:${identity.authority}`}
            >
              <TableCell className="font-medium">{identity.identifier || "-"}</TableCell>
              <TableCell>{identity.team_identifier || "-"}</TableCell>
              <TableCell className="max-w-xl whitespace-normal">
                {identity.authority || "-"}
              </TableCell>
              <TableCell className="text-right tabular-nums">
                {hostCountLabel(identity.hosts_count)}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableSurface>
  );
}

function SoftwareVersions({ title }: { title: SoftwareTitle }) {
  const versions = title.versions.items;

  return (
    <TableSurface
      heading="Versions"
      empty={versions.length ? undefined : <PanelEmptyState>No versions yet</PanelEmptyState>}
    >
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Version</TableHead>
            <TableHead className="text-right">Hosts</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {versions.map((version) => (
            <TableRow key={version.id}>
              <TableCell className="font-medium tabular-nums">
                {version.version || "Unknown version"}
              </TableCell>
              <TableCell className="text-right tabular-nums">
                <TextLink
                  to="/hosts"
                  search={{ software_title_id: title.id, software_id: version.id }}
                >
                  {hostCountLabel(version.hosts_count)}
                </TextLink>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableSurface>
  );
}
