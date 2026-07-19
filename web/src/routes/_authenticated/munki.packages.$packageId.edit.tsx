import { createFileRoute } from "@tanstack/react-router";

import { munkiPackageQueryOptions } from "@/lib/queries/munki";
import { parseRouteID } from "@/lib/route-params";
import { MunkiPackageEditPage } from "@/pages/munki/packages/edit";

export const Route = createFileRoute("/_authenticated/munki/packages/$packageId/edit")({
  loader: async ({ context, params }) => {
    const pkg = await context.queryClient.ensureQueryData(
      munkiPackageQueryOptions(parseRouteID(params.packageId)),
    );
    return { breadcrumb: `${pkg.software.name} ${pkg.version}` };
  },
  component: MunkiPackageEditPage,
});
