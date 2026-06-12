import { MutationCache, QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { RouterProvider } from "@tanstack/react-router";
import { TanStackRouterDevtools } from "@tanstack/router-devtools";
import { ThemeProvider } from "next-themes";
import { toast, Toaster } from "sonner";

import { TooltipProvider } from "@/components/ui/tooltip";
import { router } from "@/router";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
  // Unhandled mutation errors surface as a toast. Resource forms that render their
  // submit error inline opt out with meta: { inlineError: true }; mutations with an
  // explicit onError handle it themselves.
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
