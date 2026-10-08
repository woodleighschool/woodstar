import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { labelQueryOptions } from "@features/labels/queries";
import { loadResource } from "@lib/resource-loader";
import { idParams } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/labels/$id")({
  params: idParams,
  staticData: { breadcrumb: resourceName(labelQueryOptions, (label) => label.name) },
  loader: (ctx) => loadResource(ctx, labelQueryOptions(ctx.params.id)),
});
