import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { tableSearchSchema } from "@/lib/pagination";
import { SoftwareListPage } from "@/pages/software/list";

const searchSchema = z.object({
  ...tableSearchSchema.shape,
  source: z
    .preprocess((value) => (typeof value === "string" ? value.split(",").filter(Boolean) : value), z.array(z.string()))
    .optional(),
});

export const Route = createFileRoute("/_authenticated/software/")({
  validateSearch: (search) => searchSchema.parse(search),
  component: SoftwareListPage,
});
