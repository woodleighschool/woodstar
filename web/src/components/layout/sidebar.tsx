import { Link, useRouterState } from "@tanstack/react-router";
import { Star } from "lucide-react";

import { navSections } from "@/components/layout/nav-config";
import { useAuth } from "@/hooks/use-auth";
import { useVersion } from "@/hooks/use-version";
import { cn } from "@/lib/utils";

export function Sidebar({ onNavigate }: { onNavigate?: () => void }) {
  const { location } = useRouterState();
  const { data: version } = useVersion();
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";

  const visibleSections = navSections
    .map((section) => ({
      ...section,
      items: section.items.filter((item) => !item.adminOnly || isAdmin),
    }))
    .filter((section) => section.items.length > 0);

  return (
    <nav className="flex h-full flex-col bg-sidebar text-sidebar-foreground border-r border-sidebar-border">
      <div className="flex h-14 items-center gap-2 px-4 border-b border-sidebar-border">
        <Star className="size-5 text-primary" aria-hidden />
        <span className="text-base font-semibold tracking-tight">Woodstar</span>
      </div>

      <div className="flex-1 overflow-y-auto px-2 py-3 space-y-4">
        {visibleSections.map((section, index) => (
          <div key={section.label ?? `section-${index}`} className="space-y-0.5">
            {section.label ? (
              <div className="px-2 pb-1 text-[10px] uppercase tracking-wider text-muted-foreground font-semibold">
                {section.label}
              </div>
            ) : null}
            {section.items.map((item) => {
              const Icon = item.icon;
              const active = location.pathname === item.to || location.pathname.startsWith(`${item.to}/`);

              return (
                <Link
                  key={item.to}
                  to={item.to}
                  onClick={onNavigate}
                  className={cn(
                    "flex items-center gap-2.5 rounded-md px-2 py-1.5 text-sm font-medium transition-colors",
                    active
                      ? "bg-sidebar-accent text-sidebar-accent-foreground"
                      : "text-sidebar-foreground/80 hover:bg-sidebar-accent/50 hover:text-sidebar-accent-foreground",
                  )}
                >
                  <Icon className="size-4 shrink-0" aria-hidden />
                  <span className="flex-1 truncate">{item.label}</span>
                </Link>
              );
            })}
          </div>
        ))}
      </div>

      <div className="border-t border-sidebar-border px-3 py-2 text-[11px] text-muted-foreground">
        <div className="flex items-center justify-between gap-2">
          <span className="truncate" title={version?.version}>
            {version ? `v${version.version}` : "version unknown"}
          </span>
          <span className="font-mono">macadmin</span>
        </div>
      </div>
    </nav>
  );
}
