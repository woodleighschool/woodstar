import { linkOptions, type ActiveOptions } from "@tanstack/react-router";
import {
  Database,
  KeyRound,
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
        icon: Database,
        items: linkOptions([
          { label: "Reports", to: "/osquery/reports" },
          { label: "Checks", to: "/osquery/checks" },
        ]),
      },
      {
        label: "Santa",
        icon: ShieldCheck,
        items: linkOptions([
          { label: "Configurations", to: "/santa/configurations" },
          { label: "Rules", to: "/santa/rules" },
          { label: "Events", to: "/santa/events" },
        ]),
      },
      {
        label: "Munki",
        icon: PackageSearch,
        items: linkOptions([
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
      {
        label: "Enrollments",
        icon: KeyRound,
        items: linkOptions([
          { label: "Orbit", to: "/enrollments/orbit" },
          { label: "Munki", to: "/enrollments/munki" },
          { label: "Santa", to: "/enrollments/santa" },
        ]),
      },
      linkOptions({ label: "Labels", to: "/labels", icon: Tag }),
    ],
  },
];
