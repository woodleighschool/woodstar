import { createFileRoute } from "@tanstack/react-router";

import { labelQueryOptions } from "@/lib/queries/labels";
import { parseRouteID } from "@/lib/route-params";
import { LabelEditPage } from "@/pages/labels/edit";

export const Route = createFileRoute("/_authenticated/labels/$labelId/edit")({
  loader: async ({ context, params }) => {
    const label = await context.queryClient.ensureQueryData(
      labelQueryOptions(parseRouteID(params.labelId)),
    );
    return { breadcrumb: label.name };
  },
  component: LabelEditPage,
});
