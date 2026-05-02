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

export interface FetchJsonOptions extends Omit<RequestInit, "body"> {
  body?: unknown;
}

export async function fetchJson<T = unknown>(
  path: string,
  { body, headers, ...init }: FetchJsonOptions = {},
): Promise<T> {
  const finalHeaders = new Headers(headers);
  let serializedBody: BodyInit | undefined;

  if (body !== undefined && body !== null) {
    if (
      body instanceof FormData ||
      body instanceof URLSearchParams ||
      body instanceof Blob ||
      typeof body === "string"
    ) {
      serializedBody = body as BodyInit;
    } else {
      serializedBody = JSON.stringify(body);
      if (!finalHeaders.has("Content-Type")) {
        finalHeaders.set("Content-Type", "application/json");
      }
    }
  }

  if (!finalHeaders.has("Accept")) {
    finalHeaders.set("Accept", "application/json");
  }

  const response = await fetch(path, {
    ...init,
    headers: finalHeaders,
    body: serializedBody,
    credentials: init.credentials ?? "same-origin",
  });

  const contentType = response.headers.get("content-type") ?? "";
  const isJson = contentType.includes("application/json");

  if (!response.ok) {
    const errorBody = isJson ? await response.json().catch(() => null) : null;
    const message =
      (errorBody && typeof errorBody === "object" && "message" in errorBody
        ? String((errorBody as { message: unknown }).message)
        : null) ?? response.statusText ?? "request failed";
    throw new ApiError(response.status, message, errorBody);
  }

  if (response.status === 204 || !isJson) {
    return undefined as T;
  }

  return (await response.json()) as T;
}
