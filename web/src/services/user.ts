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

  async search() {
    const res = await request.get<API.SingleResponse<User[]>>('/admin/users');
    const list = res.data.data ?? [];
    return { list, total: list.length };
  },

  async fetch(id) {
    const res = await request.get<API.SingleResponse<User[]>>('/admin/users');
    const user = (res.data.data ?? []).find((u) => u.id === id);
    return { data: user };
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
