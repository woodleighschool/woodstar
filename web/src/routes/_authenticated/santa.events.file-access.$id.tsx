import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { SantaFileAccessEventDetailPage } from "@features/santa/events/file-access-detail";
import { santaFileAccessEventQueryOptions } from "@features/santa/events/queries";
import { loadResource } from "@lib/resource-loader";
import { idParams } from "@lib/route-params";

const searchSchema = z.object({
  tab: z.literal("process-chain").optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/santa/events/file-access/$id")({
  params: idParams,
  validateSearch: searchSchema,
  staticData: {
    breadcrumb: resourceName(
      santaFileAccessEventQueryOptions,
      (event) => event.primary_process.file_name || "Event",
    ),
  },
  loader: (ctx) => loadResource(ctx, santaFileAccessEventQueryOptions(ctx.params.id)),
  component: SantaFileAccessEventDetailPage,
});
