import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { santaRuleQueryOptions } from "@features/santa/rules/queries";
import { loadResource } from "@lib/resource-loader";
import { idParams } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/santa/rules/$id")({
  params: idParams,
  staticData: { breadcrumb: resourceName(santaRuleQueryOptions, (rule) => rule.name) },
  loader: (ctx) => loadResource(ctx, santaRuleQueryOptions(ctx.params.id)),
});
