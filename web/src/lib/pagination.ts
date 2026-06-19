import { nonEmpty } from "@/lib/utils";

export const DEFAULT_PAGE_SIZE = 50;

// Upper bound for "fetch everything" reads (pickers, reorder, label maps).
export const MAX_PAGE_SIZE = 1000;

interface BaseListParams {
  q?: string | null;
  page?: number;
  per_page?: number;
  sort?: string | null;
}

export function baseListParams(params: BaseListParams = {}) {
  return {
    q: nonEmpty(params.q),
    page: params.page ?? 1,
    per_page: params.per_page ?? DEFAULT_PAGE_SIZE,
    sort: nonEmpty(params.sort),
  };
}
