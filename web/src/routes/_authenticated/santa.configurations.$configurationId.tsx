import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { santaConfigurationQueryOptions } from "@/lib/queries/santa";
import { parseRouteID } from "@/lib/route-params";
import { ConfigurationEditPage } from "@/pages/santa/configurations/edit";

export const Route = createFileRoute("/_authenticated/santa/configurations/$configurationId")({
  staticData: { breadcrumb: ConfigurationBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(
      santaConfigurationQueryOptions(parseRouteID(params.configurationId)),
    );
  },
  component: ConfigurationEditPage,
});

function ConfigurationBreadcrumb(): string {
  const { configurationId } = useParams({
    from: "/_authenticated/santa/configurations/$configurationId",
  });
  const { data } = useQuery(santaConfigurationQueryOptions(parseRouteID(configurationId)));
  return data?.name ?? configurationId;
}
