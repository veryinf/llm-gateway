import { request } from '@/lib';
import type { API } from '@/typings';

// ─── 通用查询层 ───

export interface StatsQueryFilter {
  field: string;
  op: 'eq' | 'ne' | 'gt' | 'gte' | 'lt' | 'lte' | 'in' | 'between' | 'like';
  value: unknown;
}

export interface StatsQuerySort {
  field: string;
  dir: 'asc' | 'desc';
}

export interface StatsQueryRequest {
  dimensions: string[];
  measures: string[];
  filters?: StatsQueryFilter[];
  sort?: StatsQuerySort[];
  page?: number;
  size?: number;
}

export interface StatsQueryResponse {
  rows: Record<string, unknown>[];
  total: number;
}

export const statsQueryService = {
  async query(req: StatsQueryRequest): Promise<StatsQueryResponse> {
    const res = await request.post<API.DataSet<Record<string, unknown>>>('/stats/query', req);
    return { rows: res.data.dataSet ?? [], total: res.data.total ?? 0 };
  },
};