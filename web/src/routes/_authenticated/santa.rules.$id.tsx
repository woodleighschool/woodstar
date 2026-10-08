import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { santaRuleQueryOptions } from "@features/santa/rules/queries";
import { parseRouteID } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/santa/rules/$id")({
  staticData: { breadcrumb: resourceName(santaRuleQueryOptions, (rule) => rule.name) },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(santaRuleQueryOptions(parseRouteID(params.id)));
  },
});
