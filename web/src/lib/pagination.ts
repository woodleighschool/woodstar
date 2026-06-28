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

interface BaseListOptions {
  defaultPerPage?: number;
}

export function baseListParams(params: BaseListParams = {}, options: BaseListOptions = {}) {
  return {
    q: nonEmpty(params.q),
    page: params.page ?? 1,
    per_page: params.per_page ?? options.defaultPerPage ?? DEFAULT_PAGE_SIZE,
    sort: nonEmpty(params.sort),
  };
}
