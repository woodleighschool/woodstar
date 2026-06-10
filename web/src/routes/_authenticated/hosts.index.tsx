import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { tableSearchSchema } from "@/lib/pagination";
import { HostListPage } from "@/pages/hosts/list";

const searchSchema = z.object({
  ...tableSearchSchema.shape,
  status: z.string().optional(),
  label_id: z.string().optional(),
  software_title_id: z.coerce.number().int().positive().optional(),
  software_id: z.coerce.number().int().positive().optional(),
  check_id: z.coerce.number().int().positive().optional(),
  check_response: z.enum(["pass", "fail"]).optional(),
});

export const Route = createFileRoute("/_authenticated/hosts/")({
  validateSearch: (search) => searchSchema.parse(search),
  component: HostListPage,
});
