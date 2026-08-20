import { useQuery } from "@tanstack/react-query";
import { createFileRoute, redirect, useParams } from "@tanstack/react-router";

import { santaConfigurationQueryOptions } from "@features/santa/configurations/queries";
import { parseRouteID } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/santa/configurations/$id")({
  staticData: { breadcrumb: ConfigurationBreadcrumb },
  loader: async ({ context, params }) => {
    const id = parseRouteID(params.id);
    if (id === null) throw redirect({ to: "/santa/configurations" });
    await context.queryClient.ensureQueryData(santaConfigurationQueryOptions(id));
  },
});

function ConfigurationBreadcrumb(): string {
  const { id } = useParams({
    from: "/_authenticated/santa/configurations/$id",
  });
  const { data } = useQuery(santaConfigurationQueryOptions(parseRouteID(id)));
  return data?.name ?? id;
}
