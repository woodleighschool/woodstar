import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { munkiSoftwareQueryOptions } from "@features/munki/software/queries";
import { parseRouteID } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/munki/software/$id")({
  staticData: { breadcrumb: resourceName(munkiSoftwareQueryOptions, (software) => software.name) },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(munkiSoftwareQueryOptions(parseRouteID(params.id)));
  },
});
