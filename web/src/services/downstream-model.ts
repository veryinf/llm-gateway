import { request } from '@/lib';
import type { API } from '@/typings';
import type { Model } from './model';

export interface DownstreamModel {
  id: number;
  name: string;
  display_name: string;
  upstream_model_id: number;
  description: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  upstream_model?: Model;
}

export const downstreamModelService: API.Service<DownstreamModel> = {
  primaryKey: (entity) => entity.id,
  title: (entity) => entity.display_name || entity.name,

  async search(_params) {
    const res = await request.get<API.DataSet<DownstreamModel>>('/admin/downstream-models');
    return res.data;
  },

  async fetch(id) {
    const res = await request.get<API.Data<DownstreamModel>>('/admin/downstream-models');
    const list = (res.data.data as unknown as DownstreamModel[]) ?? [];
    const item = list.find((m) => m.id === id);
    return { errCode: 0, errMsg: 'ok', data: item };
  },

  async add(params) {
    await request.post('/admin/downstream-models', params);
    return { errCode: 0, errMsg: 'ok' };
  },

  async update(id, params) {
    await request.put(`/admin/downstream-models/${id}`, params);
    return { errCode: 0, errMsg: 'ok' };
  },

  async delete(id) {
    await request.delete(`/admin/downstream-models/${id}`);
    return { errCode: 0, errMsg: 'ok' };
  },
};
