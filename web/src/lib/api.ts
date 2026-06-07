import { client } from "@/lib/api-client/client.gen";
import type * as API from "@/lib/api-client/types.gen";

export type Page<T> = {
  items: T[] | null;
  count: number;
};

export type Account = API.Account;
export type AccountMutation = API.AccountMutation;
export type AgentSecret = API.AgentSecret;
export type AgentSecretCreate = API.AgentSecretCreate;
export type AgentSecretMutation = API.AgentSecretMutation;
export type BundleReference = API.BundleReference;
export type CertificateReference = API.CertificateReference;
export type Check = API.OsqueryCheck;
export type CheckHostStatus = API.CheckHostStatus;
export type CheckMutation = API.OsqueryCheckMutation;
export type Configuration = API.SantaConfiguration;
export type ConfigurationMutation = API.ConfigurationMutation;
export type Department = API.Department;
export type ExecutionEvent = API.ExecutionEvent;
export type FileAccessEvent = API.FileAccessEvent;
export type Group = API.Group;
export type Handle = API.Handle;
export type Host = API.Host;
export type HostDetail = API.HostDetail;
export type HostReport = API.HostReport;
export type HostReportResults = API.HostReportResultsBody;
export type HostSoftwareInstalledVersion = API.HostSoftwareInstalledVersion;
export type HostSoftwareRow = API.HostSoftwareRow;
export type HostSummary = API.HostSummary;
export type HostUserAffinity = API.HostUserAffinity;
export type Label = API.Label;
export type LabelMutation = API.LabelMutation;
export type LabelRef = API.LabelRef;
export type LiveQueryCreate = API.LiveQueryCreateBody;
export type LiveQueryResultEvent = API.LiveQueryResultEvent;
export type LiveQueryTargetCount = API.LiveQueryTargetCountOutputBody;
export type LiveQueryTargetSelection = API.LiveQueryTargetCountBody;
export type LoginInput = API.LoginInputBody;
export type MunkiHostState = API.MunkiState;
export type MunkiArtifact = API.MunkiArtifact;
export type MunkiArtifactMutation = API.MunkiArtifactMutation;
export type MunkiArtifactUpload = API.MunkiArtifactUpload;
export type MunkiArtifactUploadMutation = API.MunkiArtifactUploadMutation;
export type MunkiPackage = API.MunkiPackage;
export type MunkiPackageImportMutation = API.MunkiPackageImportMutation;
export type MunkiPackageMutation = API.MunkiPackageMutation;
export type MunkiPackagePage = API.PageMunkiPackage;
export type MunkiSoftware = API.MunkiSoftware;
export type MunkiSoftwareDetail = API.MunkiSoftwareDetail;
export type MunkiSoftwareMutation = API.MunkiSoftwareMutation;
export type MunkiSoftwarePage = API.PageMunkiSoftware;
export type PathSignatureInformation = API.PathSignatureInformation;
export type Report = API.OsqueryReport;
export type ReportMutation = API.OsqueryReportMutation;
export type ReportResult = API.ReportResult;
export type Rule = API.SantaRule;
export type RuleMutation = API.RuleMutation;
export type RuleReference = API.RuleReference;
export type RuleStatus = API.RuleStatus;
export type RuleReferenceCandidate = API.RuleReferenceCandidate;
export type SantaHostState = API.SantaHostState;
export type Session = API.SessionBody;
export type SetupInput = API.SetupInputBody;
export type SigningIdentityReference = API.SigningIdentityReference;
export type SoftwareReference = API.SoftwareReference;
export type SoftwareInclude = API.MunkiSoftwareInclude;
export type MunkiSoftwareTargets = API.MunkiSoftwareTargets;
export type SoftwarePackageSelector = API.SoftwarePackageSelector;
export type SoftwareTitle = API.SoftwareTitle;
export type SoftwareVersion = API.SoftwareVersion;
export type User = API.User;
export type UserCreate = API.UserCreate;
export type UserMutation = API.UserMutation;

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
