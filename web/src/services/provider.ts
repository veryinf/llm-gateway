import { request } from '@/lib';
import type { API } from '@/typings';

export interface Provider {
  providerId: number;
  title: string;
  baseUrl: string;
  apiKey?: string;
  supportOpenai: boolean;
  openaiBaseUrl: string;
  supportAnthropic: boolean;
  anthropicBaseUrl: string;
  preferredApi: string;
  isActive: boolean;
  modelCount?: number;
}

export const providerService: API.Service<Provider> = {
  primaryKey: (entity) => entity.providerId,
  title: (entity) => entity.title,

  async search(params) {
    const res = await request.post<API.DataSet<Provider>>('/providers/search', params);
    return res.data;
  },

  async fetch(providerId) {
    const res = await request.post<API.Data<Provider>>('/providers/fetch', { providerId });
    return res.data;
  },

  async add(params) {
    const res = await request.post<API.ResponseStruct>('/providers/add', params);
    return res.data;
  },

  async update(providerId, params) {
    const res = await request.post('/providers/update', { providerId, ...params });
    return res.data;
  },

  async delete(providerId) {
    const res = await request.post('/providers/remove', { providerId });
    return res.data;
  },
};

export async function fetchProviderModels(baseUrl: string, apiKey: string, apiType: string): Promise<{ id: string }[]> {
  const res = await request.post<API.DataSet<{ id: string }>>('/providers/fetch-models', {
    baseUrl,
    apiKey,
    apiType,
  });
  return res.data.dataSet ?? [];
}

export async function batchImportProviderModels(providerId: number, modelNames: string[]): Promise<{ created: number; skipped: number }> {
  const res = await request.post<API.Data<{ created: number; skipped: number }>>(
    '/providers/batch-import-models',
    { providerId, modelNames },
  );
  return res.data.data!;
}
