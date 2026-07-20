import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { labelQueryOptions } from "@/lib/queries/labels";
import { parseRouteID } from "@/lib/route-params";
import { LabelEditPage } from "@/pages/labels/edit";

export const Route = createFileRoute("/_authenticated/labels/$labelId/edit")({
  staticData: { breadcrumb: LabelBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(labelQueryOptions(parseRouteID(params.labelId)));
  },
  component: LabelEditPage,
});

function LabelBreadcrumb(): string {
  const { labelId } = useParams({ from: "/_authenticated/labels/$labelId/edit" });
  const { data } = useQuery(labelQueryOptions(parseRouteID(labelId)));
  return data?.name ?? labelId;
}
