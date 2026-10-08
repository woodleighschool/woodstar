import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { santaConfigurationQueryOptions } from "@features/santa/configurations/queries";
import { parseRouteID } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/santa/configurations/$id")({
  staticData: {
    breadcrumb: resourceName(santaConfigurationQueryOptions, (configuration) => configuration.name),
  },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(
      santaConfigurationQueryOptions(parseRouteID(params.id)),
    );
  },
});
