import { Outlet } from "@tanstack/react-router";

import { AppSidebar } from "@/components/layout/app-sidebar";
import { AppTopbar } from "@/components/layout/app-topbar";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { useSessionGuard } from "@/hooks/use-auth";

export function AppLayout() {
  useSessionGuard();

  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset className="w-auto min-w-0">
        <AppTopbar />
        <main className="min-w-0 flex-1 overflow-y-auto">
          <Outlet />
        </main>
      </SidebarInset>
    </SidebarProvider>
  );
}
