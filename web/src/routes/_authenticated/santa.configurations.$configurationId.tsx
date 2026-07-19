import { createFileRoute } from "@tanstack/react-router";

import { santaConfigurationQueryOptions } from "@/lib/queries/santa";
import { parseRouteID } from "@/lib/route-params";
import { ConfigurationEditPage } from "@/pages/santa/configurations/edit";

export const Route = createFileRoute("/_authenticated/santa/configurations/$configurationId")({
  loader: async ({ context, params }) => {
    const configuration = await context.queryClient.ensureQueryData(
      santaConfigurationQueryOptions(parseRouteID(params.configurationId)),
    );
    return { breadcrumb: configuration.name };
  },
  component: ConfigurationEditPage,
});
