import { request } from '@/lib';
import type { API } from '@/typings';

export interface Provider {
  id: number;
  name: string;
  type: 'openai' | 'azure' | 'anthropic' | 'openai-compatible' | 'ollama';
  base_url: string;
  api_key?: string;
  is_active: boolean;
  priority: number;
  rate_limit_qpm: number;
  rate_limit_burst: number;
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
    const res = await request.get<API.SingleResponse<Provider>>('/admin/providers');
    const list = (res.data.data as unknown as Provider[]) ?? [];
    const provider = list.find((p) => p.id === id);
    return { errCode: 0, errMsg: 'ok', data: provider };
  },

  async add(params) {
    await request.post('/admin/providers', params);
    return { errCode: 0, errMsg: 'ok' };
  },

  async update(id, params) {
    await request.put(`/admin/providers/${id}`, params);
    return { errCode: 0, errMsg: 'ok' };
  },

  async delete(id) {
    await request.delete(`/admin/providers/${id}`);
    return { errCode: 0, errMsg: 'ok' };
  },
};
