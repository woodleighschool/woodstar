import { KeyValueRow, KeyValueRows } from "@components/key-value";
import { Link } from "@components/link";
import { Button } from "@components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@components/ui/card";
import type { OsqueryHostReport } from "@lib/api";
import { formatRelative } from "@lib/utils";

import { resultValue } from "./query-results";
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
            render={<Link to="/osquery/reports/$id" params={{ id: String(report.report_id) }} />}
            nativeButton={false}
          >
            View Report
          </Button>
        </CardAction>
      </CardHeader>
      <CardContent>
        {values.length > 0 ? (
          <KeyValueRows className="-mx-4">
            {values.map((item) => (
              <KeyValueRow
                key={item.key}
                label={item.key}
                value={resultValue(item.value)}
                className="rounded-none px-4"
                valueClassName="tabular-nums"
              />
            ))}
          </KeyValueRows>
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
interface ReportResultValue {
  key: string;
  value: string;
}
function reportResultValues(row: Record<string, string> | undefined): ReportResultValue[] {
  return Object.entries(row ?? {})
    .map(([key, value]) => ({ key, value }))
    .toSorted((a, b) => a.key.localeCompare(b.key));
}
