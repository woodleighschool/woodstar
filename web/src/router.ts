import { createRouter } from "@tanstack/react-router";

import { RouteErrorPage } from "@/pages/route-error";
import { queryClient } from "@/query-client";
import { routeTree } from "@/routeTree.gen";

export const router = createRouter({
  routeTree,
  context: { queryClient },
  defaultPreload: "intent",
  defaultErrorComponent: RouteErrorPage,
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}
