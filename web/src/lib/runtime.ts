declare global {
  interface Window {
    __WOODSTAR__?: {
      version?: string;
    };
  }
}

export const runtime = {
  version: window.__WOODSTAR__?.version ?? "",
};
