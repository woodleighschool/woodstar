import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { z } from "zod";

import { HostListPage } from "@features/hosts/list";
import { createTableSearchSchema, TABLE_SEARCH_DEFAULTS } from "@lib/table-search";

const searchSchema = createTableSearchSchema([
  "display_name",
  "hardware.serial",
  "hardware.model_identifier",
  "hardware.uuid",
  "os.version",
  "agents.osquery.version",
  "last_contact",
  "last_restarted_at",
  "storage.boot_volume.available_bytes",
  "hardware.memory_bytes",
  "network.primary_ip",
  "public_ip",
]).extend({
  status: z.enum(["online", "offline"]).optional().catch(undefined),
  label_id: z.coerce.number().int().positive().optional().catch(undefined),
  software_title_id: z.coerce.number().int().positive().optional().catch(undefined),
  software_id: z.coerce.number().int().positive().optional().catch(undefined),
});

export const Route = createFileRoute("/_authenticated/hosts/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(TABLE_SEARCH_DEFAULTS)] },
  component: HostListPage,
});
