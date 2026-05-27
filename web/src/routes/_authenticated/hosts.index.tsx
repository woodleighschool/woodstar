import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { MAX_PAGE_SIZE } from "@/lib/pagination";
import { HostsListPage } from "@/pages/hosts/list";

const searchSchema = z.object({
  q: z.string().optional(),
  page_index: z.coerce.number().int().min(0).optional(),
  page_size: z.coerce.number().int().min(10).max(MAX_PAGE_SIZE).optional(),
  sort: z.string().optional(),
  status: z.string().optional(),
  label_id: z.string().optional(),
  software_title_id: z.string().optional(),
  software_id: z.string().optional(),
});

export const Route = createFileRoute("/_authenticated/hosts/")({
  validateSearch: (search) => searchSchema.parse(search),
  component: HostsListPage,
});
