declare global {
  interface Window {
    __WOODSTAR__?: {
      version?: string;
      csrfToken?: string;
    };
  }
}

export const runtime = {
  version: window.__WOODSTAR__?.version ?? "",
  csrfToken: window.__WOODSTAR__?.csrfToken ?? "",
};
