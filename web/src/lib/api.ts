import { client } from "@/lib/api-client/client.gen";
import type * as API from "@/lib/api-client/types.gen";

export type Schemas = {
  AccountBody: API.AccountBody;
  AccountPutBody: API.AccountPutBody;
  AgentSecret: API.AgentSecret;
  Check: API.Check;
  CheckCreate: API.CheckCreate;
  CheckHostStatus: API.CheckHostStatus;
  Configuration: API.Configuration;
  ConfigurationMutation: API.ConfigurationMutation;
  Department: API.Department;
  DirectoryDepartmentsBody: API.DirectoryDepartmentsBody;
  DirectoryGroupBody: API.DirectoryGroupBody;
  DirectoryGroupsBody: API.DirectoryGroupsBody;
  DirectoryUserBody: API.DirectoryUserBody;
  DirectoryUsersBody: API.DirectoryUsersBody;
  EffectiveRuleStatus: API.EffectiveRuleStatus;
  ExecutionEvent: API.ExecutionEvent;
  FileAccessEvent: API.FileAccessEvent;
  Handle: API.Handle;
  Host: API.Host;
  HostDetailBody: API.HostDetailBody;
  HostReport: API.HostReport;
  HostSoftwareInstalledVersion: API.HostSoftwareInstalledVersion;
  HostSoftwareRow: API.HostSoftwareRow;
  HostState: API.HostState;
  HostSummary: API.HostSummary;
  ItemsBodyCheckHostStatus: API.ItemsBodyCheckHostStatus;
  ItemsBodyHostReport: API.ItemsBodyHostReport;
  ItemsBodyReportResult: API.ItemsBodyReportResult;
  Label: API.Label;
  LabelCreateBody: API.LabelCreateBody;
  LabelMutationBody: API.LabelMutationBody;
  LabelScope: API.LabelScope;
  LiveQueryCreateBody: API.LiveQueryCreateBody;
  LiveQueryResultEvent: API.LiveQueryResultEvent;
  LiveQueryTargetCountBody: API.LiveQueryTargetCountBody;
  LiveQueryTargetCountOutputBody: API.LiveQueryTargetCountOutputBody;
  LoginInputBody: API.LoginInputBody;
  PaginatedBodyCheck: API.PaginatedBodyCheck;
  PaginatedBodyConfiguration: API.PaginatedBodyConfiguration;
  PaginatedBodyEffectiveRuleStatus: API.PaginatedBodyEffectiveRuleStatus;
  PaginatedBodyExecutionEvent: API.PaginatedBodyExecutionEvent;
  PaginatedBodyFileAccessEvent: API.PaginatedBodyFileAccessEvent;
  PaginatedBodyHost: API.PaginatedBodyHost;
  PaginatedBodyHostSoftwareRow: API.PaginatedBodyHostSoftwareRow;
  PaginatedBodyLabel: API.PaginatedBodyLabel;
  PaginatedBodyReport: API.PaginatedBodyReport;
  PaginatedBodyRule: API.PaginatedBodyRule;
  PaginatedBodySoftwareTitle: API.PaginatedBodySoftwareTitle;
  PaginatedBodyUser: API.PaginatedBodyUser;
  PathSignatureInformation: API.PathSignatureInformation;
  Report: API.Report;
  ReportCreate: API.ReportCreate;
  ReportResult: API.ReportResult;
  Rule: API.Rule;
  RuleMutation: API.RuleMutation;
  SessionBody: API.SessionBody;
  SetupInputBody: API.SetupInputBody;
  SoftwareTitle: API.SoftwareTitle;
  SoftwareVersion: API.SoftwareVersion;
  User: API.User;
  UserCreateInputBody: API.UserCreateInputBody;
  UserPutBody: API.UserPutBody;
};

client.setConfig({
  credentials: "same-origin",
  querySerializer: { array: { style: "form", explode: false } },
});
client.interceptors.request.use((request) => {
  request.headers.set("Accept", "application/json");
  return request;
});

type Method = "GET" | "POST" | "PUT" | "PATCH" | "DELETE";

interface RequestOptions {
  body?: unknown;
  params?: {
    path?: Record<string, unknown>;
    query?: Record<string, unknown>;
  };
  signal?: AbortSignal;
}

type APIResponse<T> = Promise<{
  data?: T;
  error?: unknown;
  response?: Response;
}>;

function request<T = unknown>(method: Method, url: string, options: RequestOptions = {}): APIResponse<T> {
  return client.request({
    method,
    url,
    body: options.body,
    path: options.params?.path,
    query: options.params?.query,
    signal: options.signal,
  }) as unknown as APIResponse<T>;
}

export const apiClient = {
  DELETE: <T = unknown>(url: string, options?: RequestOptions) => request<T>("DELETE", url, options),
  GET: <T = unknown>(url: string, options?: RequestOptions) => request<T>("GET", url, options),
  PATCH: <T = unknown>(url: string, options?: RequestOptions) => request<T>("PATCH", url, options),
  POST: <T = unknown>(url: string, options?: RequestOptions) => request<T>("POST", url, options),
  PUT: <T = unknown>(url: string, options?: RequestOptions) => request<T>("PUT", url, options),
};

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

export async function unwrap<T>(pending: APIResponse<T>): Promise<T> {
  const result = await pending;
  if (result.error !== undefined || !result.response?.ok) {
    const status = result.response?.status ?? 0;
    throw new ApiError(status, describeError(result.error, status), result.error);
  }
  return result.data as T;
}
