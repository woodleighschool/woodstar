// Vite resolves @schema to ../schema (see vite.config.ts) and rewrites the
// `?url` import to a hashed asset URL at build time.
import schemaUrl from "@schema/osquery_fleet_schema.json?url";
import { useQuery } from "@tanstack/react-query";
import { z } from "zod";

import { queryKeys } from "@/lib/query-keys";
import type { OsqueryTable } from "@/lib/schema";

export type { OsqueryColumn, OsqueryTable } from "@/lib/schema";

const osqueryColumnSchema = z.looseObject({
  name: z.string(),
  type: z.string(),
  description: z.string().optional(),
  required: z.boolean().optional(),
  hidden: z.boolean().optional(),
  index: z.boolean().optional(),
});

const osqueryTableSchema: z.ZodType<OsqueryTable> = z.looseObject({
  name: z.string(),
  description: z.string().optional(),
  url: z.string().optional(),
  evented: z.boolean().optional(),
  cacheable: z.boolean().optional(),
  notes: z.string().optional(),
  examples: z
    .union([
      z.string(),
      z.array(
        z.object({
          description: z.string().optional(),
          query: z.string().optional(),
        }),
      ),
    ])
    .optional(),
  columns: z.array(osqueryColumnSchema),
  hidden: z.boolean().optional(),
});

const osquerySchema = z.array(osqueryTableSchema);

export function useOsquerySchema() {
  return useQuery<OsqueryTable[]>({
    queryKey: queryKeys.osquerySchema,
    queryFn: async ({ signal }) => {
      const response = await fetch(schemaUrl, { signal });
      if (!response.ok) {
        throw new Error(`schema ${response.status}`);
      }
      const data = osquerySchema.parse(await response.json());
      // Filter hidden tables once; the sidebar consumes this list directly.
      return data.filter((table) => !table.hidden);
    },
    staleTime: Infinity,
    gcTime: Infinity,
  });
}
