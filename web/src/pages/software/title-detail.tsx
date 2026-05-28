import { Link, useParams } from "@tanstack/react-router";
import { Loader2, Package } from "lucide-react";
import type { ReactNode } from "react";

import { PageShell } from "@/components/layout/page-layout";
import { SoftwareIcon } from "@/components/software/software-icon";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useSoftwareTitle, type SoftwareTitle, type SoftwareVersion } from "@/hooks/use-software";
import { softwareSourceLabel } from "@/lib/software-source-labels";
import { formatRelative } from "@/lib/utils";

export function SoftwareTitleDetailPage() {
  const { softwareId } = useParams({ from: "/_authenticated/software/titles/$softwareId" });
  const query = useSoftwareTitle(Number(softwareId));
  const title = query.data;

  if (query.error) {
    return (
      <PageShell>
        <Alert variant="destructive">
          <AlertTitle>Failed to Load Software Title</AlertTitle>
          <AlertDescription>{query.error.message}</AlertDescription>
          <Button variant="outline" size="sm" onClick={() => void query.refetch()} className="mt-2 w-fit">
            Retry
          </Button>
        </Alert>
      </PageShell>
    );
  }

  if (query.isLoading || !title) {
    if (query.isLoading) {
      return (
        <PageShell className="text-muted-foreground flex-row items-center gap-2 text-sm">
          <Loader2 className="size-4 animate-spin" /> Loading...
        </PageShell>
      );
    }
    return (
      <PageShell>
        <Empty>
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <Package />
            </EmptyMedia>
            <EmptyTitle>Software Title Not Found</EmptyTitle>
            <EmptyDescription>This title is no longer available.</EmptyDescription>
          </EmptyHeader>
        </Empty>
      </PageShell>
    );
  }

  return (
    <PageShell className="gap-6">
      <SoftwareHeader title={title} />
      <SoftwareInfoCard title={title} />
      <SoftwareVersionsCard title={title} />
    </PageShell>
  );
}

function SoftwareHeader({ title }: { title: SoftwareTitle }) {
  const displayName = title.display_name || title.name;
  const typeLabel = softwareSourceLabel(title.source, title.extension_for);

  return (
    <div className="flex flex-wrap items-start justify-between gap-4">
      <div className="flex min-w-0 items-center gap-4">
        <SoftwareIcon source={title.source} size="lg" />
        <div className="flex min-w-0 flex-col gap-1">
          <div className="flex flex-wrap items-center gap-2">
            <h1 className="text-foreground truncate text-xl font-semibold" title={displayName}>
              {displayName}
            </h1>
            <Badge variant="secondary" className="font-normal">
              {typeLabel}
            </Badge>
          </div>
          <p className="text-muted-foreground text-xs">
            {title.counts_updated_at ? (
              <span title={new Date(title.counts_updated_at).toLocaleString()}>
                Counts updated {formatRelative(title.counts_updated_at)}
              </span>
            ) : (
              "Counts not yet computed"
            )}
          </p>
        </div>
      </div>
      <Button asChild variant="outline" size="sm">
        <Link to="/hosts" search={{ software_title_id: title.id.toString() }}>
          View hosts
        </Link>
      </Button>
    </div>
  );
}

interface Tile {
  label: string;
  value: ReactNode;
}

function SoftwareInfoCard({ title }: { title: SoftwareTitle }) {
  const tiles: Tile[] = [];

  if (title.bundle_identifier) {
    tiles.push({
      label: "Bundle Identifier",
      value: title.bundle_identifier,
    });
  }
  if (title.browser) {
    tiles.push({ label: "Browser", value: title.browser });
  }
  if (title.extension_for) {
    tiles.push({ label: "Extension for", value: title.extension_for });
  }
  tiles.push({ label: "Hosts", value: <span className="tabular-nums">{title.hosts_count}</span> });
  tiles.push({ label: "Versions", value: <span className="tabular-nums">{title.versions_count}</span> });

  tiles.sort((a, b) => a.label.localeCompare(b.label));

  return (
    <Card>
      <CardContent className="grid grid-cols-[repeat(auto-fit,minmax(170px,1fr))] gap-x-8 gap-y-5">
        {tiles.map((t) => (
          <div key={t.label} className="flex min-w-0 flex-col gap-1">
            <dt className="text-muted-foreground text-xs font-semibold">{t.label}</dt>
            <dd className="text-foreground truncate text-sm">{t.value}</dd>
          </div>
        ))}
      </CardContent>
    </Card>
  );
}

function SoftwareVersionsCard({ title }: { title: SoftwareTitle }) {
  const versions = title.versions ?? [];

  return (
    <Card>
      <CardHeader>
        <CardTitle>Versions</CardTitle>
      </CardHeader>
      <CardContent>
        {versions.length === 0 ? (
          <div className="bg-muted/30 rounded-md border border-dashed px-4 py-6 text-sm">
            <p className="font-medium">No Versions Discovered</p>
            <p className="text-muted-foreground">Hosts have not reported a concrete version.</p>
          </div>
        ) : (
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Version</TableHead>
                  <TableHead className="text-right">Hosts</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {versions.map((v) => (
                  <VersionRow key={v.id} version={v} />
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function VersionRow({ version }: { version: SoftwareVersion }) {
  return (
    <TableRow>
      <TableCell className="font-medium">{version.version || "-"}</TableCell>
      <TableCell className="text-right tabular-nums">
        <Link
          to="/hosts"
          search={{ software_id: version.id.toString() }}
          className="hover:text-primary hover:underline"
        >
          {version.hosts_count}
        </Link>
      </TableCell>
    </TableRow>
  );
}
