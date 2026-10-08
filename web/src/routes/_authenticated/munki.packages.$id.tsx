import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { munkiPackageQueryOptions } from "@features/munki/packages/queries";
import { loadResource } from "@lib/resource-loader";
import { idParams } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/munki/packages/$id")({
  params: idParams,
  staticData: {
    breadcrumb: resourceName(
      munkiPackageQueryOptions,
      (pkg) => `${pkg.software.name} ${pkg.version}`,
    ),
  },
  loader: (ctx) => loadResource(ctx, munkiPackageQueryOptions(ctx.params.id)),
});
