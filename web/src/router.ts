import { createRouter } from "@tanstack/react-router";

import { queryClient } from "./query-client";
import { RouteErrorPage } from "./route-error";
import { routeTree } from "./routeTree.gen";

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
