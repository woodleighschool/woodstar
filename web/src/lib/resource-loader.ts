import type { QueryClient, QueryKey, UseQueryOptions } from "@tanstack/react-query";
import { notFound, type AnyRoute } from "@tanstack/react-router";

import { ApiError } from "@lib/api";

// Loads a route's resource before the route renders. A resource the API can't
// find makes the route a not-found page.
export async function loadResource<TData, TError, TQueryKey extends QueryKey>(
  { context, route }: { context: { queryClient: QueryClient }; route: AnyRoute },
  options: UseQueryOptions<TData, TError, TData, TQueryKey>,
): Promise<void> {
  try {
    await context.queryClient.query({
      ...options,
      // Cached data is enough to render with unless its last fetch failed.
      staleTime: (query) => (query.state.status === "error" ? 0 : "static"),
    });
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) throw notFound({ routeId: route.id });
    throw error;
  }
}
