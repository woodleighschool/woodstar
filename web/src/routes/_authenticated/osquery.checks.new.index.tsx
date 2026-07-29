import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { CheckCreatePage } from "@features/osquery/checks/create";

const searchSchema = z.object({
  tab: z.literal("targets").optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/osquery/checks/new/")({
  staticData: { breadcrumb: "Create" },
  validateSearch: searchSchema,
  component: CheckCreatePage,
});
