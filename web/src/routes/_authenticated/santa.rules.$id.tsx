import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { santaRuleQueryOptions } from "@/lib/queries/santa";
import { parseRouteID } from "@/lib/route-params";
import { RuleEditPage } from "@/pages/santa/rules/edit";

export const Route = createFileRoute("/_authenticated/santa/rules/$id")({
  staticData: { breadcrumb: RuleBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(santaRuleQueryOptions(parseRouteID(params.id)));
  },
  component: RuleEditPage,
});

function RuleBreadcrumb(): string {
  const { id } = useParams({ from: "/_authenticated/santa/rules/$id" });
  const { data } = useQuery(santaRuleQueryOptions(parseRouteID(id)));
  return data?.name ?? id;
}
