import { nonEmpty } from "@lib/utils";

export const DEFAULT_PAGE_SIZE = 50;

// Upper bound for "fetch everything" reads such as exports, pickers, and label maps.
export const MAX_PAGE_SIZE = 1000;

export const PAGE_SIZE_OPTIONS = [25, 50, 100, 200, 500, MAX_PAGE_SIZE] as const;

interface Page<T> {
  count: number;
  items: T[];
}

export async function collectAllPages<T>(
  loadPage: (page: number, perPage: number) => Promise<Page<T>>,
): Promise<T[]> {
  const firstPage = await loadPage(1, MAX_PAGE_SIZE);
  const pageCount = Math.ceil(firstPage.count / MAX_PAGE_SIZE);
  if (pageCount <= 1) return firstPage.items;

  const items = [...firstPage.items];
  for (let page = 2; page <= pageCount; page += 1) {
    const response = await loadPage(page, MAX_PAGE_SIZE);
    items.push(...response.items);
  }
  return items;
}

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
