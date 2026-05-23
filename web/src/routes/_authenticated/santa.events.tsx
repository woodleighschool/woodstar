import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { SantaEventsPage } from "@/pages/santa/events";

const searchSchema = z.object({
  host_id: z.string().optional(),
  decision: z.string().optional(),
  after: z.string().optional(),
});

export const Route = createFileRoute("/_authenticated/santa/events")({
  validateSearch: (search) => searchSchema.parse(search),
  component: SantaEventsPage,
});
