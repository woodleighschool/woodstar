import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useParams } from "@tanstack/react-router";
import { z } from "zod";

import { SantaFileAccessEventDetailPage } from "@features/santa/events/file-access-detail";
import { santaFileAccessEventQueryOptions } from "@features/santa/events/queries";
import { parseRouteID } from "@lib/route-params";

const searchSchema = z.object({
  tab: z.literal("process-chain").optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/santa/events/file-access/$id")({
  validateSearch: searchSchema,
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
