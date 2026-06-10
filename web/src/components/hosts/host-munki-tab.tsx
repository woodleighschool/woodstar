import type { ReactNode } from "react";

import { EmptyPanel } from "@/components/empty-panel";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import type { HostDetail } from "@/hooks/use-hosts";
import { formatRelative } from "@/lib/utils";

type MunkiHostState = NonNullable<HostDetail["munki"]>;

interface Tile {
  label: string;
  value: ReactNode;
}

export function HostMunkiTab({ host }: { host: HostDetail }) {
  const munki = host.munki;
  if (!munki) return null;

  const items = munki.items ?? [];
  const problems = [
    ...problemRows("Errors", munki.errors),
    ...problemRows("Warnings", munki.warnings),
    ...problemRows("Problem Installs", munki.problem_installs),
  ];

  const tiles: Tile[] = [
    { label: "Version", value: munki.version || "-" },
    { label: "Manifest", value: munki.manifest_name || "-" },
    { label: "Status", value: <MunkiStatusBadge munki={munki} /> },
    { label: "Last Seen", value: formatRelative(munki.last_seen_at) },
    { label: "Last Run Started", value: formatRelative(munki.run_started_at) },
    { label: "Last Run Ended", value: formatRelative(munki.run_ended_at) },
  ];

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardContent>
          <dl className="grid grid-cols-[repeat(auto-fit,minmax(170px,1fr))] gap-x-8 gap-y-5">
            {tiles.map((tile) => (
              <div key={tile.label} className="flex min-w-0 flex-col gap-1">
                <dt className="text-muted-foreground text-xs font-semibold">{tile.label}</dt>
                <dd className="text-foreground truncate text-sm">{tile.value}</dd>
              </div>
            ))}
          </dl>
        </CardContent>
      </Card>

      {problems.length > 0 ? (
        <Card className="gap-4 py-4">
          <CardHeader>
            <CardTitle>Problems</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-col gap-3">
              {problems.map((problem) => (
                <div key={`${problem.kind}-${problem.value}`} className="flex min-w-0 items-start gap-3">
                  <Badge variant="secondary" className="shrink-0">
                    {problem.kind}
                  </Badge>
                  <span className="min-w-0 text-sm break-words">{problem.value}</span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      ) : null}

      <Card className="gap-4 py-4">
        <CardHeader>
          <CardTitle>Managed Items</CardTitle>
        </CardHeader>
        <CardContent>
          {items.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Version</TableHead>
                  <TableHead>Last Seen</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((item) => (
                  <TableRow key={item.name}>
                    <TableCell className="font-medium">{item.name}</TableCell>
                    <TableCell>{item.installed ? "Installed" : "Pending"}</TableCell>
                    <TableCell>{item.installed_version || "-"}</TableCell>
                    <TableCell>{formatRelative(item.last_seen_at)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : (
            <EmptyPanel>No managed items yet</EmptyPanel>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function MunkiStatusBadge({ munki }: { munki: MunkiHostState }) {
  if (munki.success === false) {
    return <Badge variant="destructive">Failed</Badge>;
  }
  if ((munki.errors?.length ?? 0) > 0 || (munki.problem_installs?.length ?? 0) > 0) {
    return <Badge variant="secondary">Problems</Badge>;
  }
  if ((munki.warnings?.length ?? 0) > 0) {
    return <Badge variant="secondary">Warnings</Badge>;
  }
  if (munki.success === true) {
    return <Badge variant="outline">OK</Badge>;
  }
  return <Badge variant="outline">Observed</Badge>;
}

function problemRows(kind: string, values: string[] | null | undefined) {
  return (values ?? []).map((value) => ({ kind, value }));
}
