import type { SoftwareTitle } from "@lib/api";
import { countLabel } from "@lib/utils";

type SoftwareSource = SoftwareTitle["source"];

export const SOFTWARE_SOURCE_FILTER_VALUES = [
  "apps",
  "homebrew_packages",
  "browser_plugins",
  "npm_packages",
  "ide_extensions",
  "go_binaries",
  "python_packages",
] as const;

type SoftwareSourceFilterValue = (typeof SOFTWARE_SOURCE_FILTER_VALUES)[number];

const SOFTWARE_SOURCE_GROUPS = {
  apps: { filterLabel: "Applications", sources: ["apps"] },
  homebrew_packages: {
    filterLabel: "Homebrew Packages",
    sources: ["homebrew_packages"],
  },
  browser_plugins: {
    filterLabel: "Browser Extensions",
    sources: ["chrome_extensions", "firefox_addons", "safari_extensions"],
  },
  npm_packages: {
    filterLabel: "npm Packages",
    sources: ["npm_packages"],
  },
  ide_extensions: {
    filterLabel: "IDE Extensions",
    sources: ["vscode_extensions", "jetbrains_plugins"],
  },
  go_binaries: {
    filterLabel: "Go Binaries",
    sources: ["go_binaries"],
  },
  python_packages: {
    filterLabel: "Python Packages",
    sources: ["python_packages"],
  },
} as const satisfies Record<
  SoftwareSourceFilterValue,
  { filterLabel: string; sources: readonly SoftwareSource[] }
>;

export const SOURCE_FILTER_OPTIONS = SOFTWARE_SOURCE_FILTER_VALUES.map((value) => ({
  value,
  label: SOFTWARE_SOURCE_GROUPS[value].filterLabel,
}));

export function expandSoftwareSourceFilters(
  values: readonly SoftwareSourceFilterValue[],
): SoftwareSource[] {
  const expanded = new Set<SoftwareSource>();
  for (const value of values) {
    for (const source of SOFTWARE_SOURCE_GROUPS[value].sources) {
      expanded.add(source);
    }
  }
  return Array.from(expanded);
}

const EXTENSION_FOR_LABELS: Record<string, string> = {
  arc: "Arc",
  brave: "Brave",
  chrome: "Chrome",
  chromium: "Chromium",
  edge: "Edge",
  edge_beta: "Edge Beta",
  firefox: "Firefox",
  opera: "Opera",
  safari: "Safari",
  yandex: "Yandex",
  cursor: "Cursor",
  trae: "Trae",
  vscode: "VS Code",
  vscode_insiders: "VS Code Insiders",
  vscodium: "VSCodium",
  vscodium_insiders: "VSCodium Insiders",
  windsurf: "Windsurf",
  clion: "CLion",
  datagrip: "DataGrip",
  goland: "GoLand",
  intellij_idea: "IntelliJ IDEA",
  intellij_idea_community_edition: "IntelliJ IDEA Community Edition",
  phpstorm: "PhpStorm",
  pycharm: "PyCharm",
  pycharm_community_edition: "PyCharm Community Edition",
  rider: "Rider",
  rubymine: "RubyMine",
  webstorm: "WebStorm",
};

export function softwareSourceLabel(source: SoftwareSource, extensionFor?: string): string {
  const variant = extensionFor ? EXTENSION_FOR_LABELS[extensionFor] : undefined;

  switch (source) {
    case "apps":
      return "Application";
    case "homebrew_packages":
      return "Homebrew Package";
    case "chrome_extensions":
      return `${variant ?? "Chrome"} Extension`;
    case "firefox_addons":
      return `${variant ?? "Firefox"} Add-on`;
    case "safari_extensions":
      return `${variant ?? "Safari"} Extension`;
    case "vscode_extensions":
      return `${variant ?? "VS Code"} Extension`;
    case "jetbrains_plugins":
      return `${variant ?? "JetBrains"} Plugin`;
    case "npm_packages":
      return "npm Package";
    case "go_binaries":
      return "Go Binary";
    case "python_packages":
      return "Python Package";
    default:
      return unreachableSoftwareSource(source);
  }
}

export function versionsSummaryLabel(versions: ReadonlyArray<{ version: string }>): string {
  if (versions.length === 0) return "-";
  if (versions.length === 1) return versions[0].version || "-";
  return countLabel(versions.length, "version");
}

function unreachableSoftwareSource(source: never): never {
  throw new Error(`unsupported software source: ${String(source)}`);
}
