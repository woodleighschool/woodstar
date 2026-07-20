import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { userQueryOptions } from "@/lib/queries/users";
import { parseRouteID } from "@/lib/route-params";
import { UserEditPage } from "@/pages/users/edit";

export const Route = createFileRoute("/_authenticated/directory/users/$userId/edit")({
  staticData: { breadcrumb: UserBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(userQueryOptions(parseRouteID(params.userId)));
  },
  component: UserEditPage,
});

function UserBreadcrumb(): string {
  const { userId } = useParams({ from: "/_authenticated/directory/users/$userId/edit" });
  const { data } = useQuery(userQueryOptions(parseRouteID(userId)));
  return data?.name || data?.email || userId;
}
