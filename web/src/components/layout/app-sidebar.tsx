import { Link, useRouterState } from "@tanstack/react-router";
import { ChevronRight, LogOut, User as UserIcon } from "lucide-react";

import { WoodstarMark } from "@/components/brand/woodstar-mark";
import { navSections, type NavItem } from "@/components/layout/nav-config";
import { ThemeToggle } from "@/components/theme-toggle";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  SidebarRail,
} from "@/components/ui/sidebar";
import { useAuth, useLogout } from "@/hooks/use-auth";
import { runtime } from "@/lib/runtime";
import { nonEmpty } from "@/lib/utils";

export function AppSidebar() {
  const { location } = useRouterState();
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";

  const visibleSections = navSections
    .map((section) => ({
      ...section,
      items: section.items.filter((item) => !item.adminOnly || isAdmin),
    }))
    .filter((section) => section.placeholder === true || section.items.length > 0);

  return (
    <Sidebar collapsible="icon">
      <SidebarHeader className="border-sidebar-border border-b">
        <div className="flex items-center gap-2 px-2 py-1.5">
          <WoodstarMark />
          <div className="flex min-w-0 flex-col group-data-[collapsible=icon]:hidden">
            <span className="truncate text-sm font-semibold tracking-tight">Woodstar</span>
            <span className="text-muted-foreground truncate text-[11px]">{`v${runtime.version}`}</span>
          </div>
        </div>
      </SidebarHeader>

      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupContent>
            <SidebarMenu>
              {visibleSections.map((section) => {
                const Icon = section.icon;
                const active = section.items.some((item) => isActivePath(location.pathname, item));

                if (section.collapsible === false) {
                  return section.items.map((item) => {
                    const ItemIcon = item.icon;
                    return (
                      <SidebarMenuItem key={item.to}>
                        <SidebarMenuButton
                          asChild
                          tooltip={item.label}
                          isActive={isActivePath(location.pathname, item)}
                        >
                          <Link to={item.to}>
                            <ItemIcon />
                            <span>{item.label}</span>
                          </Link>
                        </SidebarMenuButton>
                      </SidebarMenuItem>
                    );
                  });
                }

                if (section.items.length === 0) {
                  return (
                    <SidebarMenuItem key={section.label}>
                      <SidebarMenuButton disabled tooltip={section.label}>
                        <Icon />
                        <span>{section.label}</span>
                        <ChevronRight className="ml-auto opacity-60" />
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  );
                }

                return (
                  <Collapsible key={section.label} asChild defaultOpen={active} className="group/collapsible">
                    <SidebarMenuItem>
                      <CollapsibleTrigger asChild>
                        <SidebarMenuButton tooltip={section.label} isActive={active}>
                          <Icon />
                          <span>{section.label}</span>
                          <ChevronRight className="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90" />
                        </SidebarMenuButton>
                      </CollapsibleTrigger>
                      <CollapsibleContent>
                        <SidebarMenuSub>
                          {section.items.map((item) => {
                            const ChildIcon = item.icon;
                            return (
                              <SidebarMenuSubItem key={item.to}>
                                <SidebarMenuSubButton asChild isActive={isActivePath(location.pathname, item)}>
                                  <Link to={item.to}>
                                    <ChildIcon />
                                    <span>{item.label}</span>
                                  </Link>
                                </SidebarMenuSubButton>
                              </SidebarMenuSubItem>
                            );
                          })}
                        </SidebarMenuSub>
                      </CollapsibleContent>
                    </SidebarMenuItem>
                  </Collapsible>
                );
              })}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      <SidebarFooter className="border-sidebar-border border-t">
        <div className="flex items-center justify-between gap-1 px-1">
          <UserMenu />
          <ThemeToggle />
        </div>
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}

function isActivePath(pathname: string, item: NavItem) {
  return pathname === item.to || pathname.startsWith(`${item.to}/`);
}

function UserMenu() {
  const { user } = useAuth();
  const logout = useLogout();

  const initials = (nonEmpty(user?.name) ?? nonEmpty(user?.email) ?? "?")
    .split(/[\s@]+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((s) => s[0].toUpperCase())
    .join("");

  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="flex min-w-0 flex-1 items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground">
        <div className="bg-sidebar-accent text-sidebar-accent-foreground flex size-7 shrink-0 items-center justify-center rounded-md text-xs font-medium">
          {initials || "?"}
        </div>
        <div className="flex min-w-0 flex-col leading-tight group-data-[collapsible=icon]:hidden">
          <span className="truncate font-medium">{nonEmpty(user?.name) ?? nonEmpty(user?.email) ?? "Signed out"}</span>
          {user?.role ? <span className="text-muted-foreground truncate text-[11px]">{user.role}</span> : null}
        </div>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" side="top" className="w-56">
        <DropdownMenuLabel className="truncate">{user?.email ?? "Not signed in"}</DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuGroup>
          <DropdownMenuItem asChild>
            <Link to="/account">
              <UserIcon />
              Account
            </Link>
          </DropdownMenuItem>
          <DropdownMenuItem onSelect={() => logout.mutate()} disabled={logout.isPending}>
            <LogOut />
            Sign out
          </DropdownMenuItem>
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
