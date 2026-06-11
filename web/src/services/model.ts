import { request } from '@/lib';
import type { API } from '@/typings';
import type { Provider } from './provider';

export interface Model {
  id: number;
  provider_id: number;
  name: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  provider?: Provider;
}

export const modelService: API.Service<Model> = {
  primaryKey: (entity) => entity.id,
  title: (entity) => entity.name,

  async search() {
    const res = await request.get<API.DataSet<Model>>('/admin/models');
    return res.data;
  },

  async fetch(id) {
    const res = await request.get<API.SingleResponse<Model>>('/admin/models');
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

  async delete(_id) {
    return { errCode: 0, errMsg: 'ok' };
  },
};
