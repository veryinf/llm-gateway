import { request } from '@/lib';
import type { API } from '@/typings';

export interface RequestLog {
  traceId: string;
  userId: number;
  apiKeyId: number;
  modelName: string;
  summary: string;
  isStream: boolean;
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
  isDetail: boolean;
  statusCode: number;
  errorMessage: string;
  latencyMs: number;
  cost: number;
  ipAddress: string;
  userAgent: string;
  createdAt: string;
}

export interface RequestDetail {
  traceId: string;
  requestBody: string;
  responseBody: string;
}

export interface RequestChunk {
  chunkId: number;
  traceId: string;
  index: number;
  data: string;
  createdAt: string;
}

export const requestLogService = {
  async search(params: API.SearchParams): Promise<API.DataSet<RequestLog>> {
    const res = await request.post<API.DataSet<RequestLog>>('/request-logs/search', params);
    return res.data;
  },
};

export const requestDetailService = {
  async fetch(traceId: string): Promise<RequestDetail> {
    const res = await request.post<API.Data<RequestDetail>>('/request-logs/detail', { traceId });
    return res.data.data!;
  },
};

export const requestChunkService = {
  async fetch(traceId: string): Promise<RequestChunk[]> {
    const res = await request.post<API.Data<RequestChunk[]>>('/request-logs/chunks', { traceId });
    return res.data.data ?? [];
  },
};
