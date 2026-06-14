import { request } from '@/lib';
import type { API } from '@/typings';

export interface Provider {
  id: number;
  name: string;
  base_url: string;
  api_key?: string;
  support_openai: boolean;
  openai_base_url: string;
  support_anthropic: boolean;
  anthropic_base_url: string;
  preferred_api: string;
  is_active: boolean;
  model_count?: number;
  created_at: string;
  updated_at: string;
}

export const providerService: API.Service<Provider> = {
  primaryKey: (entity) => entity.id,
  title: (entity) => entity.name,

  async search() {
    const res = await request.get<API.DataSet<Provider>>('/admin/providers');
    return res.data;
  },

  async fetch(id) {
    const res = await request.get<API.Data<Provider>>('/admin/providers');
    const list = (res.data.data as unknown as Provider[]) ?? [];
    const provider = list.find((p) => p.id === id);
    return { errCode: 0, errMsg: 'ok', data: provider };
  },

  async add(params) {
    await request.post('/admin/providers', params);
    return { errCode: 0, errMsg: 'ok' };
  },

  async update(id, params) {
    const { models, ...providerFields } = params as any;
    await request.put(`/admin/providers/${id}`, {
      provider: providerFields,
      models: models ?? undefined,
    });
    return { errCode: 0, errMsg: 'ok' };
  },

  async delete(id) {
    await request.delete(`/admin/providers/${id}`);
    return { errCode: 0, errMsg: 'ok' };
  },
};

export async function fetchProviderModels(base_url: string, api_key: string, api_type: string): Promise<{ id: string }[]> {
  const res = await request.post<API.DataSet<{ id: string }>>('/admin/providers/fetch-models', {
    base_url,
    api_key,
    api_type,
  });
  return res.data.dataSet ?? [];
}

export async function batchImportProviderModels(provider_id: number, model_names: string[]): Promise<{ created: number; skipped: number }> {
  const res = await request.post<API.Data<{ created: number; skipped: number }>>(
    '/admin/providers/batch-import-models',
    { provider_id, model_names },
  );
  return res.data.data!;
}
