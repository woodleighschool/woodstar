import { EmptyPanel } from "@/components/empty-panel";
import { KeyValueGrid, KeyValueItem } from "@/components/key-value";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { MunkiHostState } from "@/lib/api";
import { formatRelative } from "@/lib/utils";

export function HostMunkiTab({ munki }: { munki: MunkiHostState }) {
  const problems = [
    ...problemRows("Errors", munki.errors),
    ...problemRows("Warnings", munki.warnings),
    ...problemRows("Problem Installs", munki.problem_installs),
  ];

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardContent>
          <KeyValueGrid>
            <KeyValueItem label="Version" value={munki.version} />
            <KeyValueItem label="Manifest" value={munki.manifest_name} />
            <KeyValueItem label="Status" value={<MunkiStatusBadge munki={munki} />} />
            <KeyValueItem label="Last Run Started" value={formatRelative(munki.run_started_at)} />
            <KeyValueItem label="Last Run Ended" value={formatRelative(munki.run_ended_at)} />
          </KeyValueGrid>
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
                <div
                  key={`${problem.kind}-${problem.value}`}
                  className="flex min-w-0 items-start gap-3"
                >
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
          {munki.items.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Version</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {munki.items.map((item) => (
                  <TableRow key={item.name}>
                    <TableCell className="font-medium">{item.name}</TableCell>
                    <TableCell>{item.installed ? "Installed" : "Not Installed"}</TableCell>
                    <TableCell>{item.installed_version}</TableCell>
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
  if (munki.errors.length > 0) {
    return <Badge variant="destructive">Failed</Badge>;
  }
  if (munki.problem_installs.length > 0) {
    return <Badge variant="secondary">Problems</Badge>;
  }
  if (munki.warnings.length > 0) {
    return <Badge variant="secondary">Warnings</Badge>;
  }
  return <Badge variant="outline">OK</Badge>;
}

function problemRows(kind: string, values: string[]) {
  return values.map((value) => ({ kind, value }));
}
