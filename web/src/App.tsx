import { MutationCache, QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { RouterProvider } from "@tanstack/react-router";
import { TanStackRouterDevtools } from "@tanstack/react-router-devtools";
import { ThemeProvider } from "next-themes";
import { toast, Toaster } from "sonner";

import { TooltipProvider } from "@/components/ui/tooltip";
import { router } from "@/router";

declare module "@tanstack/react-query" {
  // Typed mutation meta. inlineError renders a mutation's error on the form
  // instead of the global toast (the pre-auth login/setup forms).
  interface Register {
    mutationMeta: { inlineError?: boolean };
  }
}

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      refetchOnMount: true,
      refetchOnWindowFocus: true,
      refetchOnReconnect: true,
      refetchInterval: 30_000,
      refetchIntervalInBackground: false,
      retry: false,
      retryOnMount: false,
    },
  },
  // Mutation errors surface as a toast unless the mutation handles them itself:
  // an explicit onError, or meta.inlineError for the auth forms that render the
  // error in place.
  mutationCache: new MutationCache({
    onError: (error, _variables, _context, mutation) => {
      if (mutation.meta?.inlineError || mutation.options.onError) return;
      toast.error(error instanceof Error ? error.message : "Request failed");
    },
  }),
});

export default function App() {
  return (
    <ThemeProvider attribute="class" defaultTheme="system" enableSystem disableTransitionOnChange>
      <QueryClientProvider client={queryClient}>
        <TooltipProvider delayDuration={150}>
          <RouterProvider router={router} />
          <Toaster richColors closeButton position="bottom-right" />
        </TooltipProvider>
        {import.meta.env.DEV ? (
          <>
            <ReactQueryDevtools buttonPosition="bottom-left" />
            <TanStackRouterDevtools router={router} position="bottom-right" />
          </>
        ) : null}
      </QueryClientProvider>
    </ThemeProvider>
  );
}
