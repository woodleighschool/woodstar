import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { policyQueryOptions } from "@features/osquery/policies/queries";
import { loadResource } from "@lib/resource-loader";
import { idParams } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/osquery/policies/$id")({
  params: idParams,
  staticData: { breadcrumb: resourceName(policyQueryOptions, (policy) => policy.name) },
  loader: (ctx) => loadResource(ctx, policyQueryOptions(ctx.params.id)),
});
