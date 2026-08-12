import { TextLink } from "@components/link";

import type { PolicyResultStatus } from "./model";

export function PolicyResultCountLink({
  policyId,
  count,
  status,
}: {
  policyId: number;
  count: number;
  status: PolicyResultStatus;
}) {
  return (
    <TextLink
      to="/osquery/policies/$id"
      params={{ id: String(policyId) }}
      search={{ tab: "results", status: [status] }}
      className="w-fit"
    >
      {count} {count === 1 ? "host" : "hosts"}
    </TextLink>
  );
}
