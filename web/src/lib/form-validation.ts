import type { ZodError } from "zod";
import { z } from "zod";

export function requiredString(label: string) {
  return z.string().trim().min(1, `${label} is required.`);
}

export function emailAddress(message = "Enter a valid email.") {
  return z.string().trim().pipe(z.email(message));
}

export function uniqueOptions(values: string[]) {
  return Array.from(new Set(values.map((value) => value.trim()).filter(Boolean))).toSorted((a, b) =>
    a.localeCompare(b),
  );
}

export function selectedIDArray(label: string) {
  return z.array(z.number().int(`${label} selection is invalid.`));
}

export function integerRange(label: string, min: number, max?: number) {
  const schema = z
    .number()
    .int(`${label} must be a whole number.`)
    .min(min, `${label} must be at least ${min}.`);
  return max === undefined ? schema : schema.max(max, `${label} must be at most ${max}.`);
}

export function fieldErrors(
  result: { success: true } | { success: false; error: ZodError },
): Partial<Record<string, string>> {
  if (result.success) return {};
  const out: Record<string, string> = {};
  for (const issue of result.error.issues) {
    const key = issue.path[0];
    if (typeof key === "string" && !out[key]) out[key] = issue.message;
  }
  return out;
}

export function firstErrorMessage(errors: readonly unknown[]) {
  for (const error of errors) {
    if (typeof error === "string") return error;
    if (error && typeof error === "object" && "message" in error) {
      const message = (error as { message?: unknown }).message;
      if (typeof message === "string") return message;
    }
  }
  return undefined;
}
