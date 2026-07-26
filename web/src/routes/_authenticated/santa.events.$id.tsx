import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { SantaEventDetailPage } from "@features/santa/events/detail";
import { santaEventQueryOptions } from "@features/santa/events/queries";
import { parseRouteID } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/santa/events/$id")({
  staticData: { breadcrumb: EventBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(santaEventQueryOptions(parseRouteID(params.id)));
  },
  component: SantaEventDetailPage,
});

function EventBreadcrumb(): string {
  const { id } = useParams({ from: "/_authenticated/santa/events/$id" });
  const { data } = useQuery(santaEventQueryOptions(parseRouteID(id)));
  return data?.executable.file_name || "Execution";
}
