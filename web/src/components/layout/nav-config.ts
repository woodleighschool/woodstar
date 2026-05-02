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
  /**
   * Marks navigation items whose page is a shell only — no backend yet.
   * Surfaced in the sidebar as a small "pending" indicator so admins
   * understand which screens are real vs scaffold.
   */
  pending?: boolean;
}

export interface NavSection {
  /** Section title; omit for sections that should render without a heading. */
  label?: string;
  items: NavItem[];
}

export const navSections: NavSection[] = [
  {
    label: "Inventory",
    items: [
      { label: "Hosts", to: "/hosts", icon: ServerCog, pending: true },
      { label: "Software", to: "/software", icon: Package, pending: true },
      { label: "Labels", to: "/labels", icon: Tag, pending: true },
      { label: "Reports", to: "/reports", icon: FileBarChart2, pending: true },
      { label: "Checks", to: "/checks", icon: ClipboardCheck, pending: true },
    ],
  },
  {
    items: [
      { label: "Santa", to: "/santa", icon: Shield, pending: true },
      { label: "Munki", to: "/munki", icon: Layers, pending: true },
    ],
  },
  {
    items: [
      { label: "Settings", to: "/settings", icon: SettingsIcon, pending: true },
    ],
  },
];
