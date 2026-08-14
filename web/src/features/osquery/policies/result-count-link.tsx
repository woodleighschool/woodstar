import { TextLink } from "@components/link";
import { countLabel } from "@lib/utils";

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
      {countLabel(count, "host")}
    </TextLink>
  );
}
