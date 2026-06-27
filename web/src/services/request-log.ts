import { request } from '@/lib';
import type { API } from '@/typings';

export interface RequestLog {
  traceId: string;
  userId: number;
  apiKeyId: number;
  userModel: string;
  providerModel: string;
  userApiType: 'openai' | 'anthropic';
  providerApiType: 'openai' | 'anthropic';
  passthroughLevel: 'none' | 'user' | 'provider';
  summary: string;
  isStream: boolean;
  promptTokens: number;
  completionTokens: number;
  reasoningTokens: number;
  totalTokens: number;
  cachedTokens: number;
  isDetail: boolean;
  statusCode: number;
  errorMessage: string;
  duration: number;
  ipAddress: string;
  userAgent: string;
  createdAt: string;
}

export interface RequestDetail {
  traceId: string;
  request: string;
  requestRaw: string;
  response: string;
  responseRaw: string;
  reasoning: string;
}

export type ChunkType = 'message' | 'reasoning' | 'usage' | 'done';

export interface RequestChunk {
  chunkId: number;
  traceId: string;
  index: number;
  type: ChunkType;
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
