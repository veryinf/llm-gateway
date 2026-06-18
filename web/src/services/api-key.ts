import { request } from '@/lib';
import type { API } from '@/typings';

export interface UserKey {
  keyId: number;
  uid: number;
  key: string;
  title: string;
  isActive: boolean;
}

export const userKeyService: API.Service<UserKey> = {
  primaryKey: (entity) => entity.keyId,
  title: (entity) => entity.title || entity.key,

  async search(params) {
    const res = await request.post<API.DataSet<UserKey>>('/apikey/search', params);
    return res.data;
  },

  async fetch(keyId) {
    const res = await request.post<API.Data<UserKey>>('/apikey/fetch', { keyId });
    return res.data;
  },

  async add(params) {
    const res = await request.post<API.ResponseStruct>('/apikey/add', {
      ...params,
      uid: Number(params.uid),
    });
    return res.data;
  },

  async update(keyId, params) {
    const res = await request.post('/apikey/update', { keyId, ...params });
    return res.data;
  },

  async delete(keyId) {
    const res = await request.post('/apikey/remove', { keyId });
    return res.data;
  },
};
