const SOFTWARE_SOURCE_GROUPS = [
  { value: "apps", label: "App", filterLabel: "App", sources: ["apps"] },
  { value: "homebrew_packages", label: "Homebrew", filterLabel: "Homebrew", sources: ["homebrew_packages"] },
  {
    value: "browser_plugins",
    label: "Browser plugin",
    filterLabel: "Browser plugins",
    sources: ["chrome_extensions", "firefox_addons", "safari_extensions"],
  },
  { value: "npm_packages", label: "npm package", filterLabel: "npm packages", sources: ["npm_packages"] },
  {
    value: "ide_extensions",
    label: "IDE extension",
    filterLabel: "IDE extensions",
    sources: ["vscode_extensions", "jetbrains_plugins"],
  },
  { value: "go_binaries", label: "Go binary", filterLabel: "Go binaries", sources: ["go_binaries"] },
  { value: "python_packages", label: "Python package", filterLabel: "Python packages", sources: ["python_packages"] },
] as const;

export const SOURCE_FILTER_OPTIONS = SOFTWARE_SOURCE_GROUPS.map(({ value, filterLabel }) => ({
  value,
  label: filterLabel,
}));

const SOURCE_FILTER_SOURCES = new Map<string, readonly string[]>(
  SOFTWARE_SOURCE_GROUPS.map(({ value, sources }) => [value, sources]),
);

const SOURCE_LABELS = new Map<string, string>(
  SOFTWARE_SOURCE_GROUPS.flatMap(({ label, sources }) => sources.map((source) => [source, label])),
);

export function expandSoftwareSourceFilters(values: string[]): string[] {
  const expanded = new Set<string>();
  for (const value of values) {
    const sources = SOURCE_FILTER_SOURCES.get(value) ?? [value];
    for (const source of sources) {
      expanded.add(source);
    }
  }
  return Array.from(expanded);
}

export const EXTENSION_FOR_LABELS: Record<string, string> = {
  arc: "Arc",
  brave: "Brave",
  chrome: "Chrome",
  chromium: "Chromium",
  edge: "Edge",
  edge_beta: "Edge Beta",
  firefox: "Firefox",
  opera: "Opera",
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

export function softwareSourceLabel(source: string, extensionFor?: string): string {
  if (!source) return "Unknown";
  const base = SOURCE_LABELS.get(source) ?? "Unknown";
  if (extensionFor) {
    const variant = EXTENSION_FOR_LABELS[extensionFor] ?? "Unknown";
    if (variant) return `${base} (${variant})`;
  }
  return base;
}
