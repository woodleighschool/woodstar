import assert from "node:assert/strict";
import { test } from "node:test";

import { QueryClient, QueryObserver, type QueryKey } from "@tanstack/react-query";

import { invalidateAfterDelete } from "./query-invalidation.ts";

void test("a deleted record's queries go stale without a refetch", async () => {
  const queryClient = new QueryClient();
  const fetches = new Map<string, number>();
  const observe = (queryKey: QueryKey) => {
    const name = queryKey.join("/");
    const observer = new QueryObserver(queryClient, {
      queryKey,
      queryFn: () => {
        fetches.set(name, (fetches.get(name) ?? 0) + 1);
        return name;
      },
      staleTime: Infinity,
    });
    return observer.subscribe(() => undefined);
  };
  const unsubscribe = [
    observe(["hosts", "detail", 1]),
    observe(["hosts", "detail", 1, "software"]),
    observe(["hosts", "detail", 12]),
    observe(["hosts", "list"]),
    observe(["labels", "list"]),
  ];
  await queryClient.refetchQueries();
  fetches.clear();

  await invalidateAfterDelete(queryClient, ["hosts", "detail", 1], [["hosts"]]);

  assert.deepEqual(Object.fromEntries(fetches), { "hosts/detail/12": 1, "hosts/list": 1 });
  const invalidated = (queryKey: QueryKey) => queryClient.getQueryState(queryKey)?.isInvalidated;
  assert.equal(invalidated(["hosts", "detail", 1]), true);
  assert.equal(invalidated(["hosts", "detail", 1, "software"]), true);
  assert.equal(invalidated(["hosts", "detail", 12]), false);
  assert.equal(invalidated(["labels", "list"]), false);

  for (const stop of unsubscribe) stop();
  queryClient.clear();
});
