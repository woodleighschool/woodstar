import { createFileRoute } from "@tanstack/react-router";

import { checkQueryOptions } from "@/lib/queries/osquery";
import { parseRouteID } from "@/lib/route-params";

export const Route = createFileRoute("/_authenticated/osquery/checks/$checkId")({
  loader: async ({ context, params }) => {
    const check = await context.queryClient.ensureQueryData(
      checkQueryOptions(parseRouteID(params.checkId)),
    );
    return { breadcrumb: check.name };
  },
});
