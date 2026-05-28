import { Link } from "@tanstack/react-router";

import { Button } from "@/components/ui/button";
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import type { HostReport } from "@/hooks/use-hosts";
import { resultValue } from "@/lib/query-results";
import { formatRelative } from "@/lib/utils";

interface ReportResultCardProps {
  report: HostReport;
  hostParam: string;
}

export function ReportResultCard({ report, hostParam }: ReportResultCardProps) {
  const values = reportResultValues(report.first_result);
  const subtitle = report.last_fetched ? `Last updated ${formatRelative(report.last_fetched)}` : "Collecting results";

  return (
    <Card>
      <CardHeader>
        <CardTitle className="min-w-0 truncate" title={report.name}>
          <Link
            to="/hosts/$hostId/reports/$reportId"
            params={{ hostId: hostParam, reportId: String(report.report_id) }}
            className="hover:underline"
          >
            {report.name}
          </Link>
        </CardTitle>
        <CardDescription>{subtitle}</CardDescription>
        <CardAction>
          <Button asChild size="sm" variant="outline">
            <Link to="/osquery/reports/$reportId" params={{ reportId: String(report.report_id) }}>
              All hosts
            </Link>
          </Button>
        </CardAction>
      </CardHeader>
      <CardContent>
        {values.length > 0 ? (
          <KeyValueGrid values={values} />
        ) : (
          <p className="text-muted-foreground text-sm">
            {report.last_fetched
              ? "This report ran but returned no rows for this host."
              : "No results have been stored yet."}
          </p>
        )}
      </CardContent>
    </Card>
  );
}

function KeyValueGrid({ values }: { values: ReportResultValue[] }) {
  return (
    <dl className="grid grid-cols-[repeat(auto-fit,minmax(min(100%,14rem),1fr))] gap-x-8 gap-y-5">
      {values.map((item) => (
        <div key={item.key} className="flex min-w-0 flex-col gap-1">
          <dt className="text-muted-foreground truncate text-xs font-semibold" title={item.key}>
            {item.key}
          </dt>
          <dd className="text-foreground truncate text-sm tabular-nums" title={item.value}>
            {resultValue(item.value)}
          </dd>
        </div>
      ))}
    </dl>
  );
}

interface ReportResultValue {
  key: string;
  value: string;
}

function reportResultValues(row: Record<string, string> | undefined): ReportResultValue[] {
  return Object.entries(row ?? {})
    .map(([key, value]) => ({ key, value }))
    .sort((a, b) => a.key.localeCompare(b.key));
}
