import { Package, Puzzle } from "lucide-react";
import type { ReactNode } from "react";

import { SOFTWARE_BRAND_ICONS } from "@/components/platform/platform-icon-data";
import { PlatformIcon, SimpleIconGlyph } from "@/components/platform/platform-icons";
import { cn } from "@/lib/utils";

type IconRenderer = (className: string) => ReactNode;

const simpleBrandIcon =
  (icon: (typeof SOFTWARE_BRAND_ICONS)[keyof typeof SOFTWARE_BRAND_ICONS]): IconRenderer =>
  (className) => <SimpleIconGlyph icon={icon} className={className} />;

const lucideIcon =
  (Icon: typeof Package): IconRenderer =>
  (className) => <Icon className={className} aria-hidden />;

const ICON_BY_SOURCE: Record<string, IconRenderer> = {
  apps: simpleBrandIcon(SOFTWARE_BRAND_ICONS.apple),
  ios_apps: simpleBrandIcon(SOFTWARE_BRAND_ICONS.apple),
  ipados_apps: simpleBrandIcon(SOFTWARE_BRAND_ICONS.apple),
  programs: (className) => <PlatformIcon platform="windows" className={className} />,
  android_apps: simpleBrandIcon(SOFTWARE_BRAND_ICONS.android),
  homebrew_packages: lucideIcon(Package),
  npm_packages: lucideIcon(Package),
  python_packages: lucideIcon(Package),
  go_binaries: lucideIcon(Package),
  pkg_packages: lucideIcon(Package),
  deb_packages: lucideIcon(Package),
  rpm_packages: lucideIcon(Package),
  chocolatey_packages: lucideIcon(Package),
  apt_sources: lucideIcon(Package),
  yum_sources: lucideIcon(Package),
  chrome_extensions: simpleBrandIcon(SOFTWARE_BRAND_ICONS.chrome),
  firefox_addons: lucideIcon(Puzzle),
  safari_extensions: simpleBrandIcon(SOFTWARE_BRAND_ICONS.apple),
  ie_extensions: (className) => <PlatformIcon platform="windows" className={className} />,
  vscode_extensions: lucideIcon(Puzzle),
  jetbrains_plugins: lucideIcon(Puzzle),
  atom_packages: lucideIcon(Puzzle),
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
  const icon = ICON_BY_SOURCE[source] ?? lucideIcon(Package);
  const sizes = SIZE_CLASS[size];
  return (
    <span className={cn("bg-muted/40 inline-flex shrink-0 items-center justify-center border", sizes.box, className)}>
      {icon(cn("text-muted-foreground", sizes.icon))}
    </span>
  );
}
