import { AppBreadcrumbs } from "@/components/layout/app-breadcrumbs";
import { PageActionsSlot } from "@/components/layout/page-actions";

export function AppTopbar() {
  return (
    <header className="bg-background flex h-12 shrink-0 items-center gap-3 border-b px-4 lg:px-6">
      <AppBreadcrumbs />
      <div className="flex-1" />
      <PageActionsSlot className="flex items-center gap-2" />
    </header>
  );
}
