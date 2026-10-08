import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { SantaEventDetailPage } from "@features/santa/events/detail";
import { santaEventQueryOptions } from "@features/santa/events/queries";
import { loadResource } from "@lib/resource-loader";
import { idParams } from "@lib/route-params";

const searchSchema = z.object({
  tab: z.enum(["signing-chain", "entitlements"]).optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/santa/events/$id")({
  params: idParams,
  validateSearch: searchSchema,
  staticData: {
    breadcrumb: resourceName(
      santaEventQueryOptions,
      (event) => event.executable.file_name || "Execution",
    ),
  },
  loader: (ctx) => loadResource(ctx, santaEventQueryOptions(ctx.params.id)),
  component: SantaEventDetailPage,
});
