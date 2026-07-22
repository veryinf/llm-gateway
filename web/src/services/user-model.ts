import { request } from '@/lib';
import type { API } from '@/typings';
import type { ProviderModel } from './provider-model';

export interface UserModel {
  userModelId: number;
  name: string;
  displayName: string;
  description: string;
  isActive: boolean;
  activeProviderModel?: ProviderModel;
}

export const userModelService: API.Service<UserModel> = {
  primaryKey: (entity) => entity.userModelId,
  title: (entity) => entity.displayName || entity.name,

  async search(params) {
    const res = await request.post<API.DataSet<UserModel>>('/user-models/search', params);
    return res.data;
  },

  async fetch(userModelId) {
    const res = await request.post<API.Data<UserModel>>('/user-models/fetch', { userModelId });
    return res.data;
  },

  async add(params) {
    const res = await request.post<API.ResponseStruct>('/user-models/add', params);
    return res.data;
  },

  async update(userModelId, params) {
    const res = await request.post('/user-models/update', { userModelId, ...params });
    return res.data;
  },

  async delete(userModelId) {
    const res = await request.post('/user-models/remove', { userModelId });
    return res.data;
  },
};
