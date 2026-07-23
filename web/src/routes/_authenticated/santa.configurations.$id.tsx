import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { santaConfigurationQueryOptions } from "@/lib/queries/santa";
import { parseRouteID } from "@/lib/route-params";
import { ConfigurationEditPage } from "@/pages/santa/configurations/edit";

export const Route = createFileRoute("/_authenticated/santa/configurations/$id")({
  staticData: { breadcrumb: ConfigurationBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(
      santaConfigurationQueryOptions(parseRouteID(params.id)),
    );
  },
  component: ConfigurationEditPage,
});

function ConfigurationBreadcrumb(): string {
  const { id } = useParams({
    from: "/_authenticated/santa/configurations/$id",
  });
  const { data } = useQuery(santaConfigurationQueryOptions(parseRouteID(id)));
  return data?.name ?? id;
}
