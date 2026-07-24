import { z } from "zod";

import { DEFAULT_PAGE_SIZE, MAX_PAGE_SIZE } from "@/lib/pagination";

export const LIST_SEARCH_DEFAULTS = {
  page: 1,
  per_page: DEFAULT_PAGE_SIZE,
} as const;

export function createListSearchSchema(sortKeys: readonly [string, ...string[]]) {
  const sortTokens = new Set(sortKeys.flatMap((key) => [key, `${key}.asc`, `${key}.desc`]));

  return z.object({
    q: z.string().trim().min(1).optional().catch(undefined),
    page: z.coerce.number().int().min(1).default(1).catch(1),
    per_page: z.coerce
      .number()
      .int()
      .min(1)
      .max(MAX_PAGE_SIZE)
      .default(DEFAULT_PAGE_SIZE)
      .catch(DEFAULT_PAGE_SIZE),
    sort: z
      .string()
      .refine((value) => sortTokens.has(value))
      .optional()
      .catch(undefined),
  });
}
