import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { labelQueryOptions } from "@features/labels/queries";
import { parseRouteID } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/labels/$id")({
  staticData: { breadcrumb: LabelBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(labelQueryOptions(parseRouteID(params.id)));
  },
});

function LabelBreadcrumb(): string {
  const { id } = useParams({ from: "/_authenticated/labels/$id" });
  const { data } = useQuery(labelQueryOptions(parseRouteID(id)));
  return data?.name ?? id;
}
