import { Apple, MonitorCog, Package, Puzzle } from "lucide-react";
import type { ComponentType, SVGProps } from "react";

import { cn } from "@/lib/utils";

const ICON_BY_SOURCE: Record<string, ComponentType<SVGProps<SVGSVGElement>>> = {
  apps: Apple,
  ios_apps: Apple,
  ipados_apps: Apple,
  programs: MonitorCog,
  android_apps: MonitorCog,
  homebrew_packages: Package,
  npm_packages: Package,
  python_packages: Package,
  go_binaries: Package,
  pkg_packages: Package,
  deb_packages: Package,
  rpm_packages: Package,
  chocolatey_packages: Package,
  apt_sources: Package,
  yum_sources: Package,
  chrome_extensions: Puzzle,
  firefox_addons: Puzzle,
  safari_extensions: Puzzle,
  ie_extensions: Puzzle,
  vscode_extensions: Puzzle,
  jetbrains_plugins: Puzzle,
  atom_packages: Puzzle,
};

const SIZE_CLASS = {
  sm: { box: "size-6 rounded-sm", icon: "size-4" },
  md: { box: "size-9 rounded-md", icon: "size-5" },
  lg: { box: "size-20 rounded-lg", icon: "size-8" },
} as const;

export type SoftwareIconSize = keyof typeof SIZE_CLASS;

interface SoftwareIconProps {
  source?: string;
  size?: SoftwareIconSize;
  className?: string;
}

export function SoftwareIcon({ source = "", size = "sm", className }: SoftwareIconProps) {
  const Icon = ICON_BY_SOURCE[source] ?? Package;
  const sizes = SIZE_CLASS[size];
  return (
    <span
      className={cn(
        "bg-muted/40 inline-flex shrink-0 items-center justify-center border",
        sizes.box,
        className,
      )}
    >
      <Icon className={cn("text-muted-foreground", sizes.icon)} aria-hidden />
    </span>
  );
}
