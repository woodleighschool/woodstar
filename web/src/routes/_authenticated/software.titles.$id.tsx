import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { SoftwareDetailPage } from "@features/software/detail";
import { softwareTitleQueryOptions } from "@features/software/queries";
import { loadResource } from "@lib/resource-loader";
import { idParams } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/software/titles/$id")({
  params: idParams,
  staticData: { breadcrumb: resourceName(softwareTitleQueryOptions, (title) => title.name) },
  loader: (ctx) => loadResource(ctx, softwareTitleQueryOptions(ctx.params.id)),
  component: SoftwareDetailPage,
});
