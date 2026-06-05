import { nonEmpty } from "@/lib/utils";

export interface MunkiListParams {
  q?: string;
  page_index?: number;
  page_size?: number;
  sort?: string;
}

export interface MunkiScopedListParams extends MunkiListParams {
  software_id?: number;
}

export function queryParams(params: MunkiListParams) {
  return {
    q: nonEmpty(params.q),
    page_index: params.page_index ?? 0,
    page_size: params.page_size ?? 50,
    sort: nonEmpty(params.sort),
  };
}

export function scopedQueryParams(params: MunkiScopedListParams) {
  return {
    ...queryParams(params),
    software_id: params.software_id === 0 ? undefined : params.software_id,
  };
}
