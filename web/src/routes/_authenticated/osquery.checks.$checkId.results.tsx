import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { CheckResultsPage } from "@/pages/osquery/checks/results";

const searchSchema = z.object({
  response: z.enum(["pass", "fail"]).optional(),
});

export const Route = createFileRoute("/_authenticated/osquery/checks/$checkId/results")({
  staticData: { breadcrumb: "Results" },
  validateSearch: (search) => searchSchema.parse(search),
  component: CheckResultsPage,
});
