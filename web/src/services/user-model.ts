import { request } from '@/lib';
import type { API } from '@/typings';
import type { ProviderModel } from './provider-model';

export interface UserModel {
  id: number;
  name: string;
  display_name: string;
  upstream_model_id: number;
  description: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  upstream_model?: ProviderModel;
}

export const userModelService: API.Service<UserModel> = {
  primaryKey: (entity) => entity.id,
  title: (entity) => entity.display_name || entity.name,

  async search(_params) {
    const res = await request.get<API.DataSet<UserModel>>('/admin/user-models');
    return res.data;
  },

  async fetch(id) {
    const res = await request.get<API.Data<UserModel>>('/admin/user-models');
    const list = (res.data.data as unknown as UserModel[]) ?? [];
    const item = list.find((m) => m.id === id);
    return { errCode: 0, errMsg: 'ok', data: item };
  },

  async add(params) {
    await request.post('/admin/user-models', params);
    return { errCode: 0, errMsg: 'ok' };
  },

  async update(id, params) {
    await request.put(`/admin/user-models/${id}`, params);
    return { errCode: 0, errMsg: 'ok' };
  },

  async delete(id) {
    await request.delete(`/admin/user-models/${id}`);
    return { errCode: 0, errMsg: 'ok' };
  },
};
