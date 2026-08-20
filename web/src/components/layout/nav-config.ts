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

export interface NavItem {
  label: string;
  to?: string;
  activeOptions?: ActiveOptions;
  icon?: LucideIcon;
  disabled?: boolean;
  items?: readonly NavItem[];
}

export interface NavMenu {
  label: string;
  items: readonly NavItem[];
}

export const navSections: NavMenu[] = [
  {
    label: "Inventory",
    items: linkOptions([
      { label: "Hosts", to: "/hosts", icon: ServerCog },
      { label: "Software", to: "/software", icon: Package },
    ]),
  },
  {
    label: "Integrations",
    items: [
      {
        label: "Osquery",
        to: "/osquery",
        activeOptions: { exact: true },
        icon: Database,
        items: linkOptions([
          { label: "Overview", to: "/osquery", activeOptions: { exact: true } },
          { label: "Reports", to: "/osquery/reports" },
          { label: "Policies", to: "/osquery/policies" },
        ]),
      },
      {
        label: "Santa",
        to: "/santa",
        activeOptions: { exact: true },
        icon: ShieldCheck,
        items: linkOptions([
          { label: "Overview", to: "/santa", activeOptions: { exact: true } },
          { label: "Configurations", to: "/santa/configurations" },
          { label: "Events", to: "/santa/events" },
        ]),
      },
      {
        label: "Munki",
        to: "/munki",
        activeOptions: { exact: true },
        icon: PackageSearch,
        items: linkOptions([
          { label: "Overview", to: "/munki", activeOptions: { exact: true } },
          { label: "Software", to: "/munki/software" },
          { label: "Packages", to: "/munki/packages" },
          { label: "Distribution Points", to: "/munki/distribution-points" },
          { label: "Client Resources", to: "/munki/client-resources" },
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
          { label: "Overview", to: "/directory", activeOptions: { exact: true } },
          { label: "Users", to: "/directory/users" },
          { label: "Groups", to: "/directory/groups" },
        ]),
      },
      linkOptions({ label: "Labels", to: "/labels", icon: Tag }),
    ],
  },
];
