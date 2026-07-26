import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { santaRuleQueryOptions } from "@features/santa/rules/queries";
import { parseRouteID } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/santa/rules/$id")({
  staticData: { breadcrumb: RuleBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(santaRuleQueryOptions(parseRouteID(params.id)));
  },
});

function RuleBreadcrumb(): string {
  const { id } = useParams({ from: "/_authenticated/santa/rules/$id" });
  const { data } = useQuery(santaRuleQueryOptions(parseRouteID(id)));
  return data?.name ?? id;
}
