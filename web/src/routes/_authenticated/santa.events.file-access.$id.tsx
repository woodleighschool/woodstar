import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { SantaFileAccessEventDetailPage } from "@features/santa/events/file-access-detail";
import { santaFileAccessEventQueryOptions } from "@features/santa/events/queries";
import { parseRouteID } from "@lib/route-params";

const searchSchema = z.object({
  tab: z.literal("process-chain").optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/santa/events/file-access/$id")({
  validateSearch: searchSchema,
  staticData: {
    breadcrumb: resourceName(
      santaFileAccessEventQueryOptions,
      (event) => event.primary_process.file_name || "Event",
    ),
  },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(
      santaFileAccessEventQueryOptions(parseRouteID(params.id)),
    );
  },
  component: SantaFileAccessEventDetailPage,
});
