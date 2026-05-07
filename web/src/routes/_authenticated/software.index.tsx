import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { SoftwarePage } from "@/pages/software/list";

const searchSchema = z.object({
  q: z.string().optional(),
  source: z
    .preprocess((value) => (typeof value === "string" ? value.split(",").filter(Boolean) : value), z.array(z.string()))
    .optional(),
  page: z.coerce.number().int().min(1).optional(),
  per_page: z.coerce.number().int().min(10).max(200).optional(),
  order_key: z.string().optional(),
  order_direction: z.enum(["asc", "desc"]).optional(),
});

export const Route = createFileRoute("/_authenticated/software/")({
  validateSearch: (search) => searchSchema.parse(search),
  component: SoftwarePage,
});
