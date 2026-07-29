import { createFileRoute, redirect } from "@tanstack/react-router";
import { z } from "zod";

import { requireAdmin } from "@features/auth/guards";
import { ReportEditPage } from "@features/osquery/reports/edit";

const searchSchema = z.object({
  tab: z.literal("targets").optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/osquery/reports/$id/edit")({
  staticData: { breadcrumb: "Edit" },
  validateSearch: searchSchema,
  beforeLoad: ({ context, params }) =>
    requireAdmin(context.currentUser, () => {
      throw redirect({
        to: "/osquery/reports/$id",
        params: { id: params.id },
      });
    }),
  component: ReportEditPage,
});
