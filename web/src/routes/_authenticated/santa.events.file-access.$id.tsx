import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";

import { SantaFileAccessEventDetailPage } from "@features/santa/events/file-access-detail";
import { santaFileAccessEventQueryOptions } from "@features/santa/events/queries";
import { parseRouteID } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/santa/events/file-access/$id")({
  staticData: { breadcrumb: EventBreadcrumb },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(
      santaFileAccessEventQueryOptions(parseRouteID(params.id)),
    );
  },
  component: SantaFileAccessEventDetailPage,
});

function EventBreadcrumb(): string {
  const { id } = useParams({
    from: "/_authenticated/santa/events/file-access/$id",
  });
  const { data } = useQuery(santaFileAccessEventQueryOptions(parseRouteID(id)));
  return data?.primary_process.file_name || "Event";
}
