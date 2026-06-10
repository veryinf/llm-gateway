import { request } from '@/lib';
import type { API } from '@/typings';

export interface AuditLog {
  id: number;
  trace_id: string;
  user_id: number;
  api_key_id: number;
  model_name: string;
  request_summary: string;
  response_summary: string;
  prompt_tokens: number;
  completion_tokens: number;
  status_code: number;
  error_message: string;
  latency_ms: number;
  cost: number;
  ip_address: string;
  user_agent: string;
  created_at: string;
}

export interface AuditLogPage {
  list: AuditLog[];
  total: number;
  page: number;
  page_size: number;
}

export interface AuditLogSearchParams {
  page?: number;
  pageSize?: number;
  user_id?: number;
  model?: string;
  status?: string;
  start?: string;
  end?: string;
}

export const auditService = {
  async search(params: AuditLogSearchParams): Promise<{ list: AuditLog[]; total: number }> {
    const query = new URLSearchParams();
    if (params.page) query.set('page', String(params.page));
    if (params.pageSize) query.set('pageSize', String(params.pageSize));
    if (params.user_id) query.set('user_id', String(params.user_id));
    if (params.model) query.set('model', params.model);
    if (params.status) query.set('status', params.status);
    if (params.start) query.set('start', params.start);
    if (params.end) query.set('end', params.end);

    const res = await request.get<API.SingleResponse<AuditLogPage>>(`/audit/logs?${query.toString()}`);
    const data = res.data.data;
    return { list: data?.list ?? [], total: data?.total ?? 0 };
  },

  async fetchByTraceId(traceId: string): Promise<AuditLog[]> {
    const res = await request.get<API.SingleResponse<AuditLog[]>>(`/audit/logs/${traceId}`);
    return res.data.data ?? [];
  },
};
