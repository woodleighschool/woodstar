declare global {
  interface Window {
    __WOODSTAR__?: {
      version?: string;
      server_url?: string;
    };
  }
}

export const runtime = {
  version: window.__WOODSTAR__?.version ?? "0.0.0-dev",
  serverURL: window.__WOODSTAR__?.server_url,
};
