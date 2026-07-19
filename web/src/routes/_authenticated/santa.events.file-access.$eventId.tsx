import { createFileRoute } from "@tanstack/react-router";

import { santaFileAccessEventQueryOptions } from "@/lib/queries/santa";
import { parseRouteID } from "@/lib/route-params";
import { SantaFileAccessEventDetailPage } from "@/pages/santa/file-access-events/detail";

export const Route = createFileRoute("/_authenticated/santa/events/file-access/$eventId")({
  loader: async ({ context, params }) => {
    const event = await context.queryClient.ensureQueryData(
      santaFileAccessEventQueryOptions(parseRouteID(params.eventId)),
    );
    return { breadcrumb: event.primary_process.file_name || "Event" };
  },
  component: SantaFileAccessEventDetailPage,
});
