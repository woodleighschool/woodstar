import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { santaRuleQueryOptions } from "@/lib/queries/santa";
import { parseRouteID } from "@/lib/route-params";
import { RuleEditPage } from "@/pages/santa/rules/edit";

export const Route = createFileRoute("/_authenticated/santa/rules/$ruleId")({
  staticData: { breadcrumb: RuleBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(santaRuleQueryOptions(parseRouteID(params.ruleId)));
  },
  component: RuleEditPage,
});

function RuleBreadcrumb(): string {
  const { ruleId } = useParams({ from: "/_authenticated/santa/rules/$ruleId" });
  const { data } = useQuery(santaRuleQueryOptions(parseRouteID(ruleId)));
  return data?.name ?? ruleId;
}
