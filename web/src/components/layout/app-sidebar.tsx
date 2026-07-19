import { Link, useRouterState } from "@tanstack/react-router";
import { ChevronRight, ChevronsUpDown, LogOut, User as UserIcon } from "lucide-react";

import { WoodstarMark } from "@/components/brand/woodstar-mark";
import { type NavItem, type NavMenu, navSections } from "@/components/layout/nav-config";
import { Pending } from "@/components/pending";
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
import { Spinner } from "@/components/ui/spinner";
import { useAuth, useLogout } from "@/hooks/use-auth";
import { runtime } from "@/lib/runtime";
import { userRoleLabel } from "@/lib/users";
import { nonEmpty } from "@/lib/utils";
export function AppSidebar() {
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  });
  return (
    <Sidebar collapsible="icon">
      <SidebarHeader>
        <SidebarBrand />
      </SidebarHeader>
      <SidebarContent>
        {navSections.map((section) => (
          <SidebarNavGroup key={section.label} section={section} pathname={pathname} />
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
        <SidebarMenuButton size="lg" render={<Link to="/hosts" />}>
          <WoodstarMark />
          <div className="grid flex-1 text-left text-sm leading-tight">
            <span className="truncate font-semibold">Woodstar</span>
            <span className="truncate text-xs text-muted-foreground">{`v${runtime.version}`}</span>
          </div>
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
      <Collapsible defaultOpen={active} className="group/collapsible" render={<SidebarMenuItem />}>
        <CollapsibleTrigger render={<SidebarMenuButton tooltip={item.label} isActive={active} />}>
          {Icon ? <Icon /> : null}
          <span>{item.label}</span>
          <ChevronRight className="ml-auto transition-transform duration-200 group-data-open/collapsible:rotate-90" />
        </CollapsibleTrigger>
        <CollapsibleContent className="sidebar-subnav-collapsible">
          <SidebarMenuSub>
            {item.items.map((child) => (
              <SidebarMenuSubItem key={child.to ?? child.label}>
                {child.to ? (
                  <SidebarMenuSubButton
                    render={<Link to={child.to} />}
                    isActive={isActivePath(pathname, child)}
                  >
                    <span>{child.label}</span>
                  </SidebarMenuSubButton>
                ) : (
                  <SidebarMenuSubButton isActive={isActivePath(pathname, child)}>
                    <span>{child.label}</span>
                  </SidebarMenuSubButton>
                )}
              </SidebarMenuSubItem>
            ))}
          </SidebarMenuSub>
        </CollapsibleContent>
      </Collapsible>
    );
  }
  return (
    <SidebarMenuItem>
      <SidebarMenuButton
        render={item.to && !item.disabled ? <Link to={item.to} /> : undefined}
        tooltip={item.label}
        isActive={active}
        disabled={item.disabled}
      >
        {Icon ? <Icon /> : null}
        <span>{item.label}</span>
      </SidebarMenuButton>
    </SidebarMenuItem>
  );
}
function SidebarUserMenu() {
  const { isMobile } = useSidebar();
  const { user } = useAuth();
  const logout = useLogout();
  const label = nonEmpty(user?.name) ?? nonEmpty(user?.email) ?? "Signed out";
  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <SidebarMenuButton
                size="lg"
                className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
              />
            }
          >
            <SidebarUserAvatar />
            <div className="grid flex-1 text-left text-sm leading-tight">
              <span className="truncate font-medium">{label}</span>
              {user?.role ? (
                <span className="truncate text-xs text-muted-foreground">
                  {userRoleLabel(user.role)}
                </span>
              ) : null}
            </div>
            <ChevronsUpDown className="ml-auto" />
          </DropdownMenuTrigger>
          <DropdownMenuContent
            className="w-(--anchor-width) min-w-56 rounded-lg"
            side={isMobile ? "bottom" : "right"}
            align="end"
            sideOffset={4}
          >
            <DropdownMenuGroup>
              <DropdownMenuLabel className="p-0 font-normal">
                <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                  <SidebarUserAvatar />
                  <div className="grid flex-1 text-left text-sm leading-tight">
                    <span className="truncate font-medium">{label}</span>
                    <span className="truncate text-xs text-muted-foreground">
                      {user?.email ?? "Not signed in"}
                    </span>
                  </div>
                </div>
              </DropdownMenuLabel>
            </DropdownMenuGroup>
            <DropdownMenuSeparator />
            <DropdownMenuGroup>
              <DropdownMenuItem render={<Link to="/account" />}>
                <UserIcon />
                Account
              </DropdownMenuItem>
            </DropdownMenuGroup>
            <DropdownMenuSeparator />
            <DropdownMenuGroup>
              <Pending
                isPending={logout.isPending}
                render={<DropdownMenuItem onClick={() => logout.mutate()} />}
              >
                {logout.isPending ? <Spinner /> : <LogOut />}
                Sign out
              </Pending>
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}
function SidebarUserAvatar() {
  return (
    <Avatar className="rounded-lg">
      <AvatarFallback className="rounded-lg">
        <UserIcon className="size-4" />
      </AvatarFallback>
    </Avatar>
  );
}
function isActivePath(pathname: string, item: NavItem): boolean {
  if (item.to && (pathname === item.to || pathname.startsWith(`${item.to}/`))) return true;
  return item.items?.some((child) => isActivePath(pathname, child)) ?? false;
}
