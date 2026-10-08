import { MutationCache, QueryClient } from "@tanstack/react-query";

import { toast } from "@components/ui/toast";
import { ApiError } from "@lib/api";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      refetchOnMount: true,
      refetchOnWindowFocus: true,
      refetchOnReconnect: true,
      // A 4xx response is the server's answer; asking again returns the same one.
      retry: (failureCount, error) =>
        !(error instanceof ApiError && error.status >= 400 && error.status < 500) &&
        failureCount < 2,
      retryOnMount: false,
    },
  },
  mutationCache: new MutationCache({
    onError: (error, _variables, _context, mutation) => {
      if (mutation.meta?.inlineError || mutation.options.onError) return;
      toast.add({
        title: error instanceof Error ? error.message : "Request Failed",
        type: "error",
      });
    },
  }),
});
