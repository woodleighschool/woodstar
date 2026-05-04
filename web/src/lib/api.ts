import createClient, { type Middleware } from "openapi-fetch";

import type { components, paths } from "@/lib/api-schema";
import { runtime } from "@/lib/runtime";

export type Schemas = components["schemas"];

const MUTATING_METHODS = new Set(["POST", "PUT", "PATCH", "DELETE"]);

const requestMiddleware: Middleware = {
  async onRequest({ request }) {
    request.headers.set("Accept", "application/json");
    if (runtime.csrfToken && MUTATING_METHODS.has(request.method.toUpperCase())) {
      request.headers.set("X-CSRF-Token", runtime.csrfToken);
    }
    return request;
  },
};

export const apiClient = createClient<paths>({
  credentials: "same-origin",
});
apiClient.use(requestMiddleware);

export class ApiError extends Error {
  readonly status: number;
  readonly body?: unknown;

  constructor(status: number, message: string, body?: unknown) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.body = body;
  }
}

interface HumaError {
  title?: string;
  detail?: string;
  errors?: Array<{ message?: string; location?: string }>;
}

function describeError(body: unknown, status: number): string {
  if (body && typeof body === "object") {
    const huma = body as HumaError;
    if (huma.detail) return huma.detail;
    if (huma.title) return huma.title;
    if (huma.errors?.length) {
      return huma.errors
        .map((e) => (e.location ? `${e.location}: ${e.message ?? ""}` : (e.message ?? "")))
        .filter(Boolean)
        .join("; ");
    }
  }
  return `request failed (${status})`;
}

export async function unwrap<T>(pending: Promise<{ data?: T; error?: unknown; response: Response }>): Promise<T> {
  const result = await pending;
  if (result.error !== undefined || !result.response.ok) {
    throw new ApiError(result.response.status, describeError(result.error, result.response.status), result.error);
  }
  return result.data as T;
}
