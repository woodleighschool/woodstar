import {
  ClipboardCheck,
  Database,
  FileBarChart2,
  Package,
  PackageSearch,
  ServerCog,
  Settings as SettingsIcon,
  ShieldCheck,
  Tag,
  Users as UsersIcon,
  Warehouse,
  type LucideIcon,
} from "lucide-react";

export interface NavItem {
  label: string;
  to: string;
  icon: LucideIcon;
  adminOnly?: boolean;
}

export interface NavMenu {
  label: string;
  icon: LucideIcon;
  items: NavItem[];
  collapsible?: boolean;
  placeholder?: boolean;
}

export const navSections: NavMenu[] = [
  {
    label: "Inventory",
    icon: Warehouse,
    collapsible: false,
    items: [
      { label: "Hosts", to: "/hosts", icon: ServerCog },
      { label: "Software", to: "/software", icon: Package },
    ],
  },
  {
    label: "Osquery",
    icon: Database,
    collapsible: true,
    items: [
      { label: "Reports", to: "/reports", icon: FileBarChart2 },
      { label: "Checks", to: "/checks", icon: ClipboardCheck },
    ],
  },
  {
    label: "Santa",
    icon: ShieldCheck,
    collapsible: true,
    items: [],
    placeholder: true,
  },
  {
    label: "Munki",
    icon: PackageSearch,
    collapsible: true,
    items: [],
    placeholder: true,
  },
  {
    label: "Settings",
    icon: SettingsIcon,
    collapsible: false,
    items: [
      { label: "Labels", to: "/labels", icon: Tag },
      { label: "Users", to: "/users", icon: UsersIcon, adminOnly: true },
      { label: "Settings", to: "/settings", icon: SettingsIcon },
    ],
  },
];
