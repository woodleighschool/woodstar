import { Outlet } from "@tanstack/react-router";

import { AppSidebar } from "@components/layout/app-sidebar";
import { AppTopbar } from "@components/layout/app-topbar";
import { SidebarInset, SidebarProvider } from "@components/ui/sidebar";
import { AuthzProvider } from "@features/authz/access";

export function AppLayout() {
  return (
    <AuthzProvider>
      <SidebarProvider>
        <AppSidebar />
        <SidebarInset className="h-svh min-h-0 w-auto min-w-0 overflow-clip">
          <AppTopbar />
          <div className="min-h-0 min-w-0 flex-1 overflow-y-auto">
            <Outlet />
          </div>
        </SidebarInset>
      </SidebarProvider>
    </AuthzProvider>
  );
}
