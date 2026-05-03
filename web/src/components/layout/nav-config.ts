import {
  ClipboardCheck,
  FileBarChart2,
  Layers,
  Package,
  ServerCog,
  Settings as SettingsIcon,
  Shield,
  Tag,
  type LucideIcon,
} from "lucide-react";

export interface NavItem {
  label: string;
  to: string;
  icon: LucideIcon;
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
      { label: "Labels", to: "/labels", icon: Tag },
      { label: "Reports", to: "/reports", icon: FileBarChart2 },
      { label: "Checks", to: "/checks", icon: ClipboardCheck },
    ],
  },
  {
    items: [
      { label: "Santa", to: "/santa", icon: Shield },
      { label: "Munki", to: "/munki", icon: Layers },
    ],
  },
  {
    items: [{ label: "Settings", to: "/settings", icon: SettingsIcon }],
  },
];
