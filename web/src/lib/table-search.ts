import { z } from "zod";

import { DEFAULT_PAGE_SIZE, MAX_PAGE_SIZE } from "@lib/pagination";

export const TABLE_SEARCH_DEFAULTS = {
  page: 1,
  per_page: DEFAULT_PAGE_SIZE,
} as const;

interface TableSearchSchemaOptions {
  defaultSort?: string;
}

export function createTableSearchSchema(
  sortKeys: readonly [string, ...string[]],
  options: TableSearchSchemaOptions = {},
) {
  const sortTokens = new Set(sortKeys.flatMap((key) => [key, `${key}.asc`, `${key}.desc`]));
  if (options.defaultSort !== undefined && !sortTokens.has(options.defaultSort)) {
    throw new Error(`invalid default table sort: ${options.defaultSort}`);
  }
  const sortSchema = z.string().refine((value) => sortTokens.has(value));
  const sort =
    options.defaultSort === undefined
      ? sortSchema.optional().catch(undefined)
      : sortSchema.default(options.defaultSort).catch(options.defaultSort);

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
    sort,
  });
}
