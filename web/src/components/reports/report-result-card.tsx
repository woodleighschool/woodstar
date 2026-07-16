import { Link } from "@tanstack/react-router";

import { resultValue } from "@/components/reports/query-results";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import type { OsqueryHostReport } from "@/lib/api";
import { formatRelative } from "@/lib/utils";
interface ReportResultCardProps {
  report: OsqueryHostReport;
}
export function ReportResultCard({ report }: ReportResultCardProps) {
  const values = reportResultValues(report.first_result);
  const subtitle = report.last_fetched
    ? `Last updated ${formatRelative(report.last_fetched)}`
    : "Collecting results";
  return (
    <Card>
      <CardHeader>
        <CardTitle className="min-w-0 truncate" title={report.name}>
          {report.name}
        </CardTitle>
        <CardDescription>{subtitle}</CardDescription>
        <CardAction>
          <Button
            size="sm"
            variant="outline"
            render={
              <Link
                to="/osquery/reports/$reportId"
                params={{ reportId: String(report.report_id) }}
              />
            }
            nativeButton={false}
          >
            View Report
          </Button>
        </CardAction>
      </CardHeader>
      <CardContent>
        {values.length > 0 ? (
          <ReportValueGrid values={values} />
        ) : (
          <p className="text-sm text-muted-foreground">
            {report.last_fetched
              ? "This report ran but returned no rows for this host."
              : "No results have been stored yet."}
          </p>
        )}
      </CardContent>
    </Card>
  );
}
function ReportValueGrid({ values }: { values: ReportResultValue[] }) {
  return (
    <dl className="grid grid-cols-[repeat(auto-fit,minmax(min(100%,14rem),1fr))] gap-x-8 gap-y-5">
      {values.map((item) => (
        <div key={item.key} className="flex min-w-0 flex-col gap-1">
          <dt className="truncate text-xs font-semibold text-muted-foreground" title={item.key}>
            {item.key}
          </dt>
          <dd className="truncate text-sm text-foreground tabular-nums" title={item.value}>
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
    .toSorted((a, b) => a.key.localeCompare(b.key));
}
