import { useQuery } from "@tanstack/react-query";
import { createFileRoute, redirect, useParams } from "@tanstack/react-router";

import { santaRuleQueryOptions } from "@features/santa/rules/queries";
import { parseRouteID } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/santa/configurations/$id/rules/$ruleId")({
  staticData: { breadcrumb: RuleBreadcrumb },
  loader: async ({ context, params }) => {
    const ruleID = parseRouteID(params.ruleId);
    if (ruleID === null) {
      throw redirect({
        to: "/santa/configurations/$id/rules",
        params: { id: params.id },
      });
    }
    const rule = await context.queryClient.ensureQueryData(santaRuleQueryOptions(ruleID));
    if (rule.configuration_id !== parseRouteID(params.id)) {
      throw redirect({
        to: "/santa/configurations/$id/rules/$ruleId",
        params: { id: String(rule.configuration_id), ruleId: String(rule.id) },
      });
    }
  },
});

function RuleBreadcrumb(): string {
  const { ruleId } = useParams({
    from: "/_authenticated/santa/configurations/$id/rules/$ruleId",
  });
  const { data } = useQuery(santaRuleQueryOptions(parseRouteID(ruleId)));
  return data?.name ?? ruleId;
}
