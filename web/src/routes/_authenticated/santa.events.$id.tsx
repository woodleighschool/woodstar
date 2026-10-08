import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { SantaEventDetailPage } from "@features/santa/events/detail";
import { santaEventQueryOptions } from "@features/santa/events/queries";
import { parseRouteID } from "@lib/route-params";

const searchSchema = z.object({
  tab: z.enum(["signing-chain", "entitlements"]).optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/santa/events/$id")({
  validateSearch: searchSchema,
  staticData: {
    breadcrumb: resourceName(
      santaEventQueryOptions,
      (event) => event.executable.file_name || "Execution",
    ),
  },
  loader: async ({ context, params }) => {
    await context.queryClient.ensureQueryData(santaEventQueryOptions(parseRouteID(params.id)));
  },
  component: SantaEventDetailPage,
});
