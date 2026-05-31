import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { MAX_PAGE_SIZE } from "@/lib/pagination";
import { MunkiReleasesPage } from "@/pages/munki/list";

const searchSchema = z.object({
  page_index: z.coerce.number().int().min(0).optional(),
  page_size: z.coerce.number().int().min(10).max(MAX_PAGE_SIZE).optional(),
  sort: z.string().optional(),
});

export const Route = createFileRoute("/_authenticated/munki/releases/")({
  validateSearch: (search) => searchSchema.parse(search),
  component: MunkiReleasesPage,
});
