import { Link } from "@components/link";

import type { CheckResultStatus } from "./model";

export function CheckResultCountLink({
  checkId,
  count,
  status,
}: {
  checkId: number;
  count: number;
  status: CheckResultStatus;
}) {
  return (
    <Link
      to="/osquery/checks/$id"
      params={{ id: String(checkId) }}
      search={{ tab: "results", status: [status] }}
      className="w-fit"
    >
      {count} {count === 1 ? "host" : "hosts"}
    </Link>
  );
}
