import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { z } from "zod";

import { createListSearchSchema, LIST_SEARCH_DEFAULTS } from "@/lib/list-search";
import { HostListPage } from "@/pages/hosts/list";

const searchSchema = createListSearchSchema([
  "display_name",
  "hardware.serial",
  "hardware.model_identifier",
  "hardware.uuid",
  "os.version",
  "agents.osquery.version",
  "timestamps.last_seen_at",
  "timestamps.last_restarted_at",
  "storage.boot_volume.available_bytes",
  "hardware.memory_bytes",
  "network.primary_ip",
  "network.last_remote_ip",
]).extend({
  status: z.enum(["online", "offline"]).optional().catch(undefined),
  label_id: z.coerce.number().int().positive().optional().catch(undefined),
  software_title_id: z.coerce.number().int().positive().optional().catch(undefined),
  software_id: z.coerce.number().int().positive().optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/hosts/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(LIST_SEARCH_DEFAULTS)] },
  component: HostListPage,
});
