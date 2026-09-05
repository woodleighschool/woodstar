import { can, type Permissions } from "@woodleighschool/authz";

import type { AuthzResource } from "@features/authz/permissions";

import type { NavItem } from "./nav-config";

export function filterNavigation(
  items: readonly NavItem[],
  permissions: Permissions<AuthzResource> | undefined,
): NavItem[] {
  return items.flatMap((item) => {
    if (item.disabled) return [];
    const children = item.items && filterNavigation(item.items, permissions);
    const allowed = item.permission
      ? can(permissions, item.permission.resource, item.permission.access)
      : !item.items;
    if (!allowed && !children?.length) return [];
    return [
      { ...item, to: allowed ? item.to : firstNavigationTarget(children ?? []), items: children },
    ];
  });
}

export function firstNavigationTarget(items: readonly NavItem[]): string | undefined {
  for (const item of items) {
    const target = item.to ?? (item.items && firstNavigationTarget(item.items));
    if (target) return target;
  }
  return undefined;
}
