import { createFileRoute, redirect } from "@tanstack/react-router";
import { z } from "zod";

import { requirePermission } from "@features/authn/guards";
import { ConfigurationEditPage } from "@features/santa/configurations/edit";

const searchSchema = z.object({
  tab: z.literal("targets").optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/santa/configurations/$id/edit")({
  staticData: { breadcrumb: "Edit" },
  validateSearch: searchSchema,
  beforeLoad: ({ context, params }) =>
    requirePermission(
      context.queryClient,
      { resource: "santa.configurations", access: "edit" },
      () => {
        throw redirect({
          to: "/santa/configurations/$id",
          params: { id: params.id },
        });
      },
    ),
  component: ConfigurationEditPage,
});
