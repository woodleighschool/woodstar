import { Outlet } from "@tanstack/react-router";

import { AppSidebar } from "@/components/layout/app-sidebar";
import { AppTopbar } from "@/components/layout/app-topbar";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";

export function AppLayout() {
  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset className="min-w-0 w-auto">
        <AppTopbar />
        <main className="min-w-0 flex-1 overflow-y-auto">
          <Outlet />
        </main>
      </SidebarInset>
    </SidebarProvider>
  );
}
