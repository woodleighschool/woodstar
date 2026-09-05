import { useRouterState } from "@tanstack/react-router";
import { ChevronRight, ChevronsUpDown, History, LogOut, User as UserIcon } from "lucide-react";
import { useState } from "react";

import {
  firstAccessiblePath,
  type NavItem,
  type NavMenu,
  visibleNavSections,
} from "@components/layout/nav-config";
import { Link } from "@components/link";
import { Logo } from "@components/logo";
import { Pending } from "@components/pending";
import { Avatar, AvatarFallback } from "@components/ui/avatar";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@components/ui/collapsible";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@components/ui/dropdown-menu";
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
  useSidebar,
} from "@components/ui/sidebar";
import { Spinner } from "@components/ui/spinner";
import { useAccount } from "@features/account/queries";
import { useAuth, useLogout } from "@features/authn/queries";
import { Can } from "@features/authz/access";
import { runtime } from "@lib/runtime";
import { nonEmpty } from "@lib/utils";
export function AppSidebar() {
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  });
  const account = useAccount();
  const sections = visibleNavSections(account.data);
  const home = firstAccessiblePath(account.data) ?? "/account";
  return (
    <Sidebar variant="floating" collapsible="icon">
      <SidebarHeader>
        <SidebarBrand to={home} />
      </SidebarHeader>
      <SidebarContent>
        {sections.map((section) => (
          <SidebarNavGroup key={section.label} section={section} pathname={pathname} />
        ))}
      </SidebarContent>
      <SidebarFooter>
        <SidebarUserMenu />
      </SidebarFooter>
    </Sidebar>
  );
}

function SidebarBrand({ to }: { to: string }) {
  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <SidebarMenuButton size="lg" render={<Link to={to} />}>
          <Logo />
          <div className="grid flex-1 text-left text-sm/tight">
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
      <SidebarGroupLabel className="group-data-[collapsible=icon]:pointer-events-none">
        {section.label}
      </SidebarGroupLabel>
      <SidebarMenu>
        {section.items.map((item) => (
          <SidebarNavItem key={item.label} item={item} pathname={pathname} />
        ))}
      </SidebarMenu>
    </SidebarGroup>
  );
}
function SidebarNavItem({ item, pathname }: { item: NavItem; pathname: string }) {
  const { isMobile, state } = useSidebar();
  const Icon = item.icon;
  const active = isActivePath(pathname, item);
  const collapsedOverviewTarget =
    state === "collapsed" && !isMobile && !item.disabled ? item.to : undefined;
  const [manuallyOpen, setManuallyOpen] = useState(false);

  if (item.items?.length) {
    if (collapsedOverviewTarget) {
      return (
        <SidebarMenuItem>
          <SidebarMenuButton
            render={<Link to={collapsedOverviewTarget} activeOptions={item.activeOptions} />}
            tooltip={item.label}
            isActive={active}
          >
            {Icon ? <Icon /> : null}
            <span>{item.label}</span>
          </SidebarMenuButton>
        </SidebarMenuItem>
      );
    }

    return (
      <Collapsible
        open={active || manuallyOpen}
        onOpenChange={setManuallyOpen}
        className="group/collapsible"
        render={<SidebarMenuItem />}
      >
        <CollapsibleTrigger render={<SidebarMenuButton tooltip={item.label} isActive={active} />}>
          {Icon ? <Icon /> : null}
          <span>{item.label}</span>
          <ChevronRight className="ml-auto transition-transform duration-200 group-data-open/collapsible:rotate-90" />
        </CollapsibleTrigger>
        <CollapsibleContent className="h-(--collapsible-panel-height) overflow-hidden opacity-100 transition-[height,opacity] duration-200 ease-out data-ending-style:h-0 data-ending-style:opacity-0 data-starting-style:h-0 data-starting-style:opacity-0 motion-reduce:transition-none">
          <SidebarMenuSub>
            {item.items.map((child) => (
              <SidebarMenuSubItem key={child.to ?? child.label}>
                {child.to ? (
                  <SidebarMenuSubButton
                    render={<Link to={child.to} activeOptions={child.activeOptions} />}
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
        render={
          item.to && !item.disabled ? (
            <Link to={item.to} activeOptions={item.activeOptions} />
          ) : undefined
        }
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
  const label = nonEmpty(user?.name) ?? nonEmpty(user?.email) ?? "Signed Out";
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
            <div className="grid flex-1 text-left text-sm/tight">
              <span className="truncate font-medium">{label}</span>
              <span className="truncate text-xs text-muted-foreground">{user?.email}</span>
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
                  <div className="grid flex-1 text-left text-sm/tight">
                    <span className="truncate font-medium">{label}</span>
                    <span className="truncate text-xs text-muted-foreground">
                      {user?.email ?? "Not Signed In"}
                    </span>
                  </div>
                </div>
              </DropdownMenuLabel>
            </DropdownMenuGroup>
            <DropdownMenuSeparator />
            <DropdownMenuGroup>
              <Can resource="activity" access="view">
                <DropdownMenuItem render={<Link to="/activity" />}>
                  <History />
                  Activity
                </DropdownMenuItem>
              </Can>
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
  if (item.to && pathname === item.to) return true;
  if (item.to && !item.activeOptions?.exact && pathname.startsWith(`${item.to}/`)) return true;
  return item.items?.some((child) => isActivePath(pathname, child)) ?? false;
}
