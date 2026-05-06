export const SOURCE_LABELS: Record<string, string> = {
  apps: "App",
  homebrew_packages: "Homebrew",
  chrome_extensions: "Browser plugin",
  firefox_addons: "Browser plugin",
  safari_extensions: "Browser plugin",
  npm_packages: "npm package",
  vscode_extensions: "IDE extension",
  jetbrains_plugins: "IDE extension",
  go_binaries: "Go binary",
  python_packages: "Python package",
};

export const SOURCE_FILTER_OPTIONS = Object.entries(SOURCE_LABELS).map(([value, label]) => ({ value, label }));

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
  const base = SOURCE_LABELS[source] ?? "Unknown";
  if (extensionFor) {
    const variant = EXTENSION_FOR_LABELS[extensionFor] ?? "Unknown";
    if (variant) return `${base} (${variant})`;
  }
  return base;
}
