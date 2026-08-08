import { useQuery } from "@tanstack/react-query";
import { createFileRoute, redirect, useParams } from "@tanstack/react-router";
import type { ReactNode } from "react";

import { santaRuleQueryOptions } from "@features/santa/rules/queries";
import { loadResource } from "@lib/resource-loader";
import { idParams } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/santa/configurations/$id/rules/$ruleId")({
  params: {
    parse: ({ ruleId }) => {
      const parsed = idParams.parse({ id: ruleId });
      return parsed === false ? false : { ruleId: parsed.id };
    },
  },
  staticData: { breadcrumb: RuleBreadcrumb },
  loader: async (ctx) => {
    const options = santaRuleQueryOptions(ctx.params.ruleId);
    await loadResource(ctx, options);
    const rule = ctx.context.queryClient.getQueryData(options.queryKey);
    if (rule && rule.configuration_id !== ctx.params.id) {
      throw redirect({
        to: "/santa/configurations/$id/rules/$ruleId",
        params: { id: rule.configuration_id, ruleId: rule.id },
      });
    }
  },
});

function RuleBreadcrumb({ children }: { children: (name: string | undefined) => ReactNode }) {
  const { ruleId } = useParams({
    from: "/_authenticated/santa/configurations/$id/rules/$ruleId",
  });
  const { data } = useQuery(santaRuleQueryOptions(ruleId));
  return children(data?.name);
}
