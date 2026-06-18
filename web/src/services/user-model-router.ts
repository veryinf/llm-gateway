import { request } from '@/lib';
import type { API } from '@/typings';

export interface UserModelRouter {
  routerId: number;
  userModelId: number;
  providerModelId: number;
  priority: number;
}

export const userModelRouterService: API.Service<UserModelRouter> = {
  primaryKey: (entity) => entity.routerId,
  title: (entity) => `路由 #${entity.routerId}`,

  async search(params) {
    const res = await request.post<API.DataSet<UserModelRouter>>('/user-model-routers/search', params);
    return res.data;
  },

  async fetch(routerId) {
    const res = await request.post<API.Data<UserModelRouter>>('/user-model-routers/fetch', { routerId });
    return res.data;
  },

  async add(params) {
    const res = await request.post<API.ResponseStruct>('/user-model-routers/add', params);
    return res.data;
  },

  async update(routerId, params) {
    const res = await request.post('/user-model-routers/update', { routerId, ...params });
    return res.data;
  },

  async delete(routerId) {
    const res = await request.post('/user-model-routers/remove', { routerId });
    return res.data;
  },
};
