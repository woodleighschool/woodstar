import { type ClassValue, clsx } from "clsx";
import { formatDistanceToNow } from "date-fns";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}

// Collapses missing/empty/whitespace-only strings to undefined. Used when shaping
// query params so empty filters don't poison cache keys or hit the API.
export function nonEmpty(value: string | null | undefined): string | undefined {
  if (value === null || value === undefined) return undefined;
  const trimmed = value.trim();
  return trimmed === "" ? undefined : trimmed;
}

// Reads a FormData entry as a string. FormData.get returns File | string | null;
// for our text-only forms anything else is a programmer error and we return "".
export function formString(form: FormData, key: string): string {
  const value = form.get(key);
  return typeof value === "string" ? value : "";
}

export function isOneOf<const Values extends readonly string[]>(
  value: unknown,
  values: Values,
): value is Values[number] {
  return typeof value === "string" && values.some((candidate) => candidate === value);
}

export function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

export function assertNever(value: never): never {
  throw new Error(`Unexpected value: ${String(value)}`);
}

export function formatRelative(input: string | number | Date | null | undefined): string {
  if (input === null || input === undefined) return "-";
  const date = input instanceof Date ? input : new Date(input);
  if (Number.isNaN(date.getTime())) return "-";
  return formatDistanceToNow(date, { addSuffix: true });
}

export function formatDate(
  input: string | number | Date | null | undefined,
  opts: Intl.DateTimeFormatOptions = {},
): string {
  if (input === null || input === undefined) return "";
  const date = input instanceof Date ? input : new Date(input);
  if (Number.isNaN(date.getTime())) return "";
  return new Intl.DateTimeFormat("en-US", {
    month: opts.month ?? "long",
    day: opts.day ?? "numeric",
    year: opts.year ?? "numeric",
    ...opts,
  }).format(date);
}

export function formatDateTime(input: string | number | Date | null | undefined): string {
  if (input === null || input === undefined) return "-";
  const date = input instanceof Date ? input : new Date(input);
  if (Number.isNaN(date.getTime())) return "-";
  return date.toLocaleString();
}

export function formatInterval(seconds: number): string {
  if (seconds <= 0) return "never";
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (days >= 7 && seconds % 604800 === 0)
    return `${seconds / 604800} week${seconds === 604800 ? "" : "s"}`;
  if (days > 0 && seconds % 86400 === 0) return `${days} day${days === 1 ? "" : "s"}`;
  if (hours > 0 && seconds % 3600 === 0) return `${hours} hour${hours === 1 ? "" : "s"}`;
  if (minutes > 0 && seconds % 60 === 0) return `${minutes} minute${minutes === 1 ? "" : "s"}`;
  return `${seconds} seconds`;
}
