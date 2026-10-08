import { partialMatchKey, type QueryClient, type QueryKey } from "@tanstack/react-query";

// Invalidates `stale` after the record cached under `deleted` was deleted. The
// record's own queries can only fail now, so they go stale without a refetch
// and the page showing them keeps its data until it leaves.
export async function invalidateAfterDelete(
  queryClient: QueryClient,
  deleted: QueryKey,
  stale: readonly QueryKey[],
): Promise<void> {
  void queryClient.invalidateQueries({ queryKey: deleted, refetchType: "none" });
  await Promise.all(
    stale.map((queryKey) =>
      queryClient.invalidateQueries({
        queryKey,
        predicate: (query) => !partialMatchKey(query.queryKey, deleted),
      }),
    ),
  );
}
