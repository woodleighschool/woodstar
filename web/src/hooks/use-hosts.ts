import { useResourceList } from "@/hooks/use-pending-list";
import { endpoints } from "@/lib/endpoints";
import { queryKeys } from "@/lib/query-keys";
import type { Host } from "@/lib/types";

export function useHosts() {
  return useResourceList<Host>(endpoints.hosts, queryKeys.hosts());
}
