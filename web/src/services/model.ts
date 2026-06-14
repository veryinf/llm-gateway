import { request } from '@/lib';
import type { API } from '@/typings';
import type { Provider } from './provider';

export interface Model {
  id: number;
  provider_id: number;
  name: string;
  api_type: 'openai' | 'anthropic';
  display_name: string;
  description: string;
  max_context_tokens: number;
  max_output_tokens: number;
  input_price: number;
  output_price: number;
  tpm: number;
  qpm: number;
  is_chat: boolean;
  is_completion: boolean;
  is_vision: boolean;
  is_embedding: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  provider?: Provider;
}

export const modelService: API.Service<Model> = {
  primaryKey: (entity) => entity.id,
  title: (entity) => entity.display_name || entity.name,

  async search(_params) {
    const res = await request.get<API.DataSet<Model>>('/admin/models');
    return res.data;
  },

  async fetch(id) {
    const res = await request.get<API.Data<Model>>('/admin/models');
    const list = (res.data.data as unknown as Model[]) ?? [];
    const model = list.find((m) => m.id === id);
    return { errCode: 0, errMsg: 'ok', data: model };
  },

  async add(params) {
    await request.post('/admin/models', params);
    return { errCode: 0, errMsg: 'ok' };
  },

  async update(id, params) {
    await request.put(`/admin/models/${id}`, params);
    return { errCode: 0, errMsg: 'ok' };
  },

  async delete(id) {
    await request.delete(`/admin/models/${id}`);
    return { errCode: 0, errMsg: 'ok' };
  },
};
