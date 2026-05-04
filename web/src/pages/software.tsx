import { Package } from "lucide-react";

import { ErrorState } from "@/components/feedback/error-state";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/data-table";
import { EmptyState } from "@/components/ui/empty-state";
import { PageHeader } from "@/components/ui/page-header";
import { Spinner } from "@/components/ui/spinner";
import { useSoftware } from "@/hooks/use-software";
import { softwareSourceLabel } from "@/lib/software-source-labels";

export function SoftwarePage() {
  const query = useSoftware();

  return (
    <div className="flex flex-col">
      <PageHeader title="Software" description="Apps and packages discovered across enrolled hosts." />

      <div className="p-6">
        <SoftwareTable query={query} />
      </div>
    </div>
  );
}

function SoftwareTable({ query }: { query: ReturnType<typeof useSoftware> }) {
  if (query.error) {
    return <ErrorState message={query.error.message} onRetry={() => query.refetch()} />;
  }

  if (query.isLoading) {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Spinner /> Loading…
      </div>
    );
  }

  const data = query.data ?? [];
  if (data.length === 0) {
    return (
      <EmptyState
        icon={Package}
        title="No software inventory yet"
        description="Hosts will report installed apps and packages on their next detail refresh."
      />
    );
  }

  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Version</TableHead>
            <TableHead>Source</TableHead>
            <TableHead className="text-right">Hosts</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.map((row) => (
            <TableRow key={row.id}>
              <TableCell className="font-medium">{row.name}</TableCell>
              <TableCell className="text-muted-foreground">{row.version || "-"}</TableCell>
              <TableCell className="text-muted-foreground" title={row.source}>
                {softwareSourceLabel(row.source)}
              </TableCell>
              <TableCell className="text-right tabular-nums">{row.host_count}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
