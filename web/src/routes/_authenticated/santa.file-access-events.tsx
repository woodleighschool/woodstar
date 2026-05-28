/* eslint-disable @typescript-eslint/only-throw-error -- tanstack/react-router uses thrown redirect() as control-flow */
import { createFileRoute, redirect } from "@tanstack/react-router";
import { z } from "zod";

import { MAX_PAGE_SIZE } from "@/lib/pagination";

const searchSchema = z.object({
  q: z.string().optional(),
  decisions: z
    .preprocess((value) => (typeof value === "string" ? value.split(",").filter(Boolean) : value), z.array(z.string()))
    .optional(),
  page_index: z.coerce.number().int().min(0).optional(),
  page_size: z.coerce.number().int().min(10).max(MAX_PAGE_SIZE).optional(),
  sort: z.string().optional(),
});

export const Route = createFileRoute("/_authenticated/santa/file-access-events")({
  validateSearch: (search) => searchSchema.parse(search),
  beforeLoad: ({ search }) => {
    throw redirect({ to: "/santa/events", search: { ...search, event_type: "file_access" } });
  },
});
