import { createFileRoute } from "@tanstack/react-router";

import { userQueryOptions } from "@/lib/queries/users";
import { parseRouteID } from "@/lib/route-params";
import { UserEditPage } from "@/pages/users/edit";

export const Route = createFileRoute("/_authenticated/directory/users/$userId/edit")({
  loader: async ({ context, params }) => {
    const user = await context.queryClient.ensureQueryData(
      userQueryOptions(parseRouteID(params.userId)),
    );
    return { breadcrumb: user.name || user.email || params.userId };
  },
  component: UserEditPage,
});
