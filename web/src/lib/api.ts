import { client } from "@/lib/api-client/client.gen";
import type { ErrorModel } from "@/lib/api-client/types.gen";

export * from "@/lib/api-client/sdk.gen";
export type * from "@/lib/api-client/types.gen";

client.setConfig({
  credentials: "same-origin",
  querySerializer: { array: { style: "form", explode: false } },
});
client.interceptors.request.use((request) => {
  if (!request.headers.has("Accept")) {
    request.headers.set("Accept", "application/json");
  }
  return request;
});

let unauthorizedHandler: (() => void) | undefined;

client.interceptors.response.use((response) => {
  if (response.status === 401) unauthorizedHandler?.();
  return response;
});

export function setUnauthorizedHandler(handler: () => void): void {
  unauthorizedHandler = handler;
}

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

type ApiResult = Promise<{
  data: unknown;
  error: unknown;
  response?: Response;
}>;

type ResponseData<T extends ApiResult> =
  Awaited<T> extends infer Result
    ? Result extends { data: infer Data; error: undefined }
      ? Data
      : never
    : never;

function describeError(body: unknown, status: number): string {
  if (body && typeof body === "object") {
    const huma = body as ErrorModel;
    if (huma.errors?.length) {
      const details = huma.errors
        .map((e) => (e.location ? `${e.location}: ${e.message ?? ""}` : (e.message ?? "")))
        .filter(Boolean)
        .join("; ");
      if (details) return details;
    }
    if (huma.detail) return huma.detail;
    if (huma.title) return huma.title;
  }
  return `request failed (${status})`;
}

export async function unwrap<T extends ApiResult>(pending: T): Promise<ResponseData<T>> {
  const result = await pending;
  if (result.error !== undefined || !result.response?.ok) {
    const status = result.response?.status ?? 0;
    throw new ApiError(status, describeError(result.error, status), result.error);
  }
  return result.data as ResponseData<T>;
}

export async function nullOn404<T extends ApiResult>(pending: T): Promise<ResponseData<T> | null> {
  try {
    return await unwrap(pending);
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) return null;
    throw error;
  }
}
