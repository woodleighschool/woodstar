import { createFileRoute, redirect } from "@tanstack/react-router";
import { z } from "zod";

import { requirePermission } from "@features/authn/guards";
import { ConfigurationCreatePage } from "@features/santa/configurations/create";

const searchSchema = z.object({
  tab: z.literal("targets").optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/santa/configurations/new")({
  staticData: { breadcrumb: "Create" },
  validateSearch: searchSchema,
  beforeLoad: ({ context }) =>
    requirePermission(
      context.queryClient,
      { resource: "santa.configurations", access: "edit" },
      () => {
        throw redirect({ to: "/santa/configurations" });
      },
    ),
  component: ConfigurationCreatePage,
});
