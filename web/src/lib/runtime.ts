declare global {
  interface Window {
    __WOODSTAR__?: {
      version?: string;
      public_url?: string;
    };
  }
}

export const runtime = {
  version: window.__WOODSTAR__?.version ?? "0.0.0-dev",
  publicURL: window.__WOODSTAR__?.public_url,
};
