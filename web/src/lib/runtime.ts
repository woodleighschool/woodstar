declare global {
  interface Window {
    __WOODSTAR__?: {
      version?: string;
    };
  }
}

export const runtime = {
  version: window.__WOODSTAR__?.version ?? "0.0.0-dev",
};
