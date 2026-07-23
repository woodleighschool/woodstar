import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { labelQueryOptions } from "@/lib/queries/labels";
import { parseRouteID } from "@/lib/route-params";
import { LabelEditPage } from "@/pages/labels/edit";

export const Route = createFileRoute("/_authenticated/labels/$id/edit")({
  staticData: { breadcrumb: LabelBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(labelQueryOptions(parseRouteID(params.id)));
  },
  component: LabelEditPage,
});

function LabelBreadcrumb(): string {
  const { id } = useParams({ from: "/_authenticated/labels/$id/edit" });
  const { data } = useQuery(labelQueryOptions(parseRouteID(id)));
  return data?.name ?? id;
}
