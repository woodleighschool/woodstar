import { linkOptions, type ActiveOptions } from "@tanstack/react-router";
import {
  Database,
  type LucideIcon,
  Package,
  PackageSearch,
  ServerCog,
  ShieldCheck,
  Tag,
  UsersRound,
} from "lucide-react";

import type { PermissionRequirement } from "@features/authz/permissions";
import type { Account } from "@lib/api";

import { filterNavigation, firstNavigationTarget } from "./navigation";

export interface NavItem {
  label: string;
  to?: string;
  activeOptions?: ActiveOptions;
  icon?: LucideIcon;
  disabled?: boolean;
  permission?: PermissionRequirement;
  items?: readonly NavItem[];
}

export interface NavMenu {
  label: string;
  items: readonly NavItem[];
}

const navSections: NavMenu[] = [
  {
    label: "Inventory",
    items: linkOptions([
      {
        label: "Hosts",
        to: "/hosts",
        icon: ServerCog,
        permission: { resource: "hosts", access: "view" },
      },
      {
        label: "Software",
        to: "/software",
        icon: Package,
        permission: { resource: "software", access: "view" },
      },
    ]),
  },
  {
    label: "Integrations",
    items: [
      {
        label: "osquery",
        to: "/osquery",
        activeOptions: { exact: true },
        icon: Database,
        items: linkOptions([
          {
            label: "Overview",
            to: "/osquery",
            activeOptions: { exact: true },
            permission: { resource: "osquery.overview", access: "view" },
          },
          {
            label: "Reports",
            to: "/osquery/reports",
            permission: { resource: "osquery.reports", access: "view" },
          },
          {
            label: "Policies",
            to: "/osquery/policies",
            permission: { resource: "osquery.policies", access: "view" },
          },
        ]),
      },
      {
        label: "Santa",
        to: "/santa",
        activeOptions: { exact: true },
        icon: ShieldCheck,
        items: linkOptions([
          {
            label: "Overview",
            to: "/santa",
            activeOptions: { exact: true },
            permission: { resource: "santa.configurations", access: "view" },
          },
          {
            label: "Configurations",
            to: "/santa/configurations",
            permission: { resource: "santa.configurations", access: "view" },
          },
          {
            label: "Rules",
            to: "/santa/rules",
            permission: { resource: "santa.rules", access: "view" },
          },
          {
            label: "Events",
            to: "/santa/events",
            permission: { resource: "santa.events", access: "view" },
          },
        ]),
      },
      {
        label: "Munki",
        to: "/munki",
        activeOptions: { exact: true },
        icon: PackageSearch,
        items: linkOptions([
          {
            label: "Overview",
            to: "/munki",
            activeOptions: { exact: true },
            permission: { resource: "munki.software", access: "view" },
          },
          {
            label: "Software",
            to: "/munki/software",
            permission: { resource: "munki.software", access: "view" },
          },
          {
            label: "Packages",
            to: "/munki/packages",
            permission: { resource: "munki.packages", access: "view" },
          },
          {
            label: "Distribution Points",
            to: "/munki/distribution-points",
            permission: { resource: "munki.distribution-points", access: "view" },
          },
          {
            label: "Client Resources",
            to: "/munki/client-resources",
            permission: { resource: "munki.client-resources", access: "view" },
          },
        ]),
      },
    ],
  },
  {
    label: "System",
    items: [
      {
        label: "Directory",
        to: "/directory",
        activeOptions: { exact: true },
        icon: UsersRound,
        items: linkOptions([
          {
            label: "Overview",
            to: "/directory",
            activeOptions: { exact: true },
            permission: { resource: "directory", access: "view" },
          },
          {
            label: "Users",
            to: "/directory/users",
            permission: { resource: "users", access: "view" },
          },
          {
            label: "Groups",
            to: "/directory/groups",
            permission: { resource: "groups", access: "view" },
          },
        ]),
      },
      linkOptions({
        label: "Labels",
        to: "/labels",
        icon: Tag,
        permission: { resource: "labels", access: "view" },
      }),
    ],
  },
];

export function visibleNavSections(account: Account | undefined): NavMenu[] {
  return navSections
    .map((section) => ({
      ...section,
      items: filterNavigation(section.items, account?.effective_permissions),
    }))
    .filter((section) => section.items.length > 0);
}

export function firstAccessiblePath(account: Account | undefined): string | undefined {
  return firstNavigationTarget(visibleNavSections(account).flatMap((section) => section.items));
}
