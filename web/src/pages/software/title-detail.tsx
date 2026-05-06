import { Link, useParams } from "@tanstack/react-router";
import { ChevronLeft, Package } from "lucide-react";
import type React from "react";

import { EmptyState } from "@/components/feedback/empty-state";
import { ErrorState } from "@/components/feedback/error-state";
import { Spinner } from "@/components/feedback/spinner";
import { PageHeader } from "@/components/layout/page-header";
import { Button } from "@/components/ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useSoftwareTitle, type SoftwareTitle, type SoftwareVersion } from "@/hooks/use-software";
import { softwareSourceLabel } from "@/lib/software-source-labels";

export function SoftwareTitleDetailPage() {
  const { softwareId } = useParams({ from: "/_authed/software/titles/$softwareId" });
  const query = useSoftwareTitle(softwareId);
  const title = query.data?.software_title;
  const displayName = title ? title.display_name || title.name : `Software ${softwareId}`;

  return (
    <div className="flex flex-col">
      <PageHeader
        title={displayName}
        description={title ? softwareSourceLabel(title.source, title.extension_for) : "Software title"}
        actions={
          <Button asChild variant="outline" size="sm">
            <Link to="/software" className="gap-1">
              <ChevronLeft className="size-4" /> Back to software
            </Link>
          </Button>
        }
      />

      <div className="flex flex-col gap-4 p-6">
        {query.error ? (
          <ErrorState message={query.error.message} onRetry={() => query.refetch()} />
        ) : query.isLoading ? (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Spinner /> Loading…
          </div>
        ) : title ? (
          <>
            <SoftwareTitleSummary title={title} />
            <VersionsTable title={title} />
          </>
        ) : (
          <EmptyState
            icon={Package}
            title="Software title not found"
            description="This title is no longer available."
          />
        )}
      </div>
    </div>
  );
}

function SoftwareTitleSummary({ title }: { title: SoftwareTitle }) {
  return (
    <div className="rounded-md border">
      <dl className="grid gap-px bg-border sm:grid-cols-3">
        <SummaryItem label="Hosts" value={title.hosts_count} />
        <SummaryItem label="Versions" value={title.versions_count} />
        <SummaryItem label="Type" value={softwareSourceLabel(title.source, title.extension_for)} />
      </dl>
      <div className="flex items-center justify-end border-t px-4 py-3">
        <Button asChild variant="outline" size="sm">
          <Link to="/hosts" search={{ software_title_id: title.id.toString() }}>
            View all hosts
          </Link>
        </Button>
      </div>
    </div>
  );
}

function SummaryItem({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="bg-background px-4 py-3">
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className="mt-1 text-sm font-medium">{value}</dd>
    </div>
  );
}

function VersionsTable({ title }: { title: SoftwareTitle }) {
  const versions = title.versions ?? [];

  if (versions.length === 0) {
    return (
      <div className="rounded-md border border-dashed bg-muted/30 px-4 py-6 text-sm">
        <p className="font-medium">No versions discovered</p>
        <p className="text-muted-foreground">Hosts have reported the title but no concrete version yet.</p>
      </div>
    );
  }

  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Version</TableHead>
            <TableHead className="text-right">Hosts</TableHead>
            <TableHead>Bundle identifier</TableHead>
            <TableHead className="w-28" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {versions.map((version) => (
            <VersionRow key={version.id} version={version} />
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function VersionRow({ version }: { version: SoftwareVersion }) {
  return (
    <TableRow>
      <TableCell className="font-mono text-xs">{version.version || "-"}</TableCell>
      <TableCell className="text-right tabular-nums">{version.hosts_count}</TableCell>
      <TableCell className="font-mono text-xs text-muted-foreground break-all">
        {version.bundle_identifier || "-"}
      </TableCell>
      <TableCell className="text-right">
        <Button asChild variant="ghost" size="sm">
          <Link to="/hosts" search={{ software_id: version.id.toString() }}>
            Hosts
          </Link>
        </Button>
      </TableCell>
    </TableRow>
  );
}
