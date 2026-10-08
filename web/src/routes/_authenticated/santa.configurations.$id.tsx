import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { santaConfigurationQueryOptions } from "@features/santa/configurations/queries";
import { loadResource } from "@lib/resource-loader";
import { idParams } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/santa/configurations/$id")({
  params: idParams,
  staticData: {
    breadcrumb: resourceName(santaConfigurationQueryOptions, (configuration) => configuration.name),
  },
  loader: (ctx) => loadResource(ctx, santaConfigurationQueryOptions(ctx.params.id)),
});
