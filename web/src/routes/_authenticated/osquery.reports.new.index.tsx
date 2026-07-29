import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { ReportCreatePage } from "@features/osquery/reports/create";

const searchSchema = z.object({
  tab: z.literal("targets").optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/osquery/reports/new/")({
  staticData: { breadcrumb: "Create" },
  validateSearch: searchSchema,
  component: ReportCreatePage,
});
