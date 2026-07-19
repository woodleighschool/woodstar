import { createFileRoute } from "@tanstack/react-router";

import { santaRuleQueryOptions } from "@/lib/queries/santa";
import { parseRouteID } from "@/lib/route-params";
import { RuleEditPage } from "@/pages/santa/rules/edit";

export const Route = createFileRoute("/_authenticated/santa/rules/$ruleId")({
  loader: async ({ context, params }) => {
    const rule = await context.queryClient.ensureQueryData(
      santaRuleQueryOptions(parseRouteID(params.ruleId)),
    );
    return { breadcrumb: rule.name };
  },
  component: RuleEditPage,
});
