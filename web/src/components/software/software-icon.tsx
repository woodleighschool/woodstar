import { Package, Puzzle } from "lucide-react";
import type { ElementType } from "react";
import { useEffect, useState } from "react";
import { siApple, siGooglechrome, type SimpleIcon as SimpleIconData } from "simple-icons";

import { cn } from "@/lib/utils";

interface SoftwareIconDefinition {
  icon: ElementType<{ className?: string }>;
}

const SOURCE_ICONS: Record<string, SoftwareIconDefinition> = {
  apps: {
    icon: AppleIcon,
  },
  homebrew_packages: {
    icon: Package,
  },
  npm_packages: {
    icon: Package,
  },
  python_packages: {
    icon: Package,
  },
  go_binaries: {
    icon: Package,
  },
  pkg_packages: {
    icon: Package,
  },
  chrome_extensions: {
    icon: ChromeIcon,
  },
  firefox_addons: {
    icon: Puzzle,
  },
  safari_extensions: {
    icon: AppleIcon,
  },
  vscode_extensions: {
    icon: Puzzle,
  },
  jetbrains_plugins: {
    icon: Puzzle,
  },
  atom_packages: {
    icon: Puzzle,
  },
};

const DEFAULT_ICON: SoftwareIconDefinition = {
  icon: Package,
};

const SIZE_CLASS = {
  sm: { box: "size-6 rounded-sm", icon: "size-4", image: "size-6" },
  md: { box: "size-9 rounded-md", icon: "size-5", image: "size-9" },
  lg: { box: "size-20 rounded-lg", icon: "size-8", image: "size-20" },
} as const;

export type SoftwareIconSize = keyof typeof SIZE_CLASS;

interface SoftwareIconProps extends SoftwareIconDefinition {
  size?: SoftwareIconSize;
  className?: string;
}

export function SoftwareIcon({ icon: Icon, size = "sm", className }: SoftwareIconProps) {
  const sizes = SIZE_CLASS[size];

  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center justify-center border bg-muted/40 text-muted-foreground",
        sizes.box,
        className,
      )}
    >
      <Icon className={sizes.icon} />
    </span>
  );
}

interface SoftwareArtworkProps {
  src?: string;
  size?: SoftwareIconSize;
  className?: string;
  loading?: "eager" | "lazy";
}

export function SoftwareArtwork({
  src,
  size = "sm",
  className,
  loading = "lazy",
}: SoftwareArtworkProps) {
  const [failedSrc, setFailedSrc] = useState<string>();

  useEffect(() => {
    setFailedSrc(undefined);
  }, [src]);

  if (!src || failedSrc === src) {
    return <SoftwareIcon {...DEFAULT_ICON} size={size} className={className} />;
  }

  return (
    <img
      src={src}
      alt=""
      className={cn("block shrink-0 object-fill", SIZE_CLASS[size].image, className)}
      loading={loading}
      onError={() => setFailedSrc(src)}
    />
  );
}

export function softwareIconProps(source?: string): SoftwareIconDefinition {
  return SOURCE_ICONS[source ?? ""] ?? DEFAULT_ICON;
}

function AppleIcon({ className }: { className?: string }) {
  return <SimpleIcon icon={siApple} className={className} />;
}

function ChromeIcon({ className }: { className?: string }) {
  return <SimpleIcon icon={siGooglechrome} className={className} />;
}

function SimpleIcon({ icon, className }: { icon: SimpleIconData; className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" className={className}>
      <path d={icon.path} />
    </svg>
  );
}
