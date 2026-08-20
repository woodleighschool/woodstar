import { MutationCache, QueryClient } from "@tanstack/react-query";

import { toast } from "@components/ui/toast";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      refetchOnMount: true,
      refetchOnWindowFocus: true,
      refetchOnReconnect: true,
      retry: 2,
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
