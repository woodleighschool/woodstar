import { Link, useRouterState } from "@tanstack/react-router";
import { ChevronRight, ChevronsUpDown, LogOut, Monitor, Moon, Sun, User as UserIcon } from "lucide-react";
import { useTheme } from "next-themes";

import { WoodstarMark } from "@/components/brand/woodstar-mark";
import { navSections, type NavItem, type NavMenu } from "@/components/layout/nav-config";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
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
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  SidebarRail,
  useSidebar,
} from "@/components/ui/sidebar";
import { userRoleLabel } from "@/components/users/user-role";
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
      items: visibleItems(section.items, isAdmin),
    }))
    .filter((section) => section.items.length > 0);

  return (
    <Sidebar collapsible="icon">
      <SidebarHeader>
        <SidebarBrand />
      </SidebarHeader>
      <SidebarContent>
        {visibleSections.map((section) => (
          <SidebarNavGroup key={section.label} section={section} pathname={location.pathname} />
        ))}
      </SidebarContent>
      <SidebarFooter>
        <SidebarUserMenu />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}

function SidebarBrand() {
  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <SidebarMenuButton size="lg" asChild>
          <Link to="/hosts">
            <WoodstarMark />
            <div className="grid flex-1 text-left text-sm leading-tight">
              <span className="truncate font-semibold">Woodstar</span>
              <span className="text-muted-foreground truncate text-xs">{`v${runtime.version}`}</span>
            </div>
          </Link>
        </SidebarMenuButton>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}

function SidebarNavGroup({ section, pathname }: { section: NavMenu; pathname: string }) {
  return (
    <SidebarGroup>
      <SidebarGroupLabel>{section.label}</SidebarGroupLabel>
      <SidebarMenu>
        {section.items.map((item) => (
          <SidebarNavItem key={item.label} item={item} pathname={pathname} />
        ))}
      </SidebarMenu>
    </SidebarGroup>
  );
}

function SidebarNavItem({ item, pathname }: { item: NavItem; pathname: string }) {
  const Icon = item.icon;
  const active = isActivePath(pathname, item);

  if (item.items?.length) {
    return (
      <Collapsible asChild defaultOpen={active} className="group/collapsible">
        <SidebarMenuItem>
          <CollapsibleTrigger asChild>
            <SidebarMenuButton tooltip={item.label} isActive={active}>
              {Icon ? <Icon /> : null}
              <span>{item.label}</span>
              <ChevronRight className="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90" />
            </SidebarMenuButton>
          </CollapsibleTrigger>
          <CollapsibleContent className="sidebar-subnav-collapsible">
            <SidebarMenuSub>
              {item.items.map((child) => (
                <SidebarMenuSubItem key={child.to ?? child.label}>
                  <SidebarMenuSubButton asChild={!!child.to} isActive={isActivePath(pathname, child)}>
                    {child.to ? (
                      <Link to={child.to}>
                        <span>{child.label}</span>
                      </Link>
                    ) : (
                      <span>{child.label}</span>
                    )}
                  </SidebarMenuSubButton>
                </SidebarMenuSubItem>
              ))}
            </SidebarMenuSub>
          </CollapsibleContent>
        </SidebarMenuItem>
      </Collapsible>
    );
  }

  return (
    <SidebarMenuItem>
      <SidebarMenuButton
        asChild={!!item.to && !item.disabled}
        tooltip={item.label}
        isActive={active}
        disabled={item.disabled}
      >
        {item.to && !item.disabled ? (
          <Link to={item.to}>
            {Icon ? <Icon /> : null}
            <span>{item.label}</span>
          </Link>
        ) : (
          <>
            {Icon ? <Icon /> : null}
            <span>{item.label}</span>
          </>
        )}
      </SidebarMenuButton>
    </SidebarMenuItem>
  );
}

function SidebarUserMenu() {
  const { isMobile } = useSidebar();
  const { setTheme } = useTheme();
  const { user } = useAuth();
  const logout = useLogout();
  const label = nonEmpty(user?.name) ?? nonEmpty(user?.email) ?? "Signed out";
  const initials = label
    .split(/[\s@]+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((s) => s[0].toUpperCase())
    .join("");

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <SidebarMenuButton
              size="lg"
              className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
            >
              <Avatar className="rounded-lg">
                <AvatarFallback className="rounded-lg">{initials || "?"}</AvatarFallback>
              </Avatar>
              <div className="grid flex-1 text-left text-sm leading-tight">
                <span className="truncate font-medium">{label}</span>
                {user?.role ? (
                  <span className="text-muted-foreground truncate text-xs">{userRoleLabel(user.role)}</span>
                ) : null}
              </div>
              <ChevronsUpDown className="ml-auto" />
            </SidebarMenuButton>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            className="w-(--radix-dropdown-menu-trigger-width) min-w-56 rounded-lg"
            side={isMobile ? "bottom" : "right"}
            align="end"
            sideOffset={4}
          >
            <DropdownMenuLabel className="p-0 font-normal">
              <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                <Avatar className="rounded-lg">
                  <AvatarFallback className="rounded-lg">{initials || "?"}</AvatarFallback>
                </Avatar>
                <div className="grid flex-1 text-left text-sm leading-tight">
                  <span className="truncate font-medium">{label}</span>
                  <span className="text-muted-foreground truncate text-xs">{user?.email ?? "Not signed in"}</span>
                </div>
              </div>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuGroup>
              <DropdownMenuItem asChild>
                <Link to="/account">
                  <UserIcon />
                  Account
                </Link>
              </DropdownMenuItem>
            </DropdownMenuGroup>
            <DropdownMenuSeparator />
            <DropdownMenuGroup>
              <DropdownMenuItem onSelect={() => setTheme("light")}>
                <Sun />
                Light
              </DropdownMenuItem>
              <DropdownMenuItem onSelect={() => setTheme("dark")}>
                <Moon />
                Dark
              </DropdownMenuItem>
              <DropdownMenuItem onSelect={() => setTheme("system")}>
                <Monitor />
                System
              </DropdownMenuItem>
            </DropdownMenuGroup>
            <DropdownMenuSeparator />
            <DropdownMenuItem onSelect={() => logout.mutate()} disabled={logout.isPending}>
              <LogOut />
              Sign out
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}

function isActivePath(pathname: string, item: NavItem): boolean {
  if (item.to && (pathname === item.to || pathname.startsWith(`${item.to}/`))) return true;
  return item.items?.some((child) => isActivePath(pathname, child)) ?? false;
}

function visibleItems(items: NavItem[], isAdmin: boolean): NavItem[] {
  return items
    .filter((item) => !item.adminOnly || isAdmin)
    .map((item) => ({ ...item, items: item.items ? visibleItems(item.items, isAdmin) : undefined }))
    .filter((item) => !item.items || item.items.length > 0);
}
