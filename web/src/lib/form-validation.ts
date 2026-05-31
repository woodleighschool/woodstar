import type { ZodError } from "zod";
import { z } from "zod";

export function requiredString(label: string) {
  return z.string().trim().min(1, `${label} is required.`);
}

export function optionalText(value: string) {
  const trimmed = value.trim();
  return trimmed === "" ? undefined : trimmed;
}

export function positiveIntegerArray(label: string) {
  return z.array(z.number().int().positive(`${label} IDs must be positive.`));
}

export function integerRange(label: string, min: number, max?: number) {
  const schema = z.number().int(`${label} must be a whole number.`).min(min, `${label} must be at least ${min}.`);
  return max === undefined ? schema : schema.max(max, `${label} must be at most ${max}.`);
}

export function fieldErrors(result: { success: true } | { success: false; error: ZodError }): Record<string, string> {
  if (result.success) return {};
  const out: Record<string, string> = {};
  for (const issue of result.error.issues) {
    const key = issue.path[0];
    if (typeof key === "string" && !out[key]) out[key] = issue.message;
  }
  return out;
}
