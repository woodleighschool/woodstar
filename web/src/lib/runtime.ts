function metadata(name: string): string | undefined {
  return document.querySelector<HTMLMetaElement>(`meta[name="${name}"]`)?.content || undefined;
}

export const runtime = {
  version: metadata("woodstar-version") ?? "0.0.0-dev",
  serverURL: metadata("woodstar-server-url"),
};
