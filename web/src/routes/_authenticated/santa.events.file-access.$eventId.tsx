import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { santaFileAccessEventQueryOptions } from "@/lib/queries/santa";
import { parseRouteID } from "@/lib/route-params";
import { SantaFileAccessEventDetailPage } from "@/pages/santa/file-access-events/detail";

export const Route = createFileRoute("/_authenticated/santa/events/file-access/$eventId")({
  staticData: { breadcrumb: EventBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(
      santaFileAccessEventQueryOptions(parseRouteID(params.eventId)),
    );
  },
  component: SantaFileAccessEventDetailPage,
});

function EventBreadcrumb(): string {
  const { eventId } = useParams({
    from: "/_authenticated/santa/events/file-access/$eventId",
  });
  const { data } = useQuery(santaFileAccessEventQueryOptions(parseRouteID(eventId)));
  return data?.primary_process.file_name || "Event";
}
