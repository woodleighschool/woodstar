import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { munkiPackageQueryOptions } from "@features/munki/packages/queries";
import { parseRouteID } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/munki/packages/$id")({
  staticData: {
    breadcrumb: resourceName(
      munkiPackageQueryOptions,
      (pkg) => `${pkg.software.name} ${pkg.version}`,
    ),
  },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(munkiPackageQueryOptions(parseRouteID(params.id)));
  },
});
