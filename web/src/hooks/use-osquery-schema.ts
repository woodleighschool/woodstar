import { useQuery } from "@tanstack/react-query";

// Vite resolves @schema to ../schema (see vite.config.ts) and rewrites the
// `?url` import to a hashed asset URL at build time.
import schemaUrl from "@schema/osquery_fleet_schema.json?url";

import { queryKeys } from "@/lib/query-keys";
import type { OsqueryTable } from "@/lib/schema";

export type { OsqueryColumn, OsqueryTable } from "@/lib/schema";

export function useOsquerySchema() {
  return useQuery<OsqueryTable[]>({
    queryKey: queryKeys.osquerySchema,
    queryFn: async ({ signal }) => {
      const response = await fetch(schemaUrl, { signal });
      if (!response.ok) {
        throw new Error(`schema ${response.status}`);
      }
      const data = (await response.json()) as OsqueryTable[];
      // Filter hidden tables once; the sidebar consumes this list directly.
      return data.filter((table) => !table.hidden);
    },
    staleTime: Infinity,
    gcTime: Infinity,
    refetchInterval: false,
  });
}
