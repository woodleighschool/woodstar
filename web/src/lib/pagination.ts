import { z } from "zod";

export const DEFAULT_PAGE_SIZE = 50;
export const MAX_PAGE_SIZE = 1000;
export const PAGE_SIZE_OPTIONS = [25, 50, 100, 200, 500, MAX_PAGE_SIZE] as const;

// Shared URL contract for every paginated list route. Spread into route schemas
// that add their own filters: z.object({ ...tableSearchSchema.shape, status: ... }).
export const tableSearchSchema = z.object({
  q: z.string().optional(),
  page_index: z.coerce.number().int().min(0).optional(),
  page_size: z.coerce.number().int().min(10).max(MAX_PAGE_SIZE).optional(),
  sort: z.string().optional(),
});
