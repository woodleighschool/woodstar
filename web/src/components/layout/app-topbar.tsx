import { SidebarTrigger } from "@/components/ui/sidebar";

export function AppTopbar() {
  return (
    <header className="flex h-12 shrink-0 items-center gap-3 border-b bg-background px-4 md:hidden">
      <SidebarTrigger />
      <div className="flex-1" />
    </header>
  );
}
