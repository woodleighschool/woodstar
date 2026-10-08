import { Outlet, useRouter } from "@tanstack/react-router";
import { useEffect, useRef } from "react";

import { AppSidebar } from "@components/layout/app-sidebar";
import { AppTopbar } from "@components/layout/app-topbar";
import { SidebarInset, SidebarProvider } from "@components/ui/sidebar";
import { AuthzProvider } from "@features/authz/access";

export function AppLayout() {
  const router = useRouter();
  const content = useRef<HTMLDivElement>(null);

  // The router resets window scroll on navigation, but pages scroll in this container.
  useEffect(
    () =>
      router.subscribe("onRendered", ({ pathChanged }) => {
        if (pathChanged) content.current?.scrollTo({ top: 0 });
      }),
    [router],
  );

  return (
    <AuthzProvider>
      <SidebarProvider>
        <AppSidebar />
        <SidebarInset className="h-svh min-h-0 w-auto min-w-0 overflow-clip">
          <AppTopbar />
          <div ref={content} className="min-h-0 min-w-0 flex-1 overflow-y-auto">
            <Outlet />
          </div>
        </SidebarInset>
      </SidebarProvider>
    </AuthzProvider>
  );
}
