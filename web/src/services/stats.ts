import { request } from '@/lib';
import type { API } from '@/typings';

export interface TokenStat {
  user_id: number;
  username: string;
  department: string;
  model_name: string;
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
}

export interface RequestStat {
  date: string;
  request_count: number;
  success_count: number;
  error_count: number;
  avg_latency_ms: number;
}

export interface CostStat {
  date: string;
  model_name: string;
  total_cost: number;
}

export interface BehaviorStat {
  user_id: number;
  username: string;
  department: string;
  model_name: string;
  count: number;
}

export interface DashboardOverview {
  total_requests: number;
  total_tokens: number;
  total_cost: number;
  avg_latency_ms: number;
  success_rate: number;
  active_users: number;
  top_models: { model_name: string; count: number }[];
}

function buildDateParams(start?: string, end?: string) {
  const params = new URLSearchParams();
  if (start) params.set('start', start);
  if (end) params.set('end', end);
  return params.toString();
}

export const statsService = {
  async fetchTokens(start?: string, end?: string): Promise<TokenStat[]> {
    const qs = buildDateParams(start, end);
    const res = await request.get<API.SingleResponse<TokenStat[]>>(`/stats/tokens?${qs}`);
    return res.data.data ?? [];
  },

  async fetchRequests(start?: string, end?: string): Promise<RequestStat[]> {
    const qs = buildDateParams(start, end);
    const res = await request.get<API.SingleResponse<RequestStat[]>>(`/stats/requests?${qs}`);
    return res.data.data ?? [];
  },

  async fetchCosts(start?: string, end?: string): Promise<CostStat[]> {
    const qs = buildDateParams(start, end);
    const res = await request.get<API.SingleResponse<CostStat[]>>(`/stats/costs?${qs}`);
    return res.data.data ?? [];
  },

  async fetchBehavior(start?: string, end?: string): Promise<BehaviorStat[]> {
    const qs = buildDateParams(start, end);
    const res = await request.get<API.SingleResponse<BehaviorStat[]>>(`/stats/behavior?${qs}`);
    return res.data.data ?? [];
  },

  async fetchOverview(start?: string, end?: string): Promise<DashboardOverview> {
    const qs = buildDateParams(start, end);
    const res = await request.get<API.SingleResponse<DashboardOverview>>(`/dashboard/overview?${qs}`);
    return res.data.data!;
  },
};
