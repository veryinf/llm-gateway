import { request } from '@/lib';
import type { API } from '@/typings';

export interface RequestLogEntry {
  trace_id: string;
  user_id: number;
  api_key_id: number;
  model_name: string;
  is_stream: boolean;
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  request_body: string;
  response_body: string;
  is_detail: boolean;
  status_code: number;
  error_message: string;
  latency_ms: number;
  cost: number;
  ip_address: string;
  user_agent: string;
  created_at: string;
}

export interface RequestChunk {
  id: number;
  trace_id: string;
  chunk_index: number;
  chunk_data: string;
  created_at: string;
}

export interface RequestLogSearchParams {
  page?: number;
  pageSize?: number;
  user_id?: number;
  model?: string;
  status?: string;
  start?: string;
  end?: string;
}

export const requestLogService = {
  async search(params: RequestLogSearchParams): Promise<{ list: RequestLogEntry[]; total: number }> {
    const query = new URLSearchParams();
    if (params.page) query.set('page', String(params.page));
    if (params.pageSize) query.set('pageSize', String(params.pageSize));
    if (params.user_id) query.set('user_id', String(params.user_id));
    if (params.model) query.set('model', params.model);
    if (params.status) query.set('status', params.status);
    if (params.start) query.set('start', params.start);
    if (params.end) query.set('end', params.end);

    const res = await request.get<API.DataSet<RequestLogEntry>>(`/request-logs?${query.toString()}`);
    return { list: res.data.dataSet ?? [], total: res.data.total ?? 0 };
  },

  async fetchByTraceId(traceId: string): Promise<RequestLogEntry[]> {
    const res = await request.get<API.Data<RequestLogEntry[]>>(`/request-logs/${traceId}`);
    return res.data.data ?? [];
  },

  async fetchChunks(traceId: string): Promise<RequestChunk[]> {
    const res = await request.get<API.Data<RequestChunk[]>>(`/request-logs/${traceId}/chunks`);
    return res.data.data ?? [];
  },
};
