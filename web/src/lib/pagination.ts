import { nonEmpty } from "@/lib/utils";

export const DEFAULT_PAGE_SIZE = 50;

// Upper bound for "fetch everything" reads (pickers, reorder, label maps).
export const MAX_PAGE_SIZE = 1000;

export function normalizePage(value: number | undefined): number {
  return Number.isSafeInteger(value) && value !== undefined && value > 0 ? value : 1;
}

export function normalizePageSize(value: number | undefined, fallback = DEFAULT_PAGE_SIZE): number {
  return Number.isSafeInteger(value) && value !== undefined && value > 0 && value <= MAX_PAGE_SIZE
    ? value
    : fallback;
}

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
  const defaultPerPage = normalizePageSize(options.defaultPerPage);
  return {
    q: nonEmpty(params.q),
    page: normalizePage(params.page),
    per_page: normalizePageSize(params.per_page, defaultPerPage),
    sort: nonEmpty(params.sort),
  };
}
