import { Package, Puzzle } from "lucide-react";
import type { CSSProperties } from "react";
import type { SimpleIcon } from "simple-icons";
import { siApple, siGooglechrome } from "simple-icons";

import { cn } from "@/lib/utils";

type IconPath = Pick<SimpleIcon, "path" | "title">;
type SourceIconKind = "apple" | "chrome" | "package" | "puzzle";

interface SourceDefinition {
  icon: SourceIconKind;
  tint?: string;
}

interface SourceStyle {
  box?: CSSProperties;
  iconClassName: string;
}

const SOURCE_DEFINITIONS: Record<string, SourceDefinition> = {
  apps: {
    icon: "apple",
  },
  homebrew_packages: {
    icon: "package",
    tint: "#d97706",
  },
  npm_packages: {
    icon: "package",
    tint: "#ef4444",
  },
  python_packages: {
    icon: "package",
    tint: "#3b82f6",
  },
  go_binaries: {
    icon: "package",
    tint: "#06b6d4",
  },
  pkg_packages: {
    icon: "package",
    tint: "#94a3b8",
  },
  chrome_extensions: {
    icon: "chrome",
    tint: "#4285f4",
  },
  firefox_addons: {
    icon: "puzzle",
    tint: "#f59e0b",
  },
  safari_extensions: {
    icon: "apple",
  },
  vscode_extensions: {
    icon: "puzzle",
    tint: "#0ea5e9",
  },
  jetbrains_plugins: {
    icon: "puzzle",
    tint: "#d946ef",
  },
  atom_packages: {
    icon: "puzzle",
    tint: "#78716c",
  },
};

const DEFAULT_SOURCE: SourceDefinition = {
  icon: "package",
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
  const definition = SOURCE_DEFINITIONS[source] ?? DEFAULT_SOURCE;
  const sizes = SIZE_CLASS[size];
  const style = sourceStyle(definition.tint);

  return (
    <span
      className={cn("bg-muted/40 inline-flex shrink-0 items-center justify-center border", sizes.box, className)}
      style={style.box}
    >
      {renderSourceIcon(definition.icon, cn(style.iconClassName, sizes.icon))}
    </span>
  );
}

function renderSourceIcon(kind: SourceIconKind, className: string) {
  switch (kind) {
    case "apple":
    case "chrome":
      return <SimpleIconGlyph icon={SOFTWARE_BRAND_ICONS[kind]} className={className} />;
    case "package":
      return <Package className={className} />;
    case "puzzle":
      return <Puzzle className={className} />;
  }
}

const SOFTWARE_BRAND_ICONS = {
  apple: siApple,
  chrome: siGooglechrome,
} as const;

function SimpleIconGlyph({ icon, className }: { icon: IconPath; className: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" className={cn("size-4", className)}>
      <path d={icon.path} />
    </svg>
  );
}

function sourceStyle(color?: string): SourceStyle {
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
