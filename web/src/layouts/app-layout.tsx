import { Outlet } from "@tanstack/react-router";
import { useState } from "react";

import { Sidebar } from "@/components/layout/sidebar";
import { Topbar } from "@/components/layout/topbar";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { useAuth } from "@/hooks/use-auth";

export function AppLayout() {
  const [mobileNavOpen, setMobileNavOpen] = useState(false);
  const { user } = useAuth();

  return (
    <div className="flex h-dvh bg-background text-foreground">
      <aside className="hidden lg:flex w-60 shrink-0">
        <Sidebar />
      </aside>

      <Dialog open={mobileNavOpen} onOpenChange={setMobileNavOpen}>
        <DialogContent className="lg:hidden left-0 top-0 translate-x-0 translate-y-0 max-w-[16rem] h-dvh rounded-none border-r p-0">
          <Sidebar onNavigate={() => setMobileNavOpen(false)} />
        </DialogContent>
      </Dialog>

      <div className="flex flex-1 flex-col min-w-0">
        <Topbar user={user} onOpenMobileNav={() => setMobileNavOpen(true)} />
        <main className="flex-1 overflow-y-auto">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
