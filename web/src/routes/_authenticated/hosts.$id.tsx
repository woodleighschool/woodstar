import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { HostDetailPage } from "@features/hosts/detail";
import { hostQueryOptions } from "@features/hosts/queries";
import { loadResource } from "@lib/resource-loader";
import { idParams } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/hosts/$id")({
  params: idParams,
  staticData: { breadcrumb: resourceName(hostQueryOptions, (host) => host.display_name) },
  loader: (ctx) => loadResource(ctx, hostQueryOptions(ctx.params.id)),
  component: HostDetailPage,
});
