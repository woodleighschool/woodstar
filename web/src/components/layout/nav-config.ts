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
  icon?: LucideIcon;
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
        items: [
          { label: "Software", to: "/munki/software" },
          { label: "Packages", to: "/munki/packages" },
          { label: "Distribution Points", to: "/munki/distribution-points" },
        ],
      },
    ],
  },
  {
    label: "System",
    items: [
      {
        label: "Directory",
        icon: UsersRound,
        items: [
          { label: "Users", to: "/directory/users" },
          { label: "Groups", to: "/directory/groups" },
        ],
      },
      {
        label: "Enrollments",
        icon: KeyRound,
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
