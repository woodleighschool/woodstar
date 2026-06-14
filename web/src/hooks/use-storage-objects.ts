import { useQuery } from "@tanstack/react-query";

import { MAX_PAGE_SIZE } from "@/hooks/use-data-table-search";
import type { ApiError, StorageObject } from "@/lib/api";
import { apiClient, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export type { StorageObject };

interface StorageObjectsPage {
  items: StorageObject[];
  count: number;
}

// useStorageObjects lists available objects under a prefix, e.g. munki/icons for
// the icon picker.
export function useStorageObjects(prefix: string, enabled = true) {
  const query = { prefix, page: 1, per_page: MAX_PAGE_SIZE };
  return useQuery<StorageObjectsPage, ApiError>({
    queryKey: queryKeys.storageObjects(query),
    queryFn: ({ signal }) =>
      unwrap(apiClient.GET("/api/storage/objects", { params: { query }, signal })),
    enabled,
  });
}
