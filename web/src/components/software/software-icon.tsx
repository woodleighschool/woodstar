import { Package, Puzzle } from "lucide-react";
import type { CSSProperties, ElementType } from "react";
import { useEffect, useState } from "react";
import { siApple, siGooglechrome, type SimpleIcon as SimpleIconData } from "simple-icons";

import { cn } from "@/lib/utils";

interface SoftwareIconDefinition {
  icon: ElementType<{ className?: string }>;
  color?: string;
}

interface SoftwareIconStyle {
  box?: CSSProperties;
  iconClassName: string;
}

// Intentional brand/source colors, not theme tokens.
const SOURCE_COLORS = {
  atom: "#78716c",
  firefox: "#f59e0b",
  go: "#06b6d4",
  homebrew: "#d97706",
  jetbrains: "#d946ef",
  npm: "#ef4444",
  pkg: "#94a3b8",
  python: "#3b82f6",
  vscode: "#0ea5e9",
} as const;

const SOURCE_ICONS: Record<string, SoftwareIconDefinition> = {
  apps: {
    icon: AppleIcon,
  },
  homebrew_packages: {
    icon: Package,
    color: SOURCE_COLORS.homebrew,
  },
  npm_packages: {
    icon: Package,
    color: SOURCE_COLORS.npm,
  },
  python_packages: {
    icon: Package,
    color: SOURCE_COLORS.python,
  },
  go_binaries: {
    icon: Package,
    color: SOURCE_COLORS.go,
  },
  pkg_packages: {
    icon: Package,
    color: SOURCE_COLORS.pkg,
  },
  chrome_extensions: {
    icon: ChromeIcon,
    color: `#${siGooglechrome.hex}`,
  },
  firefox_addons: {
    icon: Puzzle,
    color: SOURCE_COLORS.firefox,
  },
  safari_extensions: {
    icon: AppleIcon,
  },
  vscode_extensions: {
    icon: Puzzle,
    color: SOURCE_COLORS.vscode,
  },
  jetbrains_plugins: {
    icon: Puzzle,
    color: SOURCE_COLORS.jetbrains,
  },
  atom_packages: {
    icon: Puzzle,
    color: SOURCE_COLORS.atom,
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

export function SoftwareIcon({ icon: Icon, color, size = "sm", className }: SoftwareIconProps) {
  const sizes = SIZE_CLASS[size];
  const style = softwareIconStyle(color);

  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center justify-center border bg-muted/40",
        sizes.box,
        className,
      )}
      style={style.box}
    >
      <Icon className={cn(style.iconClassName, sizes.icon)} />
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

function softwareIconStyle(color?: string): SoftwareIconStyle {
  if (!color) {
    return { iconClassName: "text-muted-foreground" };
  }

  return {
    box: {
      color,
      backgroundColor: withAlpha(color, 0.14),
      borderColor: withAlpha(color, 0.34),
    },
    iconClassName: "",
  };
}

function withAlpha(color: string, alpha: number) {
  const hex = color.replace("#", "");
  const value = Number.parseInt(hex, 16);

  if (hex.length !== 6 || Number.isNaN(value)) {
    return color;
  }

  const red = (value >> 16) & 255;
  const green = (value >> 8) & 255;
  const blue = value & 255;

  return `rgba(${red}, ${green}, ${blue}, ${alpha})`;
}
