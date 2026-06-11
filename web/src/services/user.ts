import { request } from '@/lib';
import type { API } from '@/typings';

export interface User {
  id: number;
  username: string;
  name: string;
  phone: string;
  department: string;
  role: 'admin' | 'user' | 'viewer';
  is_active: boolean;
  api_key_count: number;
  created_at: string;
  updated_at: string;
}

export interface CreateUserParams {
  username: string;
  password: string;
  name?: string;
  phone?: string;
  department?: string;
  role?: 'admin' | 'user' | 'viewer';
}

export const userService: API.Service<User> = {
  primaryKey: (entity) => entity.id,
  title: (entity) => entity.username,

  async search(_params) {
    const res = await request.get<API.DataSet<User>>('/admin/users');
    return res.data;
  },

  async fetch(id) {
    const res = await request.get<API.SingleResponse<User>>('/admin/users');
    const list = (res.data.data as unknown as User[]) ?? [];
    const user = list.find((u) => u.id === id);
    return { errCode: 0, errMsg: 'ok', data: user };
  },

  async add(params) {
    await request.post('/admin/users', params);
    return { errCode: 0, errMsg: 'ok' };
  },

  async update(id, params) {
    await request.put(`/admin/users/${id}`, params);
    return { errCode: 0, errMsg: 'ok' };
  },

  async delete(id) {
    await request.delete(`/admin/users/${id}`);
    return { errCode: 0, errMsg: 'ok' };
  },
};
