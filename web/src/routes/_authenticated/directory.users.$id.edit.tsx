import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { userQueryOptions } from "@/lib/queries/users";
import { parseRouteID } from "@/lib/route-params";
import { UserEditPage } from "@/pages/users/edit";

export const Route = createFileRoute("/_authenticated/directory/users/$id/edit")({
  staticData: { breadcrumb: UserBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(userQueryOptions(parseRouteID(params.id)));
  },
  component: UserEditPage,
});

function UserBreadcrumb(): string {
  const { id } = useParams({ from: "/_authenticated/directory/users/$id/edit" });
  const { data } = useQuery(userQueryOptions(parseRouteID(id)));
  return data?.name || data?.email || id;
}
