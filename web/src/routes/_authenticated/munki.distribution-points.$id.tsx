import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { munkiDistributionPointQueryOptions } from "@features/munki/distribution-points/queries";
import { loadResource } from "@lib/resource-loader";
import { idParams } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/munki/distribution-points/$id")({
  params: idParams,
  staticData: {
    breadcrumb: resourceName(
      munkiDistributionPointQueryOptions,
      (distributionPoint) => distributionPoint.name,
    ),
  },
  loader: (ctx) => loadResource(ctx, munkiDistributionPointQueryOptions(ctx.params.id)),
});
