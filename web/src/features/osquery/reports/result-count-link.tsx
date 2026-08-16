import { TextLink } from "@components/link";
import type { OsqueryReportSnapshot } from "@lib/api";
import { countLabel } from "@lib/utils";

export function ReportResultCountLink({
  reportId,
  count,
  status,
}: {
  reportId: number;
  count: number;
  status: OsqueryReportSnapshot["status"];
}) {
  return (
    <TextLink
      to="/osquery/reports/$id"
      params={{ id: String(reportId) }}
      search={{ tab: "results", status }}
      className="w-fit"
    >
      {countLabel(count, "host")}
    </TextLink>
  );
}
