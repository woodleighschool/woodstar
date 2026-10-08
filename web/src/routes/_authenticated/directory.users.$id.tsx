import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { userQueryOptions } from "@features/directory/users/queries";
import { loadResource } from "@lib/resource-loader";
import { idParams } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/directory/users/$id")({
  params: idParams,
  staticData: { breadcrumb: resourceName(userQueryOptions, (user) => user.name || user.email) },
  loader: (ctx) => loadResource(ctx, userQueryOptions(ctx.params.id)),
});
