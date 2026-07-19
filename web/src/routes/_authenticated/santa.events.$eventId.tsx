import { createFileRoute } from "@tanstack/react-router";

import { santaEventQueryOptions } from "@/lib/queries/santa";
import { parseRouteID } from "@/lib/route-params";
import { SantaEventDetailPage } from "@/pages/santa/events/detail";

export const Route = createFileRoute("/_authenticated/santa/events/$eventId")({
  loader: async ({ context, params }) => {
    const event = await context.queryClient.ensureQueryData(
      santaEventQueryOptions(parseRouteID(params.eventId)),
    );
    return { breadcrumb: event.executable.file_name || "Execution" };
  },
  component: SantaEventDetailPage,
});
