const softwareSourceLabels: Record<string, string> = {
  app: "App",
  brew: "Homebrew",
  deb: "Debian package",
  npm: "NPM",
  pip: "Python (pip)",
  rpm: "RPM package",
  vscode_extension: "VS Code extension",
};

export function softwareSourceLabel(source: string): string {
  return softwareSourceLabels[source] ?? source;
}
