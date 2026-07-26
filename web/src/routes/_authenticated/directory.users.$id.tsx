import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { userQueryOptions } from "@features/directory/users/queries";
import { parseRouteID } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/directory/users/$id")({
  staticData: { breadcrumb: UserBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(userQueryOptions(parseRouteID(params.id)));
  },
});

function UserBreadcrumb(): string {
  const { id } = useParams({ from: "/_authenticated/directory/users/$id" });
  const { data } = useQuery(userQueryOptions(parseRouteID(id)));
  return data?.name || data?.email || id;
}
