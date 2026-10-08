import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { userQueryOptions } from "@features/directory/users/queries";
import { parseRouteID } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/directory/users/$id")({
  staticData: { breadcrumb: resourceName(userQueryOptions, (user) => user.name || user.email) },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(userQueryOptions(parseRouteID(params.id)));
  },
});
