import {
  Database,
  KeyRound,
  Package,
  PackageSearch,
  ServerCog,
  ShieldCheck,
  Tag,
  UsersRound,
  type LucideIcon,
} from "lucide-react";

export interface NavItem {
  label: string;
  to?: string;
  icon?: LucideIcon;
  adminOnly?: boolean;
  disabled?: boolean;
  items?: NavItem[];
}

export interface NavMenu {
  label: string;
  items: NavItem[];
}

export const navSections: NavMenu[] = [
  {
    label: "Inventory",
    items: [
      { label: "Hosts", to: "/hosts", icon: ServerCog },
      { label: "Software", to: "/software", icon: Package },
    ],
  },
  {
    label: "Integrations",
    items: [
      {
        label: "Osquery",
        icon: Database,
        items: [
          { label: "Reports", to: "/osquery/reports" },
          { label: "Checks", to: "/osquery/checks" },
        ],
      },
      {
        label: "Santa",
        icon: ShieldCheck,
        items: [
          { label: "Configurations", to: "/santa/configurations" },
          { label: "Rules", to: "/santa/rules" },
          { label: "Events", to: "/santa/events" },
        ],
      },
      {
        label: "Munki",
        icon: PackageSearch,
        adminOnly: true,
        items: [{ label: "Software", to: "/munki/software-titles" }],
      },
    ],
  },
  {
    label: "System",
    items: [
      {
        label: "Directory",
        icon: UsersRound,
        adminOnly: true,
        items: [
          { label: "Users", to: "/directory/users" },
          { label: "Groups", to: "/directory/groups" },
        ],
      },
      {
        label: "Enrollments",
        icon: KeyRound,
        adminOnly: true,
        items: [
          { label: "Orbit", to: "/enrollments/orbit" },
          { label: "Munki", to: "/enrollments/munki" },
          { label: "Santa", to: "/enrollments/santa" },
        ],
      },
      { label: "Labels", to: "/labels", icon: Tag },
    ],
  },
];
