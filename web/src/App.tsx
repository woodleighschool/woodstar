import { QueryClientProvider } from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { RouterProvider } from "@tanstack/react-router";
import { TanStackRouterDevtools } from "@tanstack/react-router-devtools";
import { Toaster } from "sonner";

import { TooltipProvider } from "@/components/ui/tooltip";
import { setUnauthorizedHandler } from "@/lib/api";
import { expireSession } from "@/lib/session-expiry";
import { queryClient } from "@/query-client";
import { router } from "@/router";

declare module "@tanstack/react-query" {
  // Typed mutation meta. inlineError renders a mutation's error on the form
  // instead of the global toast (the pre-auth login form).
  interface Register {
    mutationMeta: { inlineError?: boolean };
  }
}

setUnauthorizedHandler(() =>
  expireSession(queryClient, router.state.location.pathname, () =>
    router.navigate({ to: "/login", replace: true }),
  ),
);

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <TooltipProvider delay={150}>
        <RouterProvider router={router} />
        <Toaster theme="system" richColors closeButton position="bottom-right" />
      </TooltipProvider>
      {import.meta.env.DEV ? (
        <>
          <ReactQueryDevtools buttonPosition="bottom-left" />
          <TanStackRouterDevtools router={router} position="bottom-right" />
        </>
      ) : null}
    </QueryClientProvider>
  );
}
