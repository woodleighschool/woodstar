import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { santaEventQueryOptions } from "@/lib/queries/santa";
import { parseRouteID } from "@/lib/route-params";
import { SantaEventDetailPage } from "@/pages/santa/events/detail";

export const Route = createFileRoute("/_authenticated/santa/events/$eventId")({
  staticData: { breadcrumb: EventBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(santaEventQueryOptions(parseRouteID(params.eventId)));
  },
  component: SantaEventDetailPage,
});

function EventBreadcrumb(): string {
  const { eventId } = useParams({ from: "/_authenticated/santa/events/$eventId" });
  const { data } = useQuery(santaEventQueryOptions(parseRouteID(eventId)));
  return data?.executable.file_name || "Execution";
}
