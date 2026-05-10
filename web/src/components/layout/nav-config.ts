import {
  ClipboardCheck,
  FileBarChart2,
  Package,
  ServerCog,
  Settings as SettingsIcon,
  Tag,
  Users as UsersIcon,
  type LucideIcon,
} from "lucide-react";

export interface NavItem {
  label: string;
  to: string;
  icon: LucideIcon;
  adminOnly?: boolean;
}

export interface NavSection {
  label?: string;
  items: NavItem[];
}

export const navSections: NavSection[] = [
  {
    label: "Inventory",
    items: [
      { label: "Hosts", to: "/hosts", icon: ServerCog },
      { label: "Software", to: "/software", icon: Package },
      { label: "Reports", to: "/reports", icon: FileBarChart2 },
      { label: "Checks", to: "/checks", icon: ClipboardCheck },
    ],
  },
  {
    label: "System",
    items: [
      { label: "Labels", to: "/labels", icon: Tag },
      { label: "Users", to: "/users", icon: UsersIcon, adminOnly: true },
      { label: "Settings", to: "/settings", icon: SettingsIcon },
    ],
  },
];
