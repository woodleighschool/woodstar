import { AppBreadcrumbs } from "@/components/layout/app-breadcrumbs";
import { SidebarTrigger } from "@/components/ui/sidebar";

export function AppTopbar() {
  return (
    <header className="bg-background flex h-12 shrink-0 items-center gap-3 border-b px-4 lg:px-6">
      <SidebarTrigger className="md:hidden" />
      <AppBreadcrumbs />
      <div className="flex-1" />
    </header>
  );
}
